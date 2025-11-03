package entities

import "time"

type User struct {
	ID        *string   `json:"id" gorm:"primary_key;type:uuid;default:gen_random_uuid()"`
	ChatID    *string   `json:"chat_id"`
	Username  *string   `json:"username"`
	FirstName *string   `json:"first_name"`
	LastName  *string   `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}
