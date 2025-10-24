package gorm

import (
	"fmt"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"gorm.io/gorm"
)

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{
		db: db,
	}
}

// GetQuestionByStepAndTestID — поиск вопроса по порядковому номеру вопроса и id теста
func (r *QuestionRepository) GetQuestionByStepAndTestID(step int, testID int64) (*entities.Question, error) {
	var question entities.Question
	if err := r.db.Where("question_no = ?", step).Where("test_id = ?", testID).Preload("Options").First(&question).Error; err != nil {
		return nil, fmt.Errorf("question not found by step = %d and test_id = %d: %w", step, testID, err)
	}
	return &question, nil
}
