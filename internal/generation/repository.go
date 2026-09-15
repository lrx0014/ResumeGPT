package generation

import (
	"context"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type Repository interface {
	Create(context.Context, Run, workqueue.Job) (Run, error)
	List(context.Context, string) ([]Run, error)
	Get(context.Context, string, string) (Run, error)
	Retry(context.Context, string, string) (Run, error)
	SetStage(context.Context, workqueue.Job, string, string, string, int) error
	Complete(context.Context, workqueue.Job, string, string, string, string, int) error
	Fail(context.Context, workqueue.Job, string, string) error
}
