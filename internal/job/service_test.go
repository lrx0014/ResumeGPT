package job_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

func TestJobLifecycle(t *testing.T) {
	service := job.NewService(memory.NewJobRepository())
	created, err := service.Create(context.Background(), "ws_personal", job.CreateInput{
		Title: " Backend Engineer ", Company: " Example ", City: " Berlin ", Country: " Germany ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Title != "Backend Engineer" || created.City != "Berlin" || created.Status != "interested" || created.ImportState != "manual" {
		t.Fatalf("unexpected created job: %#v", created)
	}
	updated, err := service.Update(context.Background(), "ws_personal", created.ID, job.UpdateInput{
		Title: "Staff Engineer", Company: "Example", City: "Munich", Country: "Germany", Status: "preparing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "preparing" || updated.City != "Munich" {
		t.Fatalf("unexpected updated job: %#v", updated)
	}
	if err := service.Delete(context.Background(), "ws_personal", created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "ws_personal", created.ID); !errors.Is(err, job.ErrNotFound) {
		t.Fatalf("deleted job error = %v, want not found", err)
	}
}

func TestImportURLValidation(t *testing.T) {
	valid := []string{
		"https://www.linkedin.com/jobs/view/123?trackingId=abc#details",
		"https://de.indeed.com/viewjob?jk=123",
	}
	for _, value := range valid {
		if _, err := job.NormalizeImportURL(value); err != nil {
			t.Fatalf("URL %q rejected: %v", value, err)
		}
	}
	invalid := []string{
		"http://www.linkedin.com/jobs/view/123",
		"https://linkedin.com.evil.example/jobs/123",
		"https://user:password@indeed.com/viewjob?jk=123",
		"https://www.indeed.com:8443/viewjob?jk=123",
		"https://example.com/job/123",
	}
	for _, value := range invalid {
		if _, err := job.NormalizeImportURL(value); !errors.Is(err, job.ErrInvalidURL) {
			t.Fatalf("URL %q error = %v, want invalid URL", value, err)
		}
	}
}

func TestRejectsInvalidJobFields(t *testing.T) {
	service := job.NewService(memory.NewJobRepository())
	for _, input := range []job.CreateInput{
		{},
		{Title: "Engineer", Company: "Example", Status: "unknown"},
		{Title: "Engineer\x00", Company: "Example"},
	} {
		if _, err := service.Create(context.Background(), "ws_personal", input); !errors.Is(err, job.ErrInvalidInput) {
			t.Fatalf("input %#v error = %v, want invalid", input, err)
		}
	}
}

type captureImportRepository struct {
	task workqueue.Job
}

func (r *captureImportRepository) QueueImport(_ context.Context, value job.Job, task workqueue.Job) (job.Job, error) {
	r.task = task
	return value, nil
}
func (*captureImportRepository) StoreImport(context.Context, workqueue.Job, job.ParsedJob) (bool, error) {
	return false, nil
}
func (*captureImportRepository) SetImportState(context.Context, workqueue.Job, string, string) error {
	return nil
}

func TestAIAssistedImportAcceptsAnyPublicHTTPSHostAndQueuesModelChoice(t *testing.T) {
	repository := &captureImportRepository{}
	service := job.NewImportService(repository)
	items, err := service.Create(context.Background(), "ws_personal", job.ImportInput{
		URLs: []string{"https://careers.example.com/jobs/123#description"}, AIAssisted: true,
		ConnectionID: "llm_test", Model: "example-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SourceURL != "https://careers.example.com/jobs/123" {
		t.Fatalf("unexpected imported jobs: %#v", items)
	}
	var payload job.ImportPayload
	if err := json.Unmarshal(repository.task.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Mode != "agent" || payload.ConnectionID != "llm_test" || payload.Model != "example-model" {
		t.Fatalf("unexpected task payload: %#v", payload)
	}
}

func TestAIAssistedImportRequiresModelConfiguration(t *testing.T) {
	service := job.NewImportService(&captureImportRepository{})
	_, err := service.Create(context.Background(), "ws_personal", job.ImportInput{
		URLs: []string{"https://careers.example.com/jobs/123"}, AIAssisted: true,
	})
	if !errors.Is(err, job.ErrInvalidAIConfig) {
		t.Fatalf("error = %v, want invalid AI configuration", err)
	}
}
