package memory

import (
	"context"
	"sync"

	"github.com/lrx0014/ResumeGPT/internal/job"
)

type JobRepository struct {
	mu    sync.RWMutex
	items map[string]job.Job
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
			result = append(result, item)
		}
	}
	return result, nil
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[value.ID] = value
	return value, nil
}
