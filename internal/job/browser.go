package job

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
)

type BrowserAction struct {
	Type      string `json:"type"`
	ElementID string `json:"elementId,omitempty"`
}

type BrowserElement struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Text string `json:"text"`
}

type PageSnapshot struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	VisibleText string            `json:"visibleText"`
	Metadata    map[string]string `json:"metadata"`
	JSONLD      []string          `json:"jsonLd"`
	Elements    []BrowserElement  `json:"elements"`
}

type PageBrowser interface {
	Render(context.Context, string, []BrowserAction) (PageSnapshot, error)
}

type HTTPPageBrowser struct {
	endpoint string
	client   *http.Client
}

func NewHTTPPageBrowser(rawURL string) (*HTTPPageBrowser, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(rawURL), "/"))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("invalid web worker URL")
	}
	return &HTTPPageBrowser{endpoint: parsed.String() + "/v1/pages/render", client: &http.Client{Timeout: 50 * time.Second}}, nil
}

func (b *HTTPPageBrowser) Render(ctx context.Context, sourceURL string, actions []BrowserAction) (PageSnapshot, error) {
	if actions == nil {
		actions = []BrowserAction{}
	}
	payload, _ := json.Marshal(map[string]any{"url": sourceURL, "actions": actions})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(payload))
	if err != nil {
		return PageSnapshot{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := b.client.Do(request)
	if err != nil {
		return PageSnapshot{}, &FetchError{Code: "browser_unavailable", Message: "The AI browser could not open the job page.", Retryable: true}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 256*1024))
		var value struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &value)
		if value.Error.Message == "" {
			value.Error.Message = fmt.Sprintf("The AI browser returned status %d.", response.StatusCode)
		}
		return PageSnapshot{}, &FetchError{Code: first(value.Error.Code, "browser_failed"), Message: value.Error.Message,
			Retryable: response.StatusCode >= 500}
	}
	var snapshot PageSnapshot
	if err := json.NewDecoder(io.LimitReader(response.Body, 2*1024*1024)).Decode(&snapshot); err != nil {
		return PageSnapshot{}, &FetchError{Code: "browser_response_invalid", Message: "The AI browser returned an invalid page snapshot.", Retryable: true}
	}
	return snapshot, nil
}
