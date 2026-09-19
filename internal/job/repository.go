package job

import (
	"context"
	"errors"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

var ErrNotFound = errors.New("job not found")

// Counts mirrors the same importState categorization the web dashboard
// applies to the full job list: Ready is 'manual'/'ready', Pending is
// 'queued'/'fetching', Attention is 'needs_user_action'/'failed'.
type Counts struct {
	Total     int `json:"total"`
	Ready     int `json:"ready"`
	Pending   int `json:"pending"`
	Attention int `json:"attention"`
}

type Repository interface {
	List(ctx context.Context, workspaceID string) ([]Job, error)
	Get(ctx context.Context, workspaceID, jobID string) (Job, error)
	Create(ctx context.Context, value Job) (Job, error)
	Update(ctx context.Context, value Job) (Job, error)
	UpdateStatus(ctx context.Context, workspaceID, jobID, status string, updatedAt time.Time) error
	Delete(ctx context.Context, workspaceID, jobID string) error
	Count(ctx context.Context, workspaceID string) (Counts, error)
}

type ImportRepository interface {
	QueueImport(ctx context.Context, value Job, task workqueue.Job) (Job, error)
	StoreImport(ctx context.Context, task workqueue.Job, parsed ParsedJob) (bool, error)
	SetImportState(ctx context.Context, task workqueue.Job, state, message string) error
	QuarantineImport(ctx context.Context, task workqueue.Job, code, message string) (bool, error)
}
