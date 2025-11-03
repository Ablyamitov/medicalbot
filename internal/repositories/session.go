package repositories

import (
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
)

type SessionRepository interface {
	CreateSession(userID string, testType entities.TestType) (*entities.Session, error)
	GetSession(sessionID string) (*entities.Session, error)
	GetSessionByUser(userID string) (*entities.Session, error)
	UpdateSession(session *entities.Session) error
	AddAnswer(sessionID string, answer entities.Answer) error
	GetStaleSessions(since time.Duration) ([]entities.Session, error)
	GetActiveSessionByUserAndType(userID string, testType entities.TestType, status string) (*entities.Session, error)
	GetUsersWithUnfinishedTests() ([]string, error)
	GetUnfinishedSessions(userID string) ([]*entities.Session, error)
	HasRecentlyCompletedTest(patientID string, since time.Duration) (bool, error)
	IsLastTestCompleted(userID string) (bool, error)
	GetStatistics() ([]entities.SessionStatistic, error)
	SaveTestResult(session *entities.Session, score int, interpretation string, recommendations []string) error
}
