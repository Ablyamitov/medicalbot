package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ablyamitov/mamedicalbot/internal/database"
	"github.com/Ablyamitov/mamedicalbot/internal/handler"
	"github.com/Ablyamitov/mamedicalbot/internal/repositories/gorm"
	"github.com/Ablyamitov/mamedicalbot/internal/usecase"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/Ablyamitov/mamedicalbot/internal/config"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

func Run(cfg *config.Config) error {
	sslMode := "disable"
	if cfg.DB.SslMode {
		sslMode = "enable"
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User,
		cfg.DB.Pass,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
		sslMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic("Unable to create database connection: " + err.Error())
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic("Unable to create db driver: " + err.Error())
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		panic("Unable to create migrator: " + err.Error())
	}

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		panic("Unable to perform migration: " + err.Error())
	} else if err != nil {
		slog.Info("Migrate: nothing to do")
	}

	gormConn, err := database.NewGorm(
		cfg.DB.User,
		cfg.DB.Pass,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
		cfg.DB.SslMode,
	)
	if err != nil {
		return fmt.Errorf("db.NewGorm: %w", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Panic("there is no bot token")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	testRepo := gorm.NewTestRepository(gormConn)
	sessionRepo := gorm.NewSessionRepository(gormConn)
	questionRepo := gorm.NewQuestionRepository(gormConn)
	userRepo := gorm.NewUserRepository(gormConn)

	useCase := usecase.NewMedicalBotUseCase(testRepo, sessionRepo, questionRepo, userRepo)
	handl := handler.NewTelegramBotHandler(bot, useCase)

	// Инициализация планировщика cron
	c := cron.New()

	// проверка незавершённых тестов каждые 6 часов
	_, err = c.AddFunc("@every 6h", func() {
		log.Println("⏰ Cron job: checking for incomplete tests...")
		if err := useCase.SendRemindersForIncompleteTests(bot); err != nil {
			log.Println("Reminder job error:", err)
		} else {
			log.Println("Reminder job completed successfully")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start cron job: %w", err)
	}

	c.Start()

	// Telegram обновления в отдельной горутине
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	fmt.Println("🤖 Медицинский Telegram бот запущен...")
	fmt.Println("📱 Найдите бота в Telegram и отправьте /start")

	go func() {
		for update := range updates {
			go handl.HandleUpdate(update)
		}
	}()

	// Ожидаем сигнал завершения
	<-stop
	fmt.Println("🛑 Bot stopped")

	// 🧹 Останавливаем cron при завершении
	c.Stop()

	return nil
}
