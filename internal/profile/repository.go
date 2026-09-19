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

// Filter drives Search: server-side pagination and text search. Search is
// matched against name, target role, and content.
type Filter struct {
	Search   string
	Page     int
	PageSize int
}

type Page struct {
	Items    []Profile `json:"items"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

type Repository interface {
	List(ctx context.Context, workspaceID string) ([]Profile, error)
	Search(ctx context.Context, workspaceID string, filter Filter) (Page, error)
	Get(ctx context.Context, workspaceID, profileID string) (Profile, error)
	Create(ctx context.Context, value Profile) (Profile, error)
	Update(ctx context.Context, value Profile) (Profile, error)
	Delete(ctx context.Context, workspaceID, profileID string) error
	Count(ctx context.Context, workspaceID string) (Counts, error)
}
