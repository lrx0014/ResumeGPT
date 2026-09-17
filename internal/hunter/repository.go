package hunter

import (
	"context"
	"errors"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

var (
	ErrNotFound = errors.New("job hunter not found")
	ErrConflict = errors.New("job hunter already has an active run")
)

type Repository interface {
	List(context.Context, string) ([]Hunter, error)
	Get(context.Context, string, string) (Hunter, error)
	Create(context.Context, Hunter) (Hunter, error)
	Update(context.Context, Hunter) (Hunter, error)
	Delete(context.Context, string, string) error
	RunNow(context.Context, Hunter, workqueue.Job) error
	ScheduleDue(context.Context, int) (int, error)
	MarkRunning(context.Context, workqueue.Job, string) error
	Complete(context.Context, workqueue.Job, Hunter, []DiscoveredJob) (int, error)
	RecordFailure(context.Context, workqueue.Job, string) error
	ListReviewItems(context.Context, string, string) ([]ReviewItem, error)
	DismissReviewItem(context.Context, string, string, string) error
}

type WebSearch interface {
	Search(context.Context, string, int) ([]SearchResult, error)
}

type Agent interface {
	Hunt(context.Context, Hunter) ([]string, error)
}
