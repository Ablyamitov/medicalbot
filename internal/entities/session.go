package entities

import "time"

type Session struct {
	ID          string    `json:"id"`
	PatientID   string    `json:"patient_id"`
	TestID      int64     `json:"test_id"`
	CurrentStep int       `json:"current_step"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Answers     []Answer  `json:"answers"`
	TestType    TestType  `json:"test_type"`
}
