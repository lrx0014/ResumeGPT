package hunter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

type Processor struct {
	Queue      workqueue.ClaimQueue
	Repository Repository
	Agent      Agent
	WorkerID   string
	Logger     *slog.Logger
}

func (p *Processor) Run(ctx context.Context) error {
	return workqueue.RunLoop(ctx, p.Queue, p.WorkerID, JobKind, 5*time.Minute, 4*time.Minute,
		p.Logger, "claim job hunt", p.handle)
}

func (p *Processor) handle(ctx context.Context, task workqueue.Job) {
	var payload Payload
	if json.Unmarshal(task.Payload, &payload) != nil || payload.HunterID == "" {
		p.fail(ctx, task, "invalid_hunter_payload", "The Job Hunter task is invalid.", false)
		return
	}
	value, err := p.Repository.Get(ctx, task.WorkspaceID, payload.HunterID)
	if errors.Is(err, ErrNotFound) {
		p.fail(ctx, task, "hunter_deleted", "The Job Hunter was deleted before this run started.", false)
		return
	}
	if err != nil {
		p.fail(ctx, task, "persistence_failed", "Could not load the Job Hunter configuration.", true)
		return
	}
	if err := p.Repository.MarkRunning(ctx, task, payload.HunterID); err != nil {
		p.fail(ctx, task, "persistence_failed", "Could not start the Job Hunter run.", true)
		return
	}
	urls, err := p.Agent.Hunt(ctx, value)
	if err != nil {
		p.fail(ctx, task, "job_hunt_failed", "The Job Hunter could not complete its web search. "+err.Error(), true)
		return
	}
	if len(urls) > value.MaxResults {
		urls = urls[:value.MaxResults]
	}
	discovered := make([]DiscoveredJob, 0, len(urls))
	for _, sourceURL := range urls {
		jobID := id.New("job")
		importPayload, _ := json.Marshal(job.ImportPayload{JobID: jobID, SourceURL: sourceURL, Mode: "agent",
			ConnectionID: value.ConnectionID, Model: value.Model, MaxTokens: value.MaxTokens})
		discovered = append(discovered, DiscoveredJob{JobID: jobID, TaskID: id.New("task"), SourceURL: sourceURL, Payload: importPayload})
	}
	created, err := p.Repository.Complete(ctx, task, value, discovered)
	if err != nil {
		p.fail(ctx, task, "persistence_failed", "Could not save the discovered job opportunities.", true)
		return
	}
	p.Logger.Info("job hunt completed", "task_id", task.ID, "hunter_id", value.ID, "new_jobs", created)
}

func (p *Processor) fail(ctx context.Context, task workqueue.Job, code, message string, retryable bool) {
	if len(message) > 2000 {
		message = message[:2000]
	}
	if err := p.Repository.RecordFailure(ctx, task, message); err != nil {
		p.Logger.Error("record job hunt failure", "task_id", task.ID, "error", err)
	}
	if retryable && task.Attempt < task.MaxAttempts {
		delay := time.Duration(task.Attempt*task.Attempt) * 15 * time.Second
		if err := p.Queue.Retry(ctx, task, p.WorkerID, code, message, delay); err != nil {
			p.Logger.Error("retry job hunt", "task_id", task.ID, "error", err)
		}
		return
	}
	if err := p.Queue.Fail(ctx, task.ID, p.WorkerID, code, message); err != nil {
		p.Logger.Error("fail job hunt", "task_id", task.ID, "error", err)
	}
}

type Scheduler struct {
	Repository Repository
	Logger     *slog.Logger
}

func (s Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if count, err := s.Repository.ScheduleDue(ctx, 20); err != nil {
			s.Logger.Error("schedule due job hunts", "error", err)
		} else if count > 0 {
			s.Logger.Info("scheduled job hunts", "count", count)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
