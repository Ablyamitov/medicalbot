package repositories

import "github.com/Ablyamitov/mamedicalbot/internal/entities"

type QuestionRepository interface {
	GetQuestionByStepAndTestID(step int, testID int64) (*entities.Question, error)
}
