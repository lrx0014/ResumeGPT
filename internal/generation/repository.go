package generation

import (
	"context"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type Counts struct {
	Ready  int `json:"ready"`
	Active int `json:"active"`
}

type Repository interface {
	Create(context.Context, Run, workqueue.Job) (Run, error)
	List(context.Context, string) ([]Run, error)
	Count(context.Context, string) (Counts, error)
	Get(context.Context, string, string) (Run, error)
	Delete(context.Context, string, string) error
	Reconfigure(context.Context, Run) (Run, error)
	Retry(context.Context, string, string) (Run, error)
	Revise(context.Context, string, string, string) (Run, error)
	RecordStep(context.Context, workqueue.Job, Step) error
	ListSteps(context.Context, string, string) ([]Step, error)
	GetStep(context.Context, string, string, string) (Step, error)
	SetStage(context.Context, workqueue.Job, string, string, string, int) error
	Complete(context.Context, workqueue.Job, string, string, string, string, int) error
	Fail(context.Context, workqueue.Job, string, string) error
}
