package job

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPPageBrowserSendsEmptyActionArrayOnInitialRender(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload struct {
			Actions []BrowserAction `json:"actions"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
			return
		}
		if payload.Actions == nil {
			t.Error("initial actions must be an empty array, not null")
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"url":"https://example.com","title":"Example"}`))
	}))
	defer server.Close()

	browser, err := NewHTTPPageBrowser(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := browser.Render(context.Background(), "https://example.com", nil); err != nil {
		t.Fatal(err)
	}
}
