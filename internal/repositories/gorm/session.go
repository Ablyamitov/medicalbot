package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

// CreateSession — создаёт новую сессию
func (r *SessionRepository) CreateSession(userID string, testType entities.TestType) (*entities.Session, error) {
	// сначала найдём тест по типу
	var test entities.Test
	if err := r.db.Where("type = ?", testType).First(&test).Error; err != nil {
		return nil, fmt.Errorf("test not found for type %s: %w", testType, err)
	}

	session := &entities.Session{
		ID:          uuid.New().String(),
		UserID:      userID,
		TestID:      test.ID,
		CurrentStep: 1,
		Status:      "started",
		CreatedAt:   time.Now(),
		TestType:    testType,
	}

	if err := r.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession — получить сессию по ID
func (r *SessionRepository) GetSession(sessionID string) (*entities.Session, error) {
	var session entities.Session
	if err := r.db.
		Preload("Answers").
		First(&session, "id = ?", sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session with id %s not found", sessionID)
		}
		return nil, err
	}
	return &session, nil
}

// GetSessionByUser — найти активную сессию пользователя
func (r *SessionRepository) GetSessionByUser(userID string) (*entities.Session, error) {
	var session entities.Session
	if err := r.db.
		//Where("patient_id = ? AND status IN (?)", patientID, []string{"started", "in_progress"}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Preload("Answers").
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no session found for user %s", userID)
		}
		return nil, err
	}
	return &session, nil
}

// UpdateSession — обновляет сессию
func (r *SessionRepository) UpdateSession(session *entities.Session) error {
	if err := r.db.Save(session).Error; err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// AddAnswer — добавить ответ к сессии
func (r *SessionRepository) AddAnswer(sessionID string, answer entities.Answer) error {
	answer.ID = uuid.New().String()
	answer.SessionID = sessionID

	if err := r.db.Create(&answer).Error; err != nil {
		return fmt.Errorf("failed to add answer: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetStaleSessions(since time.Duration) ([]entities.Session, error) {
	var sessions []entities.Session
	cutoff := time.Now().Add(-since)

	if err := r.db.
		Where("status IN ?", []string{"started", "in_progress"}).
		Where("updated_at < ?", cutoff).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to get stale sessions: %w", err)
	}

	return sessions, nil
}

func (r *SessionRepository) GetActiveSessionByUserAndType(userID string, testType entities.TestType, status string) (*entities.Session, error) {
	var session entities.Session
	err := r.db.
		Where("user_id = ? AND test_type = ? AND status = ?", userID, testType, status).
		Order("created_at DESC").
		First(&session).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) GetUsersWithUnfinishedTests() ([]string, error) {
	var patientIDs []string
	err := r.db.
		Model(&entities.Session{}).
		Where("status IN ?", []string{"started", "in_progress"}).
		Distinct().
		Pluck("user_id", &patientIDs).Error
	return patientIDs, err
}

func (r *SessionRepository) GetUnfinishedSessions(userID string) ([]*entities.Session, error) {
	var sessions []*entities.Session
	err := r.db.
		Where("user_id = ? AND status IN ?", userID, []string{"started", "in_progress"}).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *SessionRepository) HasRecentlyCompletedTest(patientID string, since time.Duration) (bool, error) {
	var count int64
	err := r.db.
		Model(&entities.Session{}).
		Where("patient_id = ? AND status = ? AND updated_at > ?", patientID, "completed", time.Now().Add(-since)).
		Count(&count).Error
	return count > 0, err
}

func (r *SessionRepository) IsLastTestCompleted(userID string) (bool, error) {
	var lastSession entities.Session
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&lastSession).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return lastSession.Status == "completed", nil
}

func (r *SessionRepository) GetStatistics() ([]entities.SessionStatistic, error) {
	const query = `
SELECT
    s.id AS session_id,
    u.chat_id,
    u.first_name,
    u.last_name,
    u.username,
    s.test_type,
    s.status,
    s.created_at,
    s.updated_at,
    COUNT(a.id) AS answered_questions,
    tr.score,
    tr.interpretation,
    tr.recommendations
FROM sessions s
LEFT JOIN users u ON u.id = s.user_id
LEFT JOIN answers a ON a.session_id = s.id
LEFT JOIN test_results tr ON tr.session_id = s.id
GROUP BY s.id, u.id, u.chat_id, u.first_name, u.last_name, u.username, tr.score, tr.interpretation, tr.recommendations
ORDER BY s.created_at DESC;

`

	var stats []entities.SessionStatistic
	if err := r.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}
func (r *SessionRepository) SaveTestResult(session *entities.Session, score int, interpretation string, recommendations []string) error {
	const query = `
    INSERT INTO test_results(session_id, score, interpretation, recommendations)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT ON CONSTRAINT test_results_session_id_unique DO UPDATE
    SET score = EXCLUDED.score,
        interpretation = EXCLUDED.interpretation,
        recommendations = EXCLUDED.recommendations,
        created_at = now();
    `
	return r.db.Exec(query, session.ID, score, interpretation, pq.Array(recommendations)).Error
}
