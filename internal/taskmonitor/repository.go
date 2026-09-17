package taskmonitor

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("task not found")

type Repository interface {
	List(context.Context, string, Filter) (Page, error)
	Get(context.Context, string, string) (Task, error)
}
