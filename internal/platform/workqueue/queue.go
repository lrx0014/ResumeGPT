package workqueue

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrEmpty = errors.New("no work available")

type Job struct {
	ID             string
	WorkspaceID    string
	Kind           string
	State          string
	IdempotencyKey string
	Payload        json.RawMessage
	Attempt        int
	MaxAttempts    int
	AvailableAt    time.Time
	DeadlineAt     *time.Time
	LeaseOwner     string
	LeaseExpiresAt *time.Time
}

type Repository interface {
	Enqueue(ctx context.Context, job Job) error
	Claim(ctx context.Context, workerID string, lease time.Duration) (Job, error)
	Heartbeat(ctx context.Context, jobID, workerID string, lease time.Duration) error
	Complete(ctx context.Context, jobID, workerID string) error
	Retry(ctx context.Context, job Job, workerID, errorClass, message string, delay time.Duration) error
	Fail(ctx context.Context, jobID, workerID, errorClass, message string) error
	Cancel(ctx context.Context, workspaceID, jobID string) error
}

type Handler func(context.Context, Job) error
