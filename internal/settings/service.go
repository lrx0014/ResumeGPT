package settings

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

type Service struct {
	repository Repository
	cipher     TokenCipher
	discoverer ModelDiscoverer
	now        func() time.Time
}

func NewService(repository Repository, cipher TokenCipher, discoverer ModelDiscoverer) *Service {
	return &Service{repository: repository, cipher: cipher, discoverer: discoverer, now: time.Now}
}

func (s *Service) GetPreferences(ctx context.Context, workspaceID string) (Preferences, error) {
	return s.repository.GetPreferences(ctx, workspaceID)
}

func (s *Service) SavePreferences(ctx context.Context, workspaceID string, input PreferencesInput) (Preferences, error) {
	language, theme := strings.TrimSpace(input.InterfaceLanguage), strings.TrimSpace(input.Theme)
	if language != "en" && language != "de" || theme != "system" && theme != "light" && theme != "dark" {
		return Preferences{}, ErrInvalid
	}
	return s.repository.SavePreferences(ctx, Preferences{WorkspaceID: workspaceID, InterfaceLanguage: language,
		Theme: theme, UpdatedAt: s.now().UTC()})
}

func (s *Service) ListConnections(ctx context.Context, workspaceID string) ([]LLMConnection, error) {
	stored, err := s.repository.ListConnections(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]LLMConnection, 0, len(stored))
	for _, item := range stored {
		result = append(result, publicConnection(item))
	}
	return result, nil
}

func (s *Service) CreateConnection(ctx context.Context, workspaceID string, input LLMConnectionInput) (LLMConnection, error) {
	connection, err := prepareConnection(input)
	if err != nil {
		return LLMConnection{}, err
	}
	ciphertext, err := s.cipher.Encrypt(strings.TrimSpace(input.APIToken))
	if err != nil {
		return LLMConnection{}, err
	}
	now := s.now().UTC()
	connection.ID, connection.WorkspaceID = id.New("llm"), workspaceID
	connection.CreatedAt, connection.UpdatedAt = now, now
	stored, err := s.repository.CreateConnection(ctx, StoredConnection{Connection: connection, TokenCiphertext: ciphertext})
	return publicConnection(stored), err
}

func (s *Service) UpdateConnection(ctx context.Context, workspaceID, connectionID string, input LLMConnectionInput) (LLMConnection, error) {
	connection, err := prepareConnection(input)
	if err != nil {
		return LLMConnection{}, err
	}
	current, err := s.repository.GetConnection(ctx, workspaceID, connectionID)
	if err != nil {
		return LLMConnection{}, err
	}
	ciphertext := current.TokenCiphertext
	if input.ClearAPIToken {
		ciphertext = nil
	}
	if token := strings.TrimSpace(input.APIToken); token != "" {
		ciphertext, err = s.cipher.Encrypt(token)
		if err != nil {
			return LLMConnection{}, err
		}
	}
	connection.ID, connection.WorkspaceID = connectionID, workspaceID
	connection.CreatedAt, connection.UpdatedAt = current.Connection.CreatedAt, s.now().UTC()
	stored, err := s.repository.UpdateConnection(ctx, StoredConnection{Connection: connection, TokenCiphertext: ciphertext})
	return publicConnection(stored), err
}

func (s *Service) DeleteConnection(ctx context.Context, workspaceID, connectionID string) error {
	return s.repository.DeleteConnection(ctx, workspaceID, connectionID)
}

func (s *Service) TestConnection(ctx context.Context, workspaceID, connectionID string) (ConnectionTest, error) {
	stored, err := s.repository.GetConnection(ctx, workspaceID, connectionID)
	if err != nil {
		return ConnectionTest{}, err
	}
	token, err := s.cipher.Decrypt(stored.TokenCiphertext)
	if err != nil {
		return ConnectionTest{}, err
	}
	models, err := s.discoverer.Models(ctx, stored.Connection, token)
	if err != nil {
		return ConnectionTest{}, err
	}
	return ConnectionTest{Status: "connected", Models: models}, nil
}

func (s *Service) RuntimeConnection(ctx context.Context, workspaceID, connectionID string) (RuntimeConnection, error) {
	stored, err := s.repository.GetConnection(ctx, workspaceID, connectionID)
	if err != nil {
		return RuntimeConnection{}, err
	}
	token, err := s.cipher.Decrypt(stored.TokenCiphertext)
	if err != nil {
		return RuntimeConnection{}, err
	}
	return RuntimeConnection{Connection: stored.Connection, APIToken: token}, nil
}

func prepareConnection(input LLMConnectionInput) (LLMConnection, error) {
	input.Name, input.ExecutionMode, input.Provider = strings.TrimSpace(input.Name), strings.TrimSpace(input.ExecutionMode), strings.TrimSpace(input.Provider)
	input.BaseURL = strings.TrimSpace(input.BaseURL)
	if !validSettingText(input.Name, 120, true) || len(input.APIToken) > 8192 {
		return LLMConnection{}, ErrInvalid
	}
	if input.ExecutionMode != "cloud" && input.ExecutionMode != "local" {
		return LLMConnection{}, ErrInvalid
	}
	if input.Provider != "openai" && input.Provider != "openai_compatible" && input.Provider != "ollama" {
		return LLMConnection{}, ErrInvalid
	}
	if input.Provider == "openai" && input.ExecutionMode != "cloud" || input.Provider == "ollama" && input.ExecutionMode != "local" {
		return LLMConnection{}, ErrInvalid
	}
	if input.BaseURL == "" {
		if input.Provider == "openai" {
			input.BaseURL = "https://api.openai.com/v1"
		} else if input.Provider == "ollama" {
			input.BaseURL = "http://host.docker.internal:11434"
		}
	}
	parsed, err := url.Parse(input.BaseURL)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || input.ExecutionMode == "cloud" && parsed.Scheme != "https" || len(input.BaseURL) > 2048 {
		return LLMConnection{}, ErrInvalid
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return LLMConnection{Name: input.Name, ExecutionMode: input.ExecutionMode, Provider: input.Provider,
		BaseURL: parsed.String()}, nil
}

func validSettingText(value string, limit int, required bool) bool {
	if required && value == "" || len(value) > limit || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func publicConnection(stored StoredConnection) LLMConnection {
	result := stored.Connection
	result.APITokenConfigured = len(stored.TokenCiphertext) > 0
	return result
}
