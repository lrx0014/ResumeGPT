package generation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/settings"
)

func TestHTTPGatewayOpenAICompatibleVisionRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		messages := body["messages"].([]any)
		user := messages[1].(map[string]any)
		if _, ok := user["content"].([]any); !ok {
			t.Fatal("expected multimodal content")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"approved"}}]}`))
	}))
	defer server.Close()
	gateway := NewHTTPGateway()
	result, err := gateway.Complete(context.Background(), settings.RuntimeConnection{Connection: settings.LLMConnection{Provider: "openai_compatible", BaseURL: server.URL}, APIToken: "secret"}, "vision-model", "system", "review", []string{"aW1hZ2U="}, 512)
	if err != nil || result != "approved" {
		t.Fatalf("complete: %q %v", result, err)
	}
}

func TestHTTPGatewayOpenAIUsesCompletionTokenLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["max_completion_tokens"] != float64(2048) {
			t.Fatalf("max_completion_tokens = %v, want 2048", body["max_completion_tokens"])
		}
		if _, ok := body["max_tokens"]; ok {
			t.Fatal("official OpenAI requests must not include max_tokens")
		}
		if _, ok := body["temperature"]; ok {
			t.Fatal("official OpenAI requests must not override temperature")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"draft"}}]}`))
	}))
	defer server.Close()

	result, err := NewHTTPGateway().Complete(context.Background(), settings.RuntimeConnection{Connection: settings.LLMConnection{Provider: "openai", BaseURL: server.URL}}, "openai-model", "system", "write", nil, 2048)
	if err != nil || result != "draft" {
		t.Fatalf("complete: %q %v", result, err)
	}
}

func TestHTTPGatewayOpenAICompatibleKeepsLegacyTokenLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["max_tokens"] != float64(1024) {
			t.Fatalf("max_tokens = %v, want 1024", body["max_tokens"])
		}
		if _, ok := body["max_completion_tokens"]; ok {
			t.Fatal("OpenAI-compatible requests must not assume max_completion_tokens support")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"draft"}}]}`))
	}))
	defer server.Close()

	result, err := NewHTTPGateway().Complete(context.Background(), settings.RuntimeConnection{Connection: settings.LLMConnection{Provider: "openai_compatible", BaseURL: server.URL}}, "compatible-model", "system", "write", nil, 1024)
	if err != nil || result != "draft" {
		t.Fatalf("complete: %q %v", result, err)
	}
}

func TestHTTPGatewayClassifiesUnsupportedVisionInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"this model does not support image input"}`))
	}))
	defer server.Close()
	_, err := NewHTTPGateway().Complete(context.Background(), settings.RuntimeConnection{Connection: settings.LLMConnection{Provider: "ollama", BaseURL: server.URL}}, "text-model", "system", "review", []string{"aW1hZ2U="}, 512)
	if !errors.Is(err, ErrVisionUnsupported) {
		t.Fatalf("error = %v, want ErrVisionUnsupported", err)
	}
}

func TestVisionUnsupportedRecognizesOllamaMultimodalError(t *testing.T) {
	message := `{"error":{"code":400,"message":"Multimodal data provided, but model does not support multimodal requests."}}`
	if !visionUnsupported(message) {
		t.Fatal("expected Ollama multimodal capability error to be recognized")
	}
}

func TestHTTPGatewayOllamaRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"content":"draft"}}`))
	}))
	defer server.Close()
	result, err := NewHTTPGateway().Complete(context.Background(), settings.RuntimeConnection{Connection: settings.LLMConnection{Provider: "ollama", BaseURL: server.URL}}, "local", "system", "write", nil, 4096)
	if err != nil || result != "draft" {
		t.Fatalf("complete: %q %v", result, err)
	}
}
