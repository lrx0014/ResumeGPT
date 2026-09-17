package memory

import (
	"context"

	"github.com/lrx0014/ResumeGPT/internal/taskmonitor"
)

// TaskMonitorRepository keeps the System page available when durable jobs are
// disabled in the zero-dependency memory configuration.
type TaskMonitorRepository struct{}

func NewTaskMonitorRepository() *TaskMonitorRepository { return &TaskMonitorRepository{} }

func (*TaskMonitorRepository) List(_ context.Context, _ string, filter taskmonitor.Filter) (taskmonitor.Page, error) {
	return taskmonitor.Page{
		Items:    []taskmonitor.Task{},
		Counts:   map[string]int{},
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (*TaskMonitorRepository) Get(context.Context, string, string) (taskmonitor.Task, error) {
	return taskmonitor.Task{}, taskmonitor.ErrNotFound
}
