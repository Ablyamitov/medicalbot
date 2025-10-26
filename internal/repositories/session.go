package repositories

import (
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
)

type SessionRepository interface {
	CreateSession(patientID string, testType entities.TestType) (*entities.Session, error)
	GetSession(sessionID string) (*entities.Session, error)
	GetSessionByPatient(patientID string) (*entities.Session, error)
	UpdateSession(session *entities.Session) error
	AddAnswer(sessionID string, answer entities.Answer) error
	GetStaleSessions(since time.Duration) ([]entities.Session, error)
	GetActiveSessionByPatientAndType(patientID string, testType entities.TestType) (*entities.Session, error)
	GetPatientsWithUnfinishedTests() ([]string, error)
	GetUnfinishedSessions(patientID string) ([]*entities.Session, error)
	HasRecentlyCompletedTest(patientID string, since time.Duration) (bool, error)
	IsLastTestCompleted(patientID string) (bool, error)
}
