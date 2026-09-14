package job

import (
	"context"
	"errors"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

var ErrNotFound = errors.New("job not found")

type Repository interface {
	List(ctx context.Context, workspaceID string) ([]Job, error)
	Get(ctx context.Context, workspaceID, jobID string) (Job, error)
	Create(ctx context.Context, value Job) (Job, error)
	Update(ctx context.Context, value Job) (Job, error)
	Delete(ctx context.Context, workspaceID, jobID string) error
}

type ImportRepository interface {
	QueueImport(ctx context.Context, value Job, task workqueue.Job) (Job, error)
	StoreImport(ctx context.Context, task workqueue.Job, parsed ParsedJob) (bool, error)
	SetImportState(ctx context.Context, task workqueue.Job, state, message string) error
}
