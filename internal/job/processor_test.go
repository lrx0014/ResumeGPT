package job

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type processorTestQueue struct {
	retried bool
	failed  bool
}

func (*processorTestQueue) ClaimKind(context.Context, string, string, time.Duration) (workqueue.Job, error) {
	return workqueue.Job{}, workqueue.ErrEmpty
}

func (q *processorTestQueue) Retry(context.Context, workqueue.Job, string, string, string, time.Duration) error {
	q.retried = true
	return nil
}

func (q *processorTestQueue) Fail(context.Context, string, string, string, string) error {
	q.failed = true
	return nil
}

type processorTestRepository struct {
	states []string
	stored ParsedJob
}

func (*processorTestRepository) QueueImport(context.Context, Job, workqueue.Job) (Job, error) {
	return Job{}, nil
}

func (r *processorTestRepository) StoreImport(_ context.Context, _ workqueue.Job, parsed ParsedJob) (bool, error) {
	r.stored = parsed
	return true, nil
}

func (r *processorTestRepository) SetImportState(_ context.Context, _ workqueue.Job, state, _ string) error {
	r.states = append(r.states, state)
	return nil
}

type processorTestFetcher struct {
	parsed ParsedJob
	err    error
}

func (f processorTestFetcher) Fetch(context.Context, string) (ParsedJob, error) {
	return f.parsed, f.err
}

func TestImportProcessorStoresFetchedJob(t *testing.T) {
	repository := &processorTestRepository{}
	queue := &processorTestQueue{}
	expected := ParsedJob{Title: "Platform Engineer", Company: "Example"}
	processor := ImportProcessor{Queue: queue, Repository: repository, Fetcher: processorTestFetcher{parsed: expected},
		WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	processor.handle(context.Background(), importProcessorTask(t))
	if len(repository.states) != 1 || repository.states[0] != "fetching" || repository.stored != expected {
		t.Fatalf("unexpected processing result: states=%v stored=%#v", repository.states, repository.stored)
	}
	if queue.retried || queue.failed {
		t.Fatal("successful import finalized as a failure")
	}
}

func TestImportProcessorRecordsActionableFailure(t *testing.T) {
	repository := &processorTestRepository{}
	queue := &processorTestQueue{}
	processor := ImportProcessor{Queue: queue, Repository: repository, Fetcher: processorTestFetcher{err: &FetchError{
		Code: "page_access_denied", Message: "Add the job manually instead.",
	}}, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	processor.handle(context.Background(), importProcessorTask(t))
	if len(repository.states) != 2 || repository.states[0] != "fetching" || repository.states[1] != "needs_user_action" {
		t.Fatalf("unexpected failure states: %v", repository.states)
	}
	if !queue.failed || queue.retried {
		t.Fatalf("unexpected queue result: failed=%v retried=%v", queue.failed, queue.retried)
	}
}

func importProcessorTask(t *testing.T) workqueue.Job {
	t.Helper()
	payload, err := json.Marshal(ImportPayload{JobID: "job_test", SourceURL: "https://www.linkedin.com/jobs/view/123"})
	if err != nil {
		t.Fatal(err)
	}
	return workqueue.Job{ID: "task_test", WorkspaceID: "ws_test", Payload: payload, Attempt: 1, MaxAttempts: 4, LeaseOwner: "worker"}
}
