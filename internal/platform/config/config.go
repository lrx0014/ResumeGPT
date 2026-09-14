package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment                 string
	PersistenceMode             string
	AuthMode                    string
	DevelopmentWorkspaceID      string
	DatabaseURL                 string
	OIDCIssuer                  string
	OIDCClientID                string
	ObjectStorageMode           string
	ObjectStorageEndpoint       string
	ObjectStoragePublicEndpoint string
	ObjectStorageBucket         string
	ObjectStorageAccessKey      string
	ObjectStorageSecretKey      string
	ObjectStorageRegion         string
	DocumentWorkerURL           string
	APIAddress                  string
	ReadTimeout                 time.Duration
	WriteTimeout                time.Duration
	ShutdownTimeout             time.Duration
	WebOrigin                   string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:                 getEnv("APP_ENV", "development"),
		PersistenceMode:             getEnv("PERSISTENCE_MODE", "memory"),
		AuthMode:                    getEnv("AUTH_MODE", "development"),
		DevelopmentWorkspaceID:      getEnv("DEVELOPMENT_WORKSPACE_ID", "ws_personal_dev"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://resumegpt:resumegpt@localhost:5432/resumegpt?sslmode=disable"),
		OIDCIssuer:                  getEnv("OIDC_ISSUER", ""),
		OIDCClientID:                getEnv("OIDC_CLIENT_ID", ""),
		ObjectStorageMode:           getEnv("OBJECT_STORAGE_MODE", "memory"),
		ObjectStorageEndpoint:       getEnv("OBJECT_STORAGE_ENDPOINT", "http://localhost:9000"),
		ObjectStoragePublicEndpoint: getEnv("OBJECT_STORAGE_PUBLIC_ENDPOINT", getEnv("OBJECT_STORAGE_ENDPOINT", "http://localhost:9000")),
		ObjectStorageBucket:         getEnv("OBJECT_STORAGE_BUCKET", "resumegpt"),
		ObjectStorageAccessKey:      getEnv("OBJECT_STORAGE_ACCESS_KEY", "resumegpt"),
		ObjectStorageSecretKey:      getEnv("OBJECT_STORAGE_SECRET_KEY", "change-me"),
		ObjectStorageRegion:         getEnv("OBJECT_STORAGE_REGION", "us-east-1"),
		DocumentWorkerURL:           getEnv("DOCUMENT_WORKER_URL", "http://localhost:8090"),
		APIAddress:                  getEnv("API_ADDRESS", ":8080"),
		WebOrigin:                   getEnv("WEB_ORIGIN", "http://localhost:5173"),
	}

	var err error
	if cfg.ReadTimeout, err = getDuration("API_READ_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = getDuration("API_WRITE_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = getDuration("API_SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
