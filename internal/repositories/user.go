package repositories

import "github.com/Ablyamitov/mamedicalbot/internal/entities"

type UserRepository interface {
	Create(user *entities.User) (*entities.User, error)
	GetByChatID(chatID string) (*entities.User, error)
	ListAll() ([]entities.User, error)
	Update(user *entities.User) error
	GetByID(id string) (*entities.User, error)
}
