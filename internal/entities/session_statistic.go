package entities

import (
	"time"

	"github.com/lib/pq"
)

type SessionStatistic struct {
	SessionID         string         `json:"session_id"`
	UserID            string         `json:"user_id"`
	ChatID            string         `json:"chat_id"`
	FirstName         string         `json:"first_name"`
	LastName          string         `json:"last_name"`
	Username          string         `json:"username"`
	TestType          string         `json:"test_type"`
	Status            string         `json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	AnsweredQuestions int            `json:"answered_questions"`
	Score             int            `json:"score"`
	Interpretation    string         `json:"interpretation"`
	Recommendations   pq.StringArray `gorm:"column:recommendations;type:text[]"`
}
