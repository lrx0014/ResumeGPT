package settings

import (
	"context"
	"errors"
)

var (
	ErrNotFound         = errors.New("settings resource not found")
	ErrInvalid          = errors.New("invalid settings")
	ErrConnectionFailed = errors.New("LLM connection failed")
)

type Repository interface {
	GetPreferences(context.Context, string) (Preferences, error)
	SavePreferences(context.Context, Preferences) (Preferences, error)
	ListAgentDefaults(context.Context, string) ([]AgentDefault, error)
	SaveAgentDefaults(context.Context, string, []AgentDefault) ([]AgentDefault, error)
	ListConnections(context.Context, string) ([]StoredConnection, error)
	GetConnection(context.Context, string, string) (StoredConnection, error)
	CreateConnection(context.Context, StoredConnection) (StoredConnection, error)
	UpdateConnection(context.Context, StoredConnection) (StoredConnection, error)
	DeleteConnection(context.Context, string, string) error
}

type TokenCipher interface {
	Encrypt(string) ([]byte, error)
	Decrypt([]byte) (string, error)
}

type ModelDiscoverer interface {
	Models(context.Context, LLMConnection, string) ([]string, error)
}
