package profile

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("profile not found")

type Counts struct {
	Total       int `json:"total"`
	WithContent int `json:"withContent"`
}

type Repository interface {
	List(ctx context.Context, workspaceID string) ([]Profile, error)
	Get(ctx context.Context, workspaceID, profileID string) (Profile, error)
	Create(ctx context.Context, value Profile) (Profile, error)
	Update(ctx context.Context, value Profile) (Profile, error)
	Delete(ctx context.Context, workspaceID, profileID string) error
	Count(ctx context.Context, workspaceID string) (Counts, error)
}
