package job

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type ProcessorQueue interface {
	ClaimKind(context.Context, string, string, time.Duration) (workqueue.Job, error)
	Retry(context.Context, workqueue.Job, string, string, string, time.Duration) error
	Fail(context.Context, string, string, string, string) error
}

type ImportProcessor struct {
	Queue        ProcessorQueue
	Repository   ImportRepository
	Fetcher      Fetcher
	AgentFetcher AgentFetcher
	WorkerID     string
	Logger       *slog.Logger
}

func (p *ImportProcessor) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		task, err := p.Queue.ClaimKind(ctx, p.WorkerID, ImportJobKind, 3*time.Minute)
		if err == nil {
			taskContext, cancel := context.WithTimeout(ctx, 2*time.Minute)
			p.handle(taskContext, task)
			cancel()
		} else if !errors.Is(err, workqueue.ErrEmpty) {
			p.Logger.Error("claim job import", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (p *ImportProcessor) handle(ctx context.Context, task workqueue.Job) {
	var payload ImportPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil || payload.JobID == "" || payload.SourceURL == "" {
		p.finishFailure(ctx, task, "failed", "invalid_job_payload", "The job import task is invalid.", false)
		return
	}
	activeState := "fetching"
	if payload.Mode == "agent" {
		activeState = "analyzing"
	}
	if err := p.Repository.SetImportState(ctx, task, activeState, ""); errors.Is(err, ErrNotFound) {
		p.finishFailure(ctx, task, "failed", "job_deleted", "The job was deleted before import started.", false)
		return
	} else if err != nil {
		p.Logger.Error("update job import state", "task_id", task.ID, "job_id", payload.JobID,
			"state", activeState, "error", err)
		p.finishFailure(ctx, task, "queued", "persistence_failed", "Could not update the job import state.", true)
		return
	}
	var parsed ParsedJob
	var err error
	if payload.Mode == "agent" {
		if p.AgentFetcher == nil {
			err = &FetchError{Code: "ai_import_unavailable", Message: "AI-assisted import is not available. Try again later or add the job manually.", Retryable: true}
		} else {
			parsed, err = p.AgentFetcher.Fetch(ctx, task.WorkspaceID, payload)
		}
	} else {
		parsed, err = p.Fetcher.Fetch(ctx, payload.SourceURL)
	}
	if err != nil {
		var failure *FetchError
		if !errors.As(err, &failure) {
			failure = &FetchError{Code: "fetch_failed", Message: "The job page could not be imported.", Retryable: true}
		}
		state := "needs_user_action"
		if failure.Retryable {
			state = "queued"
		}
		p.finishFailure(ctx, task, state, failure.Code, failure.Message, failure.Retryable)
		return
	}
	processed, err := p.Repository.StoreImport(ctx, task, parsed)
	if errors.Is(err, ErrNotFound) {
		p.finishFailure(ctx, task, "failed", "job_deleted", "The job was deleted during import.", false)
		return
	}
	if err != nil {
		p.finishFailure(ctx, task, "queued", "persistence_failed", "Could not save the imported job details.", true)
		return
	}
	if processed {
		p.Logger.Info("job import completed", "task_id", task.ID, "job_id", payload.JobID)
	}
}

func (p *ImportProcessor) finishFailure(ctx context.Context, task workqueue.Job, state, code, message string, retryable bool) {
	if retryable && task.Attempt >= task.MaxAttempts {
		retryable = false
		state = "needs_user_action"
		message += " Add the job manually or try again later."
	}
	if err := p.Repository.SetImportState(ctx, task, state, message); err != nil {
		p.Logger.Error("record job import failure", "task_id", task.ID, "error", err)
	}
	var err error
	if retryable {
		delay := time.Duration(task.Attempt*task.Attempt) * 5 * time.Second
		err = p.Queue.Retry(ctx, task, p.WorkerID, code, message, delay)
	} else {
		err = p.Queue.Fail(ctx, task.ID, p.WorkerID, code, message)
	}
	if err != nil {
		p.Logger.Error("finalize job import failure", "task_id", task.ID, "error", err)
	}
	p.Logger.Warn("job import failed", "task_id", task.ID, "code", code, "retryable", retryable)
}
