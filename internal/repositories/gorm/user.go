package gorm

import (
	"errors"
	"fmt"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *entities.User) (*entities.User, error) {
	if err := r.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByChatID(chatID string) (*entities.User, error) {
	user := &entities.User{}
	if err := r.db.Where("chat_id = ?", chatID).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by chat_id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) ListAll() ([]entities.User, error) {
	var users []entities.User
	if err := r.db.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (r *UserRepository) Update(user *entities.User) error {
	result := r.db.Save(user)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) GetByID(id string) (*entities.User, error) {
	user := &entities.User{}
	if err := r.db.Where("id = ?", id).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by chat_id: %w", err)
	}
	return user, nil
}
