package document

import (
	"bytes"
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

type PDFPages struct {
	PageCount int      `json:"pageCount"`
	Images    []string `json:"images"`
}

func NewHTTPExtractor(endpoint string) (*HTTPExtractor, error) {
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid document worker URL")
	}
	return &HTTPExtractor{endpoint: parsed.String(), client: &http.Client{Timeout: 4 * time.Minute}}, nil
}

func (e *HTTPExtractor) Extract(ctx context.Context, name string, object io.Reader, size int64) (Extraction, error) {
	return e.ExtractTemplate(ctx, name, "", object, size)
}

func (e *HTTPExtractor) ExtractTemplate(ctx context.Context, name, entryFile string, object io.Reader, size int64) (Extraction, error) {
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
	if entryFile != "" {
		request.Header.Set("X-Template-Entry", url.QueryEscape(entryFile))
	}
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

func (e *HTTPExtractor) Preview(ctx context.Context, name string, object io.Reader, size int64) ([]byte, error) {
	return e.PreviewTemplate(ctx, name, "", object, size)
}

func (e *HTTPExtractor) PreviewTemplate(ctx context.Context, name, entryFile string, object io.Reader, size int64) ([]byte, error) {
	if size < 1 || size > MaxDocumentBytes {
		return nil, &ExtractionFailure{Code: "size_limit_exceeded", Message: "Templates must be between 1 byte and 10 MiB."}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+"/v1/previews", io.LimitReader(object, MaxDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	request.ContentLength = size
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("X-Document-Name", url.QueryEscape(filepath.Base(name)))
	if entryFile != "" {
		request.Header.Set("X-Template-Entry", url.QueryEscape(entryFile))
	}
	response, err := e.client.Do(request)
	if err != nil {
		return nil, &ExtractionFailure{Code: "previewer_unavailable", Message: "The isolated preview service is unavailable.", Retryable: true}
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
			return nil, &ExtractionFailure{Code: "preview_failed", Message: "The isolated preview service returned an invalid response.", Retryable: response.StatusCode >= 500}
		}
		return nil, &ExtractionFailure{Code: payload.Error.Code, Message: payload.Error.Message, Retryable: response.StatusCode >= 500}
	}
	if response.Header.Get("Content-Type") != "application/pdf" {
		return nil, &ExtractionFailure{Code: "invalid_preview", Message: "The isolated preview service did not return a PDF."}
	}
	preview, err := io.ReadAll(io.LimitReader(response.Body, 20*1024*1024+1))
	if err != nil || len(preview) < 5 || len(preview) > 20*1024*1024 || string(preview[:5]) != "%PDF-" {
		return nil, &ExtractionFailure{Code: "invalid_preview", Message: "The isolated preview service returned an invalid PDF."}
	}
	return preview, nil
}

func (e *HTTPExtractor) HTMLPDF(ctx context.Context, source string) ([]byte, error) {
	if len(source) < 1 || len(source) > MaxDocumentBytes {
		return nil, &ExtractionFailure{Code: "size_limit_exceeded", Message: "HTML documents must be between 1 byte and 10 MiB."}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+"/v1/html-pdfs", strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	request.ContentLength = int64(len(source))
	request.Header.Set("Content-Type", "text/html; charset=utf-8")
	request.Header.Set("X-Document-Name", "generated.html")
	response, err := e.client.Do(request)
	if err != nil {
		return nil, &ExtractionFailure{Code: "html_renderer_unavailable", Message: "The isolated HTML rendering service is unavailable.", Retryable: true}
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
			return nil, &ExtractionFailure{Code: "html_render_failed", Message: "The isolated HTML renderer returned an invalid response.", Retryable: response.StatusCode >= 500}
		}
		return nil, &ExtractionFailure{Code: payload.Error.Code, Message: payload.Error.Message, Retryable: response.StatusCode >= 500}
	}
	if response.Header.Get("Content-Type") != "application/pdf" {
		return nil, &ExtractionFailure{Code: "invalid_html_pdf", Message: "The isolated HTML renderer did not return a PDF."}
	}
	pdf, err := io.ReadAll(io.LimitReader(response.Body, 20*1024*1024+1))
	if err != nil || len(pdf) < 5 || len(pdf) > 20*1024*1024 || string(pdf[:5]) != "%PDF-" {
		return nil, &ExtractionFailure{Code: "invalid_html_pdf", Message: "The isolated HTML renderer returned an invalid PDF."}
	}
	return pdf, nil
}

func (e *HTTPExtractor) PDFPages(ctx context.Context, pdf []byte) (PDFPages, error) {
	if len(pdf) < 5 || len(pdf) > MaxDocumentBytes || string(pdf[:5]) != "%PDF-" {
		return PDFPages{}, &ExtractionFailure{Code: "invalid_pdf", Message: "Visual review requires a valid PDF."}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+"/v1/pdf-pages", bytes.NewReader(pdf))
	if err != nil {
		return PDFPages{}, err
	}
	request.ContentLength = int64(len(pdf))
	request.Header.Set("Content-Type", "application/pdf")
	request.Header.Set("X-Document-Name", "generated.pdf")
	response, err := e.client.Do(request)
	if err != nil {
		return PDFPages{}, &ExtractionFailure{Code: "previewer_unavailable", Message: "The visual review service is unavailable.", Retryable: true}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return PDFPages{}, &ExtractionFailure{Code: "pdf_raster_failed", Message: "The generated PDF could not be prepared for visual review.", Retryable: response.StatusCode >= 500}
	}
	var result PDFPages
	if json.NewDecoder(io.LimitReader(response.Body, 12*1024*1024)).Decode(&result) != nil || result.PageCount < 1 || len(result.Images) == 0 {
		return PDFPages{}, &ExtractionFailure{Code: "invalid_pdf_pages", Message: "The visual review service returned an invalid result."}
	}
	return result, nil
}
