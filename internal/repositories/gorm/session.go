package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"github.com/google/uuid"
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
func (r *SessionRepository) CreateSession(patientID string, testType entities.TestType) (*entities.Session, error) {
	// сначала найдём тест по типу
	var test entities.Test
	if err := r.db.Where("type = ?", testType).First(&test).Error; err != nil {
		return nil, fmt.Errorf("test not found for type %s: %w", testType, err)
	}

	session := &entities.Session{
		ID:          uuid.New().String(),
		PatientID:   patientID,
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

// GetSessionByPatient — найти активную сессию пациента
func (r *SessionRepository) GetSessionByPatient(patientID string) (*entities.Session, error) {
	var session entities.Session
	if err := r.db.
		//Where("patient_id = ? AND status IN (?)", patientID, []string{"started", "in_progress"}).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Preload("Answers").
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no session found for patient %s", patientID)
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

func (r *SessionRepository) GetActiveSessionByPatientAndType(patientID string, testType entities.TestType) (*entities.Session, error) {
	var session entities.Session
	err := r.db.
		Where("patient_id = ? AND test_type = ? AND status = ?", patientID, testType, "started").
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

func (r *SessionRepository) GetPatientsWithUnfinishedTests() ([]string, error) {
	var patientIDs []string
	err := r.db.
		Model(&entities.Session{}).
		Where("status IN ?", []string{"started", "in_progress"}).
		Distinct().
		Pluck("patient_id", &patientIDs).Error
	return patientIDs, err
}

func (r *SessionRepository) GetUnfinishedSessions(patientID string) ([]*entities.Session, error) {
	var sessions []*entities.Session
	err := r.db.
		Where("patient_id = ? AND status IN ?", patientID, []string{"started", "in_progress"}).
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

func (r *SessionRepository) IsLastTestCompleted(patientID string) (bool, error) {
	var lastSession entities.Session
	err := r.db.
		Where("patient_id = ?", patientID).
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
