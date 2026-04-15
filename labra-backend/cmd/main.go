package main

import (
	"database/sql"
	"log"
	"log/slog"
	"os"

	"labra-backend/internal/api/config"
	"labra-backend/internal/api/handlers"
	"labra-backend/internal/api/middleware"
	"labra-backend/internal/api/routes"
	"labra-backend/internal/api/services"

	"github.com/go-fuego/fuego"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lpernett/godotenv"
)

func main() {
	_ = godotenv.Load("../.env", ".env")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	if cfg.GHClientID != "" && cfg.GHClientSecret != "" {
		services.InitOauth(cfg.GHClientID, cfg.GHClientSecret)
	} else {
		logger.Warn("GitHub OAuth is not configured; /v1/login and /v1/callback will not work")
	}

	db, err := sql.Open("sqlite3", cfg.DBURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	handlers.InitAppStore(db)
	handlers.InitWebhook(cfg.GitHubWebhookSecret)
	handlers.InitReadiness(db.PingContext)

	s := fuego.NewServer(
		fuego.WithAddr(cfg.ListenAddress()),
	)
	fuego.Use(s, middleware.RequestContext(logger))

	routes.HealthRoute(s)
	routes.Oauth(s)
	routes.AWSConnections(s)
	routes.Apps(s)
	routes.Deploy(s)
	routes.Webhooks(s)

	logger.Info("server starting", "addr", cfg.ListenAddress(), "env", cfg.Environment)
	s.Run()
}

func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../sql/migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
