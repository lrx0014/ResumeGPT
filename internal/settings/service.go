package settings

import (
	"context"
	"errors"
	"fmt"
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
	modelCache ModelCache
	cacheTTL   time.Duration
	now        func() time.Time
}

func NewService(repository Repository, cipher TokenCipher, discoverer ModelDiscoverer) *Service {
	return &Service{repository: repository, cipher: cipher, discoverer: discoverer, now: time.Now}
}

func (s *Service) ConfigureModelCache(cache ModelCache, ttl time.Duration) {
	s.modelCache, s.cacheTTL = cache, ttl
}

func (s *Service) GetPreferences(ctx context.Context, workspaceID string) (Preferences, error) {
	return s.repository.GetPreferences(ctx, workspaceID)
}

func (s *Service) SavePreferences(ctx context.Context, workspaceID string, input PreferencesInput) (Preferences, error) {
	language, theme := strings.TrimSpace(input.InterfaceLanguage), strings.TrimSpace(input.Theme)
	validLanguages := map[string]bool{"en": true, "de": true, "fr": true, "es": true, "ja": true, "zh-CN": true, "zh-TW": true}
	if !validLanguages[language] || theme != "system" && theme != "light" && theme != "dark" {
		return Preferences{}, ErrInvalid
	}
	return s.repository.SavePreferences(ctx, Preferences{WorkspaceID: workspaceID, InterfaceLanguage: language,
		Theme: theme, UpdatedAt: s.now().UTC()})
}

func (s *Service) ListAgentDefaults(ctx context.Context, workspaceID string) ([]AgentDefault, error) {
	return s.repository.ListAgentDefaults(ctx, workspaceID)
}

func (s *Service) SaveAgentDefaults(ctx context.Context, workspaceID string, input AgentDefaultsInput) ([]AgentDefault, error) {
	validAgents := map[string]bool{
		AgentWriter: true, AgentTemplateApplier: true, AgentDocumentDesigner: true, AgentVisualReviewer: true,
		AgentJobImport: true, AgentJobHunter: true,
	}
	seen := make(map[string]bool)
	values := make([]AgentDefault, 0, len(input.Items))
	now := s.now().UTC()
	for _, item := range input.Items {
		item.Agent = strings.TrimSpace(item.Agent)
		item.ConnectionID = strings.TrimSpace(item.ConnectionID)
		item.Model = strings.TrimSpace(item.Model)
		if !validAgents[item.Agent] || seen[item.Agent] || !validSettingText(item.ConnectionID, 200, true) || !validSettingText(item.Model, 200, true) || item.MaxTokens < 0 || item.MaxTokens > MaxTokensCeiling {
			return nil, ErrInvalid
		}
		if _, err := s.repository.GetConnection(ctx, workspaceID, item.ConnectionID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrInvalid
			}
			return nil, err
		}
		seen[item.Agent] = true
		values = append(values, AgentDefault{Agent: item.Agent, ConnectionID: item.ConnectionID, Model: item.Model, MaxTokens: item.MaxTokens, UpdatedAt: now})
	}
	return s.repository.SaveAgentDefaults(ctx, workspaceID, values)
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

func (s *Service) CountConnections(ctx context.Context, workspaceID string) (int, error) {
	return s.repository.CountConnections(ctx, workspaceID)
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
	return s.DiscoverModels(ctx, workspaceID, connectionID, false)
}

func (s *Service) DiscoverModels(ctx context.Context, workspaceID, connectionID string, refresh bool) (ConnectionTest, error) {
	stored, err := s.repository.GetConnection(ctx, workspaceID, connectionID)
	if err != nil {
		return ConnectionTest{}, err
	}
	cacheKey := fmt.Sprintf("llm-models:%s:%s:%d", workspaceID, connectionID, stored.Connection.UpdatedAt.UnixNano())
	if !refresh && s.modelCache != nil {
		if models, found, cacheErr := s.modelCache.Get(ctx, cacheKey); cacheErr == nil && found {
			return ConnectionTest{Status: "connected", Models: models}, nil
		}
	}
	token, err := s.cipher.Decrypt(stored.TokenCiphertext)
	if err != nil {
		return ConnectionTest{}, err
	}
	models, err := s.discoverer.Models(ctx, stored.Connection, token)
	if err != nil {
		return ConnectionTest{}, err
	}
	if s.modelCache != nil && s.cacheTTL > 0 {
		_ = s.modelCache.Set(ctx, cacheKey, models, s.cacheTTL)
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
