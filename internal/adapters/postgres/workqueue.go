package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type WorkQueue struct {
	pool *pgxpool.Pool
}

func NewWorkQueue(pool *pgxpool.Pool) *WorkQueue {
	return &WorkQueue{pool: pool}
}

func (q *WorkQueue) Enqueue(ctx context.Context, value workqueue.Job) error {
	return withWorkspaceTx(ctx, q.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO durable_jobs
				(id, workspace_id, kind, idempotency_key, payload, max_attempts, available_at, deadline_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (workspace_id, kind, idempotency_key) DO NOTHING`,
			value.ID, value.WorkspaceID, value.Kind, value.IdempotencyKey, value.Payload,
			value.MaxAttempts, value.AvailableAt, value.DeadlineAt,
		)
		if err != nil {
			return fmt.Errorf("enqueue durable job: %w", err)
		}
		return nil
	})
}

func (q *WorkQueue) Claim(ctx context.Context, workerID string, lease time.Duration) (workqueue.Job, error) {
	var value workqueue.Job
	err := q.pool.QueryRow(ctx, "SELECT id, workspace_id, kind, state, idempotency_key, payload, attempt, max_attempts, available_at, deadline_at, lease_owner, lease_expires_at FROM claim_durable_job($1, $2)",
		workerID, int(lease.Seconds()),
	).Scan(&value.ID, &value.WorkspaceID, &value.Kind, &value.State, &value.IdempotencyKey, &value.Payload,
		&value.Attempt, &value.MaxAttempts, &value.AvailableAt, &value.DeadlineAt, &value.LeaseOwner, &value.LeaseExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return workqueue.Job{}, workqueue.ErrEmpty
	}
	if err != nil {
		return workqueue.Job{}, fmt.Errorf("claim durable job: %w", err)
	}
	return value, nil
}

func (q *WorkQueue) Heartbeat(ctx context.Context, jobID, workerID string, lease time.Duration) error {
	tag, err := q.pool.Exec(ctx, `
		UPDATE durable_jobs SET heartbeat_at = now(), lease_expires_at = now() + make_interval(secs => $3), updated_at = now()
		WHERE id = $1 AND lease_owner = $2 AND state = 'running'`, jobID, workerID, int(lease.Seconds()))
	if err != nil || tag.RowsAffected() != 1 {
		return fmt.Errorf("heartbeat durable job: lease lost")
	}
	return nil
}

func (q *WorkQueue) Complete(ctx context.Context, jobID, workerID string) error {
	tag, err := q.pool.Exec(ctx, `
		UPDATE durable_jobs SET state = 'succeeded', lease_owner = NULL, lease_expires_at = NULL, updated_at = now()
		WHERE id = $1 AND lease_owner = $2 AND state = 'running'`, jobID, workerID)
	if err != nil || tag.RowsAffected() != 1 {
		return fmt.Errorf("complete durable job: lease lost")
	}
	return nil
}

func (q *WorkQueue) Retry(ctx context.Context, value workqueue.Job, workerID, errorClass, message string, delay time.Duration) error {
	nextState := "retry_wait"
	if value.Attempt >= value.MaxAttempts {
		nextState = "failed"
	}
	tag, err := q.pool.Exec(ctx, `
		UPDATE durable_jobs
		SET state = $3, available_at = now() + make_interval(secs => $4), lease_owner = NULL,
		    lease_expires_at = NULL, error_class = $5, error_message = $6, updated_at = now()
		WHERE id = $1 AND lease_owner = $2 AND state = 'running'`,
		value.ID, workerID, nextState, int(delay.Seconds()), errorClass, message)
	if err != nil || tag.RowsAffected() != 1 {
		return fmt.Errorf("retry durable job: lease lost")
	}
	return nil
}

func (q *WorkQueue) Cancel(ctx context.Context, workspaceID, jobID string) error {
	return withWorkspaceTx(ctx, q.pool, workspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE durable_jobs SET state = 'cancelled', lease_owner = NULL, lease_expires_at = NULL, updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND state IN ('queued', 'retry_wait')`, workspaceID, jobID)
		return err
	})
}
