package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	postgresadapter "github.com/lrx0014/ResumeGPT/internal/adapters/postgres"
	s3adapter "github.com/lrx0014/ResumeGPT/internal/adapters/s3"
	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/config"
	"github.com/lrx0014/ResumeGPT/internal/platform/database"
	"github.com/lrx0014/ResumeGPT/internal/platform/telemetry"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/taskmonitor"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
	"github.com/lrx0014/ResumeGPT/internal/transport/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	shutdownTelemetry, err := telemetry.Setup(context.Background(), "resumegpt-api")
	if err != nil {
		logger.Error("initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			logger.Error("shut down telemetry", "error", err)
		}
	}()

	var profileRepository profile.Repository
	var jobRepository job.Repository
	var accessRepository identity.AccessRepository
	var documentRepository document.Repository
	var documentService *document.Service
	var jobImportService *job.ImportService
	var settingsRepository settings.Repository
	var templateRepository resumetemplate.Repository
	var generationRepository generation.Repository
	var taskMonitorService *taskmonitor.Service
	switch cfg.PersistenceMode {
	case "memory":
		profileRepository = memory.NewProfileRepository()
		jobRepository = memory.NewJobRepository()
		settingsRepository = memory.NewSettingsRepository()
		templateRepository = memory.NewTemplateRepository()
		taskMonitorService = taskmonitor.NewService(memory.NewTaskMonitorRepository())
		accessRepository = memory.AccessRepository{
			Subject: identity.Subject{Issuer: "development", Subject: "developer", Email: "developer@localhost"},
			Principal: identity.Principal{
				UserID: "usr_dev", WorkspaceID: cfg.DevelopmentWorkspaceID, Role: identity.RoleOwner,
			},
		}
	case "postgres":
		pool, err := database.Open(context.Background(), cfg.DatabaseURL)
		if err != nil {
			logger.Error("connect to database", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		profileRepository = postgresadapter.NewProfileRepository(pool)
		postgresJobRepository := postgresadapter.NewJobRepository(pool)
		jobRepository = postgresJobRepository
		jobImportService = job.NewImportService(postgresJobRepository)
		accessRepository = postgresadapter.NewAccessRepository(pool)
		documentRepository = postgresadapter.NewDocumentRepository(pool)
		settingsRepository = postgresadapter.NewSettingsRepository(pool)
		templateRepository = postgresadapter.NewTemplateRepository(pool)
		generationRepository = postgresadapter.NewGenerationRepository(pool)
		taskMonitorService = taskmonitor.NewService(postgresadapter.NewTaskMonitorRepository(pool))
	default:
		logger.Error("unsupported persistence mode", "mode", cfg.PersistenceMode)
		os.Exit(1)
	}
	profileService := profile.NewService(profileRepository)
	jobService := job.NewService(jobRepository)
	tokenCipher, err := settings.NewAESGCMTokenCipher(cfg.SettingsEncryptionKey)
	if err != nil {
		logger.Error("initialize settings encryption", "error", err)
		os.Exit(1)
	}
	settingsService := settings.NewService(settingsRepository, tokenCipher, settings.NewHTTPModelDiscoverer())
	var blobSigner blobstore.Signer
	switch cfg.ObjectStorageMode {
	case "memory":
		blobSigner = memory.BlobSigner{}
	case "s3":
		blobSigner, err = s3adapter.NewBlobSigner(s3adapter.Config{
			Endpoint:       cfg.ObjectStorageEndpoint,
			PublicEndpoint: cfg.ObjectStoragePublicEndpoint,
			Bucket:         cfg.ObjectStorageBucket,
			AccessKey:      cfg.ObjectStorageAccessKey,
			SecretKey:      cfg.ObjectStorageSecretKey,
			Region:         cfg.ObjectStorageRegion,
		})
		if err != nil {
			logger.Error("initialize object storage", "error", err)
			os.Exit(1)
		}
		if err := blobSigner.EnsureBucket(context.Background()); err != nil {
			logger.Error("ensure object storage bucket", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("unsupported object storage mode", "mode", cfg.ObjectStorageMode)
		os.Exit(1)
	}
	if _, ok := blobSigner.(blobstore.Reader); documentRepository != nil && ok {
		documentService = document.NewService(documentRepository, blobSigner)
	}
	templateService := resumetemplate.NewService(templateRepository, blobSigner, documentService != nil)
	if blobs, ok := blobSigner.(resumetemplate.TemplateBlobs); ok {
		previewer, previewErr := document.NewHTTPExtractor(cfg.DocumentWorkerURL)
		if previewErr != nil {
			logger.Error("initialize template previewer", "error", previewErr)
			os.Exit(1)
		}
		previewObjectID, previewErr := resumetemplate.PrepareDefaultPreview(context.Background(), blobs, previewer)
		if previewErr != nil {
			logger.Error("prepare built-in template preview", "error", previewErr)
			os.Exit(1)
		}
		templateService.ConfigurePreview(blobs, previewObjectID)
	}
	var generationService *generation.Service
	if reader, ok := blobSigner.(blobstore.Reader); ok && generationRepository != nil {
		generationService = generation.NewService(generationRepository, profileService, jobService, templateService, settingsService, reader)
	}
	var authenticator identity.Authenticator
	switch cfg.AuthMode {
	case "development":
		authenticator = identity.DevelopmentAuthenticator{
			Subject: identity.Subject{Issuer: "development", Subject: "developer", Email: "developer@localhost"},
		}
	case "oidc":
		if cfg.OIDCIssuer == "" || cfg.OIDCClientID == "" {
			logger.Error("OIDC issuer and client ID are required")
			os.Exit(1)
		}
		authenticator, err = identity.NewOIDCAuthenticator(context.Background(), cfg.OIDCIssuer, cfg.OIDCClientID)
		if err != nil {
			logger.Error("initialize OIDC", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("unsupported authentication mode", "mode", cfg.AuthMode)
		os.Exit(1)
	}
	handler := httpapi.New(httpapi.Dependencies{
		Profiles:           profileService,
		Jobs:               jobService,
		JobImports:         jobImportService,
		Logger:             logger,
		WebOrigin:          cfg.WebOrigin,
		Authenticator:      authenticator,
		Access:             accessRepository,
		DefaultWorkspaceID: developmentWorkspace(cfg),
		Blobs:              blobSigner,
		Documents:          documentService,
		Settings:           settingsService,
		Templates:          templateService,
		Generations:        generationService,
		Tasks:              taskMonitorService,
	})

	server := &http.Server{
		Addr:         cfg.APIAddress,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdownSignals
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("shut down API", "error", err)
		}
	}()

	logger.Info("API starting", "address", cfg.APIAddress, "environment", cfg.Environment, "persistence", cfg.PersistenceMode)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("serve API", "error", err)
		os.Exit(1)
	}
}

func developmentWorkspace(cfg config.Config) string {
	if cfg.AuthMode == "development" {
		return cfg.DevelopmentWorkspaceID
	}
	return ""
}
