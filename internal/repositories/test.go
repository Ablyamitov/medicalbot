package repositories

import "github.com/Ablyamitov/mamedicalbot/internal/entities"

type TestRepository interface {
	GetTest(testType entities.TestType) (*entities.Test, error)
	GetAllTests() ([]entities.Test, error)
}
