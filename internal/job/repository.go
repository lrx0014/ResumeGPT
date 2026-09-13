package job

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("job not found")

type Repository interface {
	List(ctx context.Context, workspaceID string) ([]Job, error)
	Get(ctx context.Context, workspaceID, jobID string) (Job, error)
	Create(ctx context.Context, value Job) (Job, error)
}
