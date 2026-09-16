package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	postgresadapter "github.com/lrx0014/ResumeGPT/internal/adapters/postgres"
	s3adapter "github.com/lrx0014/ResumeGPT/internal/adapters/s3"
	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/config"
	"github.com/lrx0014/ResumeGPT/internal/platform/database"
	"github.com/lrx0014/ResumeGPT/internal/platform/outbox"
	"github.com/lrx0014/ResumeGPT/internal/platform/telemetry"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
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
	go func() {
		if err := dispatcher.Run(ctx); err != nil {
			logger.Error("run outbox dispatcher", "error", err)
			stop()
		}
	}()

	queue := postgresadapter.NewWorkQueue(pool)
	jobRepository := postgresadapter.NewJobRepository(pool)

	blobs, err := s3adapter.NewBlobSigner(s3adapter.Config{
		Endpoint: cfg.ObjectStorageEndpoint, PublicEndpoint: cfg.ObjectStoragePublicEndpoint,
		Bucket: cfg.ObjectStorageBucket, AccessKey: cfg.ObjectStorageAccessKey,
		SecretKey: cfg.ObjectStorageSecretKey, Region: cfg.ObjectStorageRegion,
	})
	if err != nil {
		logger.Error("initialize object storage", "error", err)
		os.Exit(1)
	}
	extractor, err := document.NewHTTPExtractor(cfg.DocumentWorkerURL)
	if err != nil {
		logger.Error("initialize document extractor", "error", err)
		os.Exit(1)
	}
	documentProcessor := document.Processor{
		Queue: queue, Repository: postgresadapter.NewDocumentRepository(pool), Blobs: blobs,
		Extractor: extractor, WorkerID: id.New("document_worker"), Logger: logger,
	}
	go func() {
		if err := documentProcessor.Run(ctx); err != nil {
			logger.Error("run document processor", "error", err)
			stop()
		}
	}()
	templateProcessor := resumetemplate.Processor{Queue: queue, Repository: postgresadapter.NewTemplateRepository(pool), Blobs: blobs,
		Extractor: extractor, Previewer: extractor, WorkerID: id.New("template_worker"), Logger: logger}
	go func() {
		if err := templateProcessor.Run(ctx); err != nil {
			logger.Error("run template processor", "error", err)
			stop()
		}
	}()
	tokenCipher, err := settings.NewAESGCMTokenCipher(cfg.SettingsEncryptionKey)
	if err != nil {
		logger.Error("initialize settings encryption", "error", err)
		os.Exit(1)
	}
	settingsService := settings.NewService(postgresadapter.NewSettingsRepository(pool), tokenCipher, settings.NewHTTPModelDiscoverer())
	gateway := generation.NewHTTPGateway()
	pageBrowser, err := job.NewHTTPPageBrowser(cfg.WebWorkerURL)
	if err != nil {
		logger.Error("initialize AI job browser", "error", err)
		os.Exit(1)
	}
	jobProcessor := job.ImportProcessor{
		Queue: queue, Repository: jobRepository, Fetcher: job.NewHTTPFetcher(),
		AgentFetcher: job.NewJobImportAgent(settingsService, gateway, pageBrowser),
		WorkerID:     id.New("job_import_worker"), Logger: logger,
	}
	go func() {
		if err := jobProcessor.Run(ctx); err != nil {
			logger.Error("run job import processor", "error", err)
			stop()
		}
	}()
	generationProcessor := generation.Processor{Queue: queue, Repository: postgresadapter.NewGenerationRepository(pool), Settings: settingsService, Blobs: blobs, Documents: extractor, Gateway: gateway, WorkerID: id.New("generation_worker"), Logger: logger}
	if err := generationProcessor.Run(ctx); err != nil {
		logger.Error("run generation processor", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}
