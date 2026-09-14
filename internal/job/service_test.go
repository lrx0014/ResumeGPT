package job_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/job"
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
