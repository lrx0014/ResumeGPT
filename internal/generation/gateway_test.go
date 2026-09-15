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
