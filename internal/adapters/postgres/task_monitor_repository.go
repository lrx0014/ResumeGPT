package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/taskmonitor"
)

type TaskMonitorRepository struct{ pool *pgxpool.Pool }

func NewTaskMonitorRepository(pool *pgxpool.Pool) *TaskMonitorRepository {
	return &TaskMonitorRepository{pool: pool}
}

func (r *TaskMonitorRepository) List(ctx context.Context, workspaceID string, filter taskmonitor.Filter) (taskmonitor.Page, error) {
	result := taskmonitor.Page{Items: make([]taskmonitor.Task, 0), Page: filter.Page, PageSize: filter.PageSize, Counts: make(map[string]int)}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		search := "%" + filter.Search + "%"
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM durable_jobs WHERE workspace_id=$1
			AND ($2='' OR state=$2) AND ($3='' OR kind=$3)
			AND ($4='%%' OR id ILIKE $4 OR kind ILIKE $4 OR idempotency_key ILIKE $4 OR COALESCE(error_message,'') ILIKE $4)`,
			workspaceID, filter.State, filter.Kind, search).Scan(&result.Total); err != nil {
			return fmt.Errorf("count monitored tasks: %w", err)
		}
		rows, err := tx.Query(ctx, `SELECT id,workspace_id,kind,state,idempotency_key,payload,attempt,max_attempts,
			available_at,deadline_at,lease_expires_at,heartbeat_at,COALESCE(error_class,''),COALESCE(error_message,''),created_at,updated_at
			FROM durable_jobs WHERE workspace_id=$1 AND ($2='' OR state=$2) AND ($3='' OR kind=$3)
			AND ($4='%%' OR id ILIKE $4 OR kind ILIKE $4 OR idempotency_key ILIKE $4 OR COALESCE(error_message,'') ILIKE $4)
			ORDER BY updated_at DESC,id DESC LIMIT $5 OFFSET $6`, workspaceID, filter.State, filter.Kind, search,
			filter.PageSize, (filter.Page-1)*filter.PageSize)
		if err != nil {
			return fmt.Errorf("query monitored tasks: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var item taskmonitor.Task
			if err := rows.Scan(taskFields(&item)...); err != nil {
				return err
			}
			result.Items = append(result.Items, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		counts, err := tx.Query(ctx, `SELECT state,count(*) FROM durable_jobs WHERE workspace_id=$1 GROUP BY state`, workspaceID)
		if err != nil {
			return err
		}
		defer counts.Close()
		for counts.Next() {
			var state string
			var count int
			if err := counts.Scan(&state, &count); err != nil {
				return err
			}
			result.Counts[state] = count
		}
		return counts.Err()
	})
	return result, err
}

func (r *TaskMonitorRepository) Get(ctx context.Context, workspaceID, taskID string) (taskmonitor.Task, error) {
	var result taskmonitor.Task
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id,workspace_id,kind,state,idempotency_key,payload,attempt,max_attempts,
			available_at,deadline_at,lease_expires_at,heartbeat_at,COALESCE(error_class,''),COALESCE(error_message,''),created_at,updated_at
			FROM durable_jobs WHERE workspace_id=$1 AND id=$2`, workspaceID, taskID).Scan(taskFields(&result)...)
		if errors.Is(err, pgx.ErrNoRows) {
			return taskmonitor.ErrNotFound
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT id,event_type,state,attempt,COALESCE(error_class,''),message,occurred_at
			FROM durable_job_events WHERE workspace_id=$1 AND job_id=$2 ORDER BY occurred_at DESC,id DESC`, workspaceID, taskID)
		if err != nil {
			return err
		}
		defer rows.Close()
		result.Events = make([]taskmonitor.Event, 0)
		for rows.Next() {
			var event taskmonitor.Event
			if err := rows.Scan(&event.ID, &event.EventType, &event.State, &event.Attempt, &event.ErrorClass, &event.Message, &event.OccurredAt); err != nil {
				return err
			}
			result.Events = append(result.Events, event)
		}
		return rows.Err()
	})
	return result, err
}

func taskFields(value *taskmonitor.Task) []any {
	return []any{&value.ID, &value.WorkspaceID, &value.Kind, &value.State, &value.IdempotencyKey, &value.Payload,
		&value.Attempt, &value.MaxAttempts, &value.AvailableAt, &value.DeadlineAt, &value.LeaseExpiresAt,
		&value.HeartbeatAt, &value.ErrorClass, &value.ErrorMessage, &value.CreatedAt, &value.UpdatedAt}
}
