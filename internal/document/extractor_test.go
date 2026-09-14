package document

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestHTTPExtractorSendsDocumentAndDecodesResult(t *testing.T) {
	extractor := &HTTPExtractor{endpoint: "http://document-worker", client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v1/extractions" || request.Header.Get("X-Document-Name") != "resume.pdf" || request.ContentLength != 8 {
			t.Errorf("unexpected extraction request: path=%s name=%s length=%d", request.URL.Path, request.Header.Get("X-Document-Name"), request.ContentLength)
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != "evidence" {
			t.Errorf("request body = %q, want evidence", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
			`{"media_type":"application/pdf","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","parser_version":"document-extractor-v1","malware_status":"clean","segments":[{"text":"Evidence","page":1,"paragraph":1,"confidence":1}]}`,
		)), Header: make(http.Header)}, nil
	})}}
	result, err := extractor.Extract(context.Background(), "resume.pdf", strings.NewReader("evidence"), 8)
	if err != nil || len(result.Segments) != 1 || result.Segments[0].Text != "Evidence" {
		t.Fatalf("unexpected result: %#v, error: %v", result, err)
	}
}

func TestHTTPExtractorPreservesActionableFailure(t *testing.T) {
	extractor := &HTTPExtractor{endpoint: "http://document-worker", client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnprocessableEntity, Body: io.NopCloser(strings.NewReader(
			`{"error":{"code":"malware_detected","message":"Malware scanning rejected the document."}}`,
		)), Header: make(http.Header)}, nil
	})}}
	_, err := extractor.Extract(context.Background(), "resume.pdf", strings.NewReader("evidence"), 8)
	failure, ok := err.(*ExtractionFailure)
	if !ok || failure.Code != "malware_detected" || failure.Retryable {
		t.Fatalf("unexpected error: %#v", err)
	}
}
