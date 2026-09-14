package document

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const MaxDocumentBytes = 10 * 1024 * 1024

type ExtractionFailure struct {
	Code      string
	Message   string
	Retryable bool
}

func (e *ExtractionFailure) Error() string { return e.Message }

type HTTPExtractor struct {
	endpoint string
	client   *http.Client
}

func NewHTTPExtractor(endpoint string) (*HTTPExtractor, error) {
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid document worker URL")
	}
	return &HTTPExtractor{endpoint: parsed.String(), client: &http.Client{Timeout: 4 * time.Minute}}, nil
}

func (e *HTTPExtractor) Extract(ctx context.Context, name string, object io.Reader, size int64) (Extraction, error) {
	if size < 1 || size > MaxDocumentBytes {
		return Extraction{}, &ExtractionFailure{Code: "size_limit_exceeded", Message: "Documents must be between 1 byte and 10 MiB."}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+"/v1/extractions", io.LimitReader(object, MaxDocumentBytes+1))
	if err != nil {
		return Extraction{}, err
	}
	request.ContentLength = size
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("X-Document-Name", url.QueryEscape(filepath.Base(name)))
	response, err := e.client.Do(request)
	if err != nil {
		return Extraction{}, &ExtractionFailure{Code: "extractor_unavailable", Message: "The isolated extraction service is unavailable.", Retryable: true}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var payload struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&payload); err != nil || payload.Error.Code == "" {
			return Extraction{}, &ExtractionFailure{Code: "extractor_failed", Message: "The isolated extraction service returned an invalid response.", Retryable: response.StatusCode >= 500}
		}
		return Extraction{}, &ExtractionFailure{Code: payload.Error.Code, Message: payload.Error.Message, Retryable: response.StatusCode >= 500}
	}
	var result Extraction
	if err := json.NewDecoder(io.LimitReader(response.Body, 12*1024*1024)).Decode(&result); err != nil {
		return Extraction{}, &ExtractionFailure{Code: "invalid_extraction_result", Message: "The isolated extraction result is invalid."}
	}
	return result, nil
}
