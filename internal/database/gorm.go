package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const _pgDSN = "postgres://%s:%s@%s:%d/%s?sslmode=%v"

func NewGorm(user string, pass string, host string, port int64, name string, sslMode bool) (*gorm.DB, error) {
	sslModeOption := "disable"
	if sslMode {
		sslModeOption = "require"
	}

	dsn := fmt.Sprintf(_pgDSN, user, pass, host, port, name, sslModeOption)

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gorm.Open: %w", err)
	}

	return conn, nil
}
