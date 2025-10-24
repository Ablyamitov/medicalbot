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
