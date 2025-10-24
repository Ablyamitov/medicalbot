package entities

type Answer struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	QuestionID int64  `json:"question_id"`
	Value      int    `json:"value"`
}

type AnswerOption struct {
	ID         string `json:"id"`
	QuestionID int64  `json:"question_id"`
	Value      int    `json:"value"`
	Text       string `json:"text"`
}
