package gorm

import (
	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"gorm.io/gorm"
)

type TestRepository struct {
	db *gorm.DB
}

func NewTestRepository(db *gorm.DB) *TestRepository {
	return &TestRepository{
		db: db,
	}
}

func (r *TestRepository) GetTest(testType entities.TestType) (*entities.Test, error) {
	var test entities.Test
	if err := r.db.Preload("Questions.Options").
		Where("type = ?", testType).
		First(&test).Error; err != nil {
		return nil, err
	}
	return &test, nil
}

func (r *TestRepository) GetAllTests() ([]entities.Test, error) {
	var tests []entities.Test
	if err := r.db.Preload("Questions.Options").
		Find(&tests).Error; err != nil {
		return nil, err
	}
	return tests, nil
}
