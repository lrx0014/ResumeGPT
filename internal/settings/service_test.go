package settings_test

import (
	"context"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/settings"
)

type modelDiscoverer struct {
	token string
}

func (d *modelDiscoverer) Models(_ context.Context, _ settings.LLMConnection, token string) ([]string, error) {
	d.token = token
	return []string{"model-a"}, nil
}

func TestConnectionLifecycleProtectsToken(t *testing.T) {
	repository := memory.NewSettingsRepository()
	cipher, err := settings.NewAESGCMTokenCipher("test-settings-encryption-key-at-least-32-characters")
	if err != nil {
		t.Fatal(err)
	}
	discoverer := &modelDiscoverer{}
	service := settings.NewService(repository, cipher, discoverer)
	created, err := service.CreateConnection(context.Background(), "ws_test", settings.LLMConnectionInput{
		Name: "Cloud", ExecutionMode: "cloud", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIToken: "secret-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.APITokenConfigured {
		t.Fatal("created connection did not report its configured token")
	}
	defaults, err := service.SaveAgentDefaults(context.Background(), "ws_test", settings.AgentDefaultsInput{Items: []settings.AgentDefaultInput{
		{Agent: settings.AgentWriter, ConnectionID: created.ID, Model: "model-a"},
		{Agent: settings.AgentVisualReviewer, ConnectionID: created.ID, Model: "vision-model"},
	}})
	if err != nil || len(defaults) != 2 {
		t.Fatalf("save Agent defaults = %#v, error = %v", defaults, err)
	}
	storedDefaults, err := service.ListAgentDefaults(context.Background(), "ws_test")
	if err != nil || len(storedDefaults) != 2 {
		t.Fatalf("list Agent defaults = %#v, error = %v", storedDefaults, err)
	}
	if _, err := service.UpdateConnection(context.Background(), "ws_test", created.ID, settings.LLMConnectionInput{
		Name: "Cloud renamed", ExecutionMode: "cloud", Provider: "openai", BaseURL: "https://api.openai.com/v1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TestConnection(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatal(err)
	}
	if discoverer.token != "secret-token" {
		t.Fatalf("preserved token = %q", discoverer.token)
	}
	updated, err := service.UpdateConnection(context.Background(), "ws_test", created.ID, settings.LLMConnectionInput{
		Name: "Cloud renamed", ExecutionMode: "cloud", Provider: "openai", BaseURL: "https://api.openai.com/v1", ClearAPIToken: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.APITokenConfigured {
		t.Fatal("cleared API token still reports as configured")
	}
}

func TestAgentDefaultsRejectInvalidAgentAndConnection(t *testing.T) {
	cipher, _ := settings.NewAESGCMTokenCipher("test-settings-encryption-key-at-least-32-characters")
	service := settings.NewService(memory.NewSettingsRepository(), cipher, &modelDiscoverer{})
	for _, input := range []settings.AgentDefaultInput{
		{Agent: "unknown", ConnectionID: "llm_missing", Model: "model"},
		{Agent: settings.AgentWriter, ConnectionID: "llm_missing", Model: "model"},
	} {
		if _, err := service.SaveAgentDefaults(context.Background(), "ws_test", settings.AgentDefaultsInput{Items: []settings.AgentDefaultInput{input}}); err != settings.ErrInvalid {
			t.Fatalf("invalid Agent default error = %v, want ErrInvalid", err)
		}
	}
}

func TestPreferencesValidation(t *testing.T) {
	cipher, _ := settings.NewAESGCMTokenCipher("test-settings-encryption-key-at-least-32-characters")
	service := settings.NewService(memory.NewSettingsRepository(), cipher, &modelDiscoverer{})
	if _, err := service.SavePreferences(context.Background(), "ws_test", settings.PreferencesInput{InterfaceLanguage: "zh", Theme: "dark"}); err == nil {
		t.Fatal("unsupported interface language was accepted")
	}
	if _, err := service.SavePreferences(context.Background(), "ws_test", settings.PreferencesInput{InterfaceLanguage: "en", Theme: "dark"}); err != nil {
		t.Fatal(err)
	}
}
