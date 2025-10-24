package repositories

import "github.com/Ablyamitov/mamedicalbot/internal/entities"

type SessionRepository interface {
	CreateSession(patientID string, testType entities.TestType) (*entities.Session, error)
	GetSession(sessionID string) (*entities.Session, error)
	GetSessionByPatient(patientID string) (*entities.Session, error)
	UpdateSession(session *entities.Session) error
	AddAnswer(sessionID string, answer entities.Answer) error
}
