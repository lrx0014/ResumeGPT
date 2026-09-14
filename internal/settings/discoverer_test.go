package settings

import (
	"net"
	"reflect"
	"testing"
)

func TestOpenAICompatibleModelContract(t *testing.T) {
	connection := LLMConnection{ExecutionMode: "local", Provider: "openai_compatible", BaseURL: "http://localhost:8081/v1"}
	endpoint, err := modelEndpoint(connection)
	if err != nil || endpoint != "http://localhost:8081/v1/models" {
		t.Fatalf("endpoint = %q, error = %v", endpoint, err)
	}
	models, err := decodeModels(connection.Provider, []byte(`{"data":[{"id":"model-b"},{"id":"model-a"}]}`))
	if err != nil || !reflect.DeepEqual(models, []string{"model-a", "model-b"}) {
		t.Fatalf("models = %v, error = %v", models, err)
	}
}

func TestOllamaModelContract(t *testing.T) {
	connection := LLMConnection{ExecutionMode: "local", Provider: "ollama", BaseURL: "http://host.docker.internal:11434"}
	endpoint, err := modelEndpoint(connection)
	if err != nil || endpoint != "http://host.docker.internal:11434/api/tags" {
		t.Fatalf("endpoint = %q, error = %v", endpoint, err)
	}
	models, err := decodeModels(connection.Provider, []byte(`{"models":[{"name":"qwen3:4b"}]}`))
	if err != nil || !reflect.DeepEqual(models, []string{"qwen3:4b"}) {
		t.Fatalf("models = %v, error = %v", models, err)
	}
}

func TestConnectionAddressPolicies(t *testing.T) {
	private := net.ParseIP("127.0.0.1")
	public := net.ParseIP("8.8.8.8")
	metadata := net.ParseIP("169.254.169.254")
	if isPublicAddress(private) || isPublicAddress(metadata) || !isPublicAddress(public) {
		t.Fatal("unexpected cloud address policy")
	}
	if !isLocalAddress(private) || isLocalAddress(public) || isLocalAddress(metadata) {
		t.Fatal("unexpected local address policy")
	}
}
