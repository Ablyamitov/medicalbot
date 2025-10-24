package memory

import (
	"fmt"
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
)

type InMemorySessionRepo struct {
	sessions        map[string]*entities.Session
	patientSessions map[string]string // patientID -> sessionID
}

func NewInMemorySessionRepo() *InMemorySessionRepo {
	return &InMemorySessionRepo{
		sessions:        make(map[string]*entities.Session),
		patientSessions: make(map[string]string),
	}
}

func (r *InMemorySessionRepo) CreateSession(patientID string, testType entities.TestType) (*entities.Session, error) {
	sessionID := fmt.Sprintf("session_%s_%s_%d", patientID, testType, time.Now().Unix())
	session := &entities.Session{
		ID:          sessionID,
		PatientID:   patientID,
		TestType:    testType,
		Answers:     []entities.Answer{},
		CurrentStep: 1,
		Status:      "started",
		CreatedAt:   time.Now(),
	}
	r.sessions[sessionID] = session
	r.patientSessions[patientID] = sessionID
	return session, nil
}

func (r *InMemorySessionRepo) GetSession(sessionID string) (*entities.Session, error) {
	session, exists := r.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session with id %s not found", sessionID)
	}
	return session, nil
}

func (r *InMemorySessionRepo) GetSessionByPatient(patientID string) (*entities.Session, error) {
	sessionID, exists := r.patientSessions[patientID]
	if !exists {
		return nil, fmt.Errorf("no session found for patient %s", patientID)
	}
	return r.GetSession(sessionID)
}

func (r *InMemorySessionRepo) UpdateSession(session *entities.Session) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *InMemorySessionRepo) AddAnswer(sessionID string, answer entities.Answer) error {
	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session with id %s not found", sessionID)
	}
	session.Answers = append(session.Answers, answer)
	return nil
}
