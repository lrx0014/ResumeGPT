package job

import (
	"context"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/settings"
)

type agentTestConnections struct{}

func (agentTestConnections) RuntimeConnection(context.Context, string, string) (settings.RuntimeConnection, error) {
	return settings.RuntimeConnection{Connection: settings.LLMConnection{Provider: "openai", BaseURL: "https://api.example.com/v1"}}, nil
}

type agentTestGateway struct{}

func (agentTestGateway) Complete(context.Context, settings.RuntimeConnection, string, string, string, []string, int, []string) (string, error) {
	return `{"title":"Platform Engineer","company":"Example GmbH","location":"Berlin, Germany","country":"Germany","city":"Berlin","workMode":"Hybrid","employmentType":"Full-time","description":"Build reliable systems."}`, nil
}

type agentTestBrowser struct {
	actions []BrowserAction
}

func (b *agentTestBrowser) Render(_ context.Context, sourceURL string, actions []BrowserAction) (PageSnapshot, error) {
	b.actions = append([]BrowserAction(nil), actions...)
	return PageSnapshot{URL: sourceURL, Title: "Platform Engineer | Example GmbH", VisibleText: "Platform Engineer\nExample GmbH\nBuild reliable systems."}, nil
}

func TestJobImportAgentProducesStructuredJob(t *testing.T) {
	browser := &agentTestBrowser{}
	agent := NewJobImportAgent(agentTestConnections{}, agentTestGateway{}, browser)
	result, err := agent.Fetch(context.Background(), "ws_test", ImportPayload{
		SourceURL: "https://careers.example.com/jobs/123", ConnectionID: "llm_test", Model: "model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Platform Engineer" || result.Company != "Example GmbH" || result.City != "Berlin" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestFinishExtractionRejectsEmptyResult(t *testing.T) {
	tool := &finishExtractionTool{}
	response, err := tool.Call(context.Background(), `{"title":"","company":"","description":""}`)
	if err != nil || tool.succeeded || response == "" {
		t.Fatalf("unexpected empty result handling: response=%q succeeded=%v err=%v", response, tool.succeeded, err)
	}
}

func TestHeuristicSnapshotReadsJobPostingMetadata(t *testing.T) {
	result := heuristicSnapshot(PageSnapshot{JSONLD: []string{`{"@type":"JobPosting","title":"Backend Engineer","hiringOrganization":{"name":"Example"},"description":"Build APIs."}`}})
	if result.Title != "Backend Engineer" || result.Company != "Example" || result.Description != "Build APIs." {
		t.Fatalf("unexpected heuristic result: %#v", result)
	}
}
