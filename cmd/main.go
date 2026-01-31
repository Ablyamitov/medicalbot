package main

import (
	"log"

	"github.com/Ablyamitov/mamedicalbot/internal/app"
	"github.com/Ablyamitov/mamedicalbot/internal/config"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/github"
)

// main
func main() {
	cfg := config.MustLoad()

	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
