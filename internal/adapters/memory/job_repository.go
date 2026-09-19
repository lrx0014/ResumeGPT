package memory

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type JobRepository struct {
	mu    sync.RWMutex
	items map[string]job.Job
}

func (r *JobRepository) QueueImport(_ context.Context, value job.Job, _ workqueue.Job) (job.Job, error) {
	if value.Origin == "" {
		value.Origin = "url_import"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, current := range r.items {
		if current.WorkspaceID == value.WorkspaceID && current.SourceURL == value.SourceURL {
			return current, nil
		}
	}
	r.items[value.ID] = value
	return value, nil
}

func (r *JobRepository) StoreImport(_ context.Context, task workqueue.Job, parsed job.ParsedJob) (bool, error) {
	var payload job.ImportPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[payload.JobID]
	if !ok || current.WorkspaceID != task.WorkspaceID {
		return false, job.ErrNotFound
	}
	if current.ImportState != "queued" && current.ImportState != "fetching" && current.ImportState != "analyzing" {
		return true, nil
	}
	current.Title, current.Company, current.Location = parsed.Title, parsed.Company, parsed.Location
	current.Country, current.City, current.WorkMode = parsed.Country, parsed.City, parsed.WorkMode
	current.EmploymentType, current.Description = parsed.EmploymentType, parsed.Description
	current.ImportState, current.ImportError = "ready", ""
	if current.Title == "" || current.Company == "" {
		current.ImportState = "needs_user_action"
		current.ImportError = "Some job details could not be detected. Review and complete the fields manually."
	}
	r.items[current.ID] = current
	return true, nil
}

func (r *JobRepository) SetImportState(_ context.Context, task workqueue.Job, state, message string) error {
	var payload job.ImportPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[payload.JobID]
	if !ok || current.WorkspaceID != task.WorkspaceID {
		return job.ErrNotFound
	}
	if current.ImportState != "queued" && current.ImportState != "fetching" && current.ImportState != "analyzing" {
		return nil
	}
	current.ImportState, current.ImportError = state, message
	r.items[current.ID] = current
	return nil
}

func (r *JobRepository) QuarantineImport(_ context.Context, task workqueue.Job, _, _ string) (bool, error) {
	var payload job.ImportPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[payload.JobID]
	if !ok || current.WorkspaceID != task.WorkspaceID {
		return false, job.ErrNotFound
	}
	if current.Origin != "hunter" || current.HunterID == "" {
		return false, nil
	}
	delete(r.items, payload.JobID)
	return true, nil
}

func NewJobRepository() *JobRepository {
	return &JobRepository{items: make(map[string]job.Job)}
}

func (r *JobRepository) List(_ context.Context, workspaceID string) ([]job.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]job.Job, 0)
	for _, item := range r.items {
		if item.WorkspaceID == workspaceID {
			item.HasDescription = len(item.Description) > 0
			item.Description = ""
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *JobRepository) Count(_ context.Context, workspaceID string) (job.Counts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var counts job.Counts
	for _, item := range r.items {
		if item.WorkspaceID != workspaceID {
			continue
		}
		counts.Total++
		switch {
		case item.ImportState == "manual" || item.ImportState == "ready":
			counts.Ready++
		case item.ImportState == "queued" || item.ImportState == "fetching":
			counts.Pending++
		case item.ImportState == "needs_user_action" || item.ImportState == "failed":
			counts.Attention++
		}
	}
	return counts, nil
}

func (r *JobRepository) Get(_ context.Context, workspaceID, jobID string) (job.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[jobID]
	if !ok || item.WorkspaceID != workspaceID {
		return job.Job{}, job.ErrNotFound
	}
	return item, nil
}

func (r *JobRepository) Create(_ context.Context, value job.Job) (job.Job, error) {
	if value.Origin == "" {
		value.Origin = "manual"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[value.ID] = value
	return value, nil
}

func (r *JobRepository) Update(_ context.Context, value job.Job) (job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[value.ID]
	if !ok || current.WorkspaceID != value.WorkspaceID {
		return job.Job{}, job.ErrNotFound
	}
	value.CreatedAt = current.CreatedAt
	value.Origin, value.HunterID = current.Origin, current.HunterID
	r.items[value.ID] = value
	return value, nil
}

func (r *JobRepository) UpdateStatus(_ context.Context, workspaceID, jobID, status string, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[jobID]
	if !ok || current.WorkspaceID != workspaceID {
		return job.ErrNotFound
	}
	current.Status, current.UpdatedAt = status, updatedAt
	r.items[jobID] = current
	return nil
}

func (r *JobRepository) Delete(_ context.Context, workspaceID, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[jobID]
	if !ok || current.WorkspaceID != workspaceID {
		return job.ErrNotFound
	}
	delete(r.items, jobID)
	return nil
}
