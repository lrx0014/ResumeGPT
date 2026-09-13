package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/job"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) List(ctx context.Context, workspaceID string) ([]job.Job, error) {
	var items []job.Job
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, workspace_id, title, company, location, source_url, description, status, created_at, updated_at
			FROM jobs
			WHERE workspace_id = $1
			ORDER BY created_at DESC, id`, workspaceID)
		if err != nil {
			return fmt.Errorf("query jobs: %w", err)
		}
		defer rows.Close()
		items = make([]job.Job, 0)
		for rows.Next() {
			var item job.Job
			if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Title, &item.Company, &item.Location, &item.SourceURL, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return fmt.Errorf("scan job: %w", err)
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *JobRepository) Get(ctx context.Context, workspaceID, jobID string) (job.Job, error) {
	var item job.Job
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT id, workspace_id, title, company, location, source_url, description, status, created_at, updated_at
			FROM jobs
			WHERE workspace_id = $1 AND id = $2`, workspaceID, jobID).
			Scan(&item.ID, &item.WorkspaceID, &item.Title, &item.Company, &item.Location, &item.SourceURL, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return job.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("query job: %w", err)
		}
		return nil
	})
	return item, err
}

func (r *JobRepository) Create(ctx context.Context, value job.Job) (job.Job, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO jobs
				(id, workspace_id, title, company, location, source_url, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			value.ID, value.WorkspaceID, value.Title, value.Company, value.Location, value.SourceURL, value.Description, value.Status, value.CreatedAt, value.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert job: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "job.created.v1", "job", value.ID, value); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "job.create", "job", value.ID)
	})
	return value, err
}
