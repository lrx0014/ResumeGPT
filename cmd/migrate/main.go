package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/lrx0014/ResumeGPT/internal/platform/config"
	"github.com/lrx0014/ResumeGPT/internal/platform/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	pool, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := database.Migrate(context.Background(), pool); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("database migrations completed")
}
