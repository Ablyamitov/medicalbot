package entities

type Question struct {
	ID         int64          `json:"id"`
	TestID     int64          `json:"test_id"`
	QuestionNo int            `json:"question_no"`
	Text       string         `json:"text"`
	Options    []AnswerOption `json:"options"`
}
