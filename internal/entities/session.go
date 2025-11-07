package entities

import "time"

type Session struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	TestID               int64     `json:"test_id"`
	CurrentStep          int       `json:"current_step"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	Answers              []Answer  `json:"answers"`
	TestType             TestType  `json:"test_type"`
	IsConsultationNeeded bool      `json:"is_consultation_needed"`
}
