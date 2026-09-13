package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	postgresadapter "github.com/lrx0014/ResumeGPT/internal/adapters/postgres"
	"github.com/lrx0014/ResumeGPT/internal/platform/config"
	"github.com/lrx0014/ResumeGPT/internal/platform/database"
	"github.com/lrx0014/ResumeGPT/internal/platform/outbox"
	"github.com/lrx0014/ResumeGPT/internal/platform/telemetry"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	shutdownTelemetry, err := telemetry.Setup(context.Background(), "resumegpt-worker")
	if err != nil {
		logger.Error("initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Error("shut down telemetry", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("worker starting", "environment", cfg.Environment, "persistence", cfg.PersistenceMode)
	if cfg.PersistenceMode == "memory" {
		<-ctx.Done()
		logger.Info("worker stopped")
		return
	}
	if cfg.PersistenceMode != "postgres" {
		logger.Error("unsupported persistence mode", "mode", cfg.PersistenceMode)
		os.Exit(1)
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	dispatcher := outbox.Dispatcher{
		Repository: postgresadapter.NewOutboxRepository(pool),
		Publisher:  outbox.LogPublisher{Logger: logger},
		WorkerID:   id.New("worker"),
		Logger:     logger,
	}
	if err := dispatcher.Run(ctx); err != nil {
		logger.Error("run outbox dispatcher", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}
