package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/settings"
)

var (
	ErrLLM               = errors.New("LLM request failed")
	ErrVisionUnsupported = errors.New("LLM model does not support image input")
)

type Gateway interface {
	Complete(context.Context, settings.RuntimeConnection, string, string, string, []string, int, []string) (string, error)
}

type HTTPGateway struct{ client *http.Client }

func NewHTTPGateway() *HTTPGateway {
	return &HTTPGateway{client: &http.Client{Timeout: 8 * time.Minute}}
}

func (g *HTTPGateway) Complete(ctx context.Context, runtime settings.RuntimeConnection, model, systemPrompt, userPrompt string, images []string, maxTokens int, stopWords []string) (string, error) {
	base, err := url.Parse(strings.TrimRight(runtime.Connection.BaseURL, "/"))
	if err != nil {
		return "", ErrLLM
	}
	var endpoint string
	var body map[string]any
	if runtime.Connection.Provider == "ollama" {
		endpoint = base.String() + "/api/chat"
		message := map[string]any{"role": "user", "content": userPrompt}
		if len(images) > 0 {
			message["images"] = images
		}
		modelOptions := map[string]any{"temperature": 0.2, "num_ctx": 16384, "num_predict": maxTokens}
		if len(stopWords) > 0 {
			modelOptions["stop"] = stopWords
		}
		body = map[string]any{"model": model, "stream": true, "think": false, "messages": []any{map[string]any{"role": "system", "content": systemPrompt}, message}, "options": modelOptions}
	} else {
		if !strings.HasSuffix(base.Path, "/v1") {
			endpoint = base.String() + "/v1/chat/completions"
		} else {
			endpoint = base.String() + "/chat/completions"
		}
		var userContent any = userPrompt
		if len(images) > 0 {
			parts := []any{map[string]any{"type": "text", "text": userPrompt}}
			for _, image := range images {
				parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:image/jpeg;base64," + image}})
			}
			userContent = parts
		}
		body = map[string]any{"model": model, "messages": []any{map[string]any{"role": "system", "content": systemPrompt}, map[string]any{"role": "user", "content": userContent}}}
		if len(stopWords) > 0 {
			body["stop"] = stopWords
		}
		if runtime.Connection.Provider == "openai" {
			body["max_completion_tokens"] = maxTokens
		} else {
			body["temperature"] = 0.2
			body["max_tokens"] = maxTokens
		}
	}
	content, err := g.execute(ctx, runtime, endpoint, body, images)
	if err != nil && len(stopWords) > 0 && stopWordsUnsupported(err) {
		// Some models (notably newer reasoning models served through an
		// OpenAI-compatible API) reject the stop parameter outright. Retry
		// once without it rather than failing the whole agent run.
		removeStopWords(runtime.Connection.Provider, body)
		content, err = g.execute(ctx, runtime, endpoint, body, images)
	}
	return content, err
}

func (g *HTTPGateway) execute(ctx context.Context, runtime settings.RuntimeConnection, endpoint string, body map[string]any, images []string) (string, error) {
	encoded, _ := json.Marshal(body)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", ErrLLM
	}
	request.Header.Set("Content-Type", "application/json")
	if runtime.APIToken != "" {
		request.Header.Set("Authorization", "Bearer "+runtime.APIToken)
	}
	response, err := g.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLLM, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024))
		message := providerError(data)
		if message == "" {
			message = fmt.Sprintf("provider returned status %d", response.StatusCode)
		}
		if len(images) > 0 && visionUnsupported(message) {
			return "", fmt.Errorf("%w: %s", ErrVisionUnsupported, message)
		}
		return "", fmt.Errorf("%w: %s", ErrLLM, message)
	}
	if runtime.Connection.Provider == "ollama" {
		var result strings.Builder
		decoder := json.NewDecoder(io.LimitReader(response.Body, 4*1024*1024))
		for {
			var chunk struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Error string `json:"error"`
				Done  bool   `json:"done"`
			}
			err := decoder.Decode(&chunk)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", fmt.Errorf("%w: invalid Ollama stream", ErrLLM)
			}
			if chunk.Error != "" {
				if len(images) > 0 && visionUnsupported(chunk.Error) {
					return "", fmt.Errorf("%w: %s", ErrVisionUnsupported, chunk.Error)
				}
				return "", fmt.Errorf("%w: %s", ErrLLM, limit(chunk.Error, 500))
			}
			result.WriteString(chunk.Message.Content)
			if chunk.Done {
				break
			}
		}
		if strings.TrimSpace(result.String()) == "" {
			return "", ErrLLM
		}
		return result.String(), nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024))
	if err != nil {
		return "", ErrLLM
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", ErrLLM
	}
	return result.Choices[0].Message.Content, nil
}

// stopWordsUnsupported reports whether err looks like a provider rejecting
// the stop parameter itself, as some newer reasoning models do.
func stopWordsUnsupported(err error) bool {
	value := strings.ToLower(err.Error())
	if !strings.Contains(value, "stop") {
		return false
	}
	patterns := []string{"unsupported parameter", "is not supported with this model", "is not supported for this model", "not supported for this model"}
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}

func removeStopWords(provider string, body map[string]any) {
	if provider == "ollama" {
		if options, ok := body["options"].(map[string]any); ok {
			delete(options, "stop")
		}
		return
	}
	delete(body, "stop")
}

func visionUnsupported(message string) bool {
	value := strings.ToLower(message)
	patterns := []string{
		"does not support image",
		"doesn't support image",
		"not support image",
		"image input is not supported",
		"image inputs are not supported",
		"image_url is only supported",
		"vision is not supported",
		"does not support multimodal",
		"multimodal requests are not supported",
	}
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}

func providerError(data []byte) string {
	var payload struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &payload) != nil || len(payload.Error) == 0 {
		return ""
	}
	var message string
	if json.Unmarshal(payload.Error, &message) == nil {
		return limit(strings.TrimSpace(message), 500)
	}
	var nested struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(payload.Error, &nested) == nil {
		return limit(strings.TrimSpace(nested.Message), 500)
	}
	return ""
}
