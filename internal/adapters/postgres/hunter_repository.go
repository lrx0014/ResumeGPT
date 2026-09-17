package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/hunter"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type HunterRepository struct{ pool *pgxpool.Pool }

func NewHunterRepository(pool *pgxpool.Pool) *HunterRepository { return &HunterRepository{pool: pool} }

func (r *HunterRepository) List(ctx context.Context, workspaceID string) ([]hunter.Hunter, error) {
	items := make([]hunter.Hunter, 0)
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id,workspace_id,name,role_query,location,work_mode,employment_type,
			experience_years,keywords,additional_prompt,COALESCE(profile_id,''),connection_id,model,max_results,interval_minutes,enabled,next_run_at,
			last_run_at,last_state,last_error,last_found_count,
			(SELECT count(*) FROM job_hunter_review_items review WHERE review.workspace_id=job_hunters.workspace_id AND review.hunter_id=job_hunters.id),
			created_at,updated_at
			FROM job_hunters WHERE workspace_id=$1 ORDER BY created_at DESC,id`, workspaceID)
		if err != nil {
			return fmt.Errorf("query job hunters: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var item hunter.Hunter
			if err := rows.Scan(hunterFields(&item)...); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *HunterRepository) Get(ctx context.Context, workspaceID, hunterID string) (hunter.Hunter, error) {
	var value hunter.Hunter
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id,workspace_id,name,role_query,location,work_mode,employment_type,
			experience_years,keywords,additional_prompt,COALESCE(profile_id,''),connection_id,model,max_results,interval_minutes,enabled,next_run_at,
			last_run_at,last_state,last_error,last_found_count,
			(SELECT count(*) FROM job_hunter_review_items review WHERE review.workspace_id=job_hunters.workspace_id AND review.hunter_id=job_hunters.id),
			created_at,updated_at
			FROM job_hunters WHERE workspace_id=$1 AND id=$2`, workspaceID, hunterID).Scan(hunterFields(&value)...)
		if errors.Is(err, pgx.ErrNoRows) {
			return hunter.ErrNotFound
		}
		return err
	})
	return value, err
}

func (r *HunterRepository) Create(ctx context.Context, value hunter.Hunter) (hunter.Hunter, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO job_hunters
			(id,workspace_id,name,role_query,location,work_mode,employment_type,experience_years,keywords,
			additional_prompt,profile_id,connection_id,model,max_results,interval_minutes,enabled,next_run_at,last_state,created_at,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			value.ID, value.WorkspaceID, value.Name, value.RoleQuery, value.Location, value.WorkMode,
			value.EmploymentType, value.ExperienceYears, value.Keywords, value.AdditionalPrompt, value.ProfileID, value.ConnectionID,
			value.Model, value.MaxResults, value.IntervalMinutes, value.Enabled, value.NextRunAt, value.LastState, value.CreatedAt, value.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert job hunter: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "job.hunter.created.v1", "job_hunter", value.ID, map[string]string{"hunterId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "job_hunter.create", "job_hunter", value.ID)
	})
	return value, err
}

func (r *HunterRepository) Update(ctx context.Context, value hunter.Hunter) (hunter.Hunter, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE job_hunters SET name=$3,role_query=$4,location=$5,work_mode=$6,
			employment_type=$7,experience_years=$8,keywords=$9,additional_prompt=$10,profile_id=NULLIF($11,''),connection_id=$12,
			model=$13,max_results=$14,interval_minutes=$15,enabled=$16,next_run_at=$17,updated_at=$18
			WHERE workspace_id=$1 AND id=$2 RETURNING last_run_at,last_state,last_error,last_found_count,created_at`,
			value.WorkspaceID, value.ID, value.Name, value.RoleQuery, value.Location, value.WorkMode,
			value.EmploymentType, value.ExperienceYears, value.Keywords, value.AdditionalPrompt, value.ProfileID, value.ConnectionID,
			value.Model, value.MaxResults, value.IntervalMinutes, value.Enabled, value.NextRunAt, value.UpdatedAt).Scan(
			&value.LastRunAt, &value.LastState, &value.LastError, &value.LastFoundCount, &value.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return hunter.ErrNotFound
		}
		if err != nil {
			return err
		}
		if !value.Enabled {
			if _, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='cancelled',lease_owner=NULL,lease_expires_at=NULL,
				error_class='hunter_paused',error_message='The Job Hunter was paused.',updated_at=now()
				WHERE workspace_id=$1 AND kind=$2 AND payload->>'hunterId'=$3 AND state IN ('queued','retry_wait')`,
				value.WorkspaceID, hunter.JobKind, value.ID); err != nil {
				return err
			}
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "job_hunter.update", "job_hunter", value.ID)
	})
	return value, err
}

func (r *HunterRepository) Delete(ctx context.Context, workspaceID, hunterID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='cancelled',lease_owner=NULL,lease_expires_at=NULL,
			error_class='hunter_deleted',error_message='The Job Hunter was deleted.',updated_at=now()
			WHERE workspace_id=$1 AND kind=$2 AND payload->>'hunterId'=$3 AND state IN ('queued','retry_wait')`,
			workspaceID, hunter.JobKind, hunterID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM job_hunters WHERE workspace_id=$1 AND id=$2`, workspaceID, hunterID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return hunter.ErrNotFound
		}
		return appendAudit(ctx, tx, workspaceID, "job_hunter.delete", "job_hunter", hunterID)
	})
}

func (r *HunterRepository) RunNow(ctx context.Context, value hunter.Hunter, task workqueue.Job) error {
	return withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM durable_jobs WHERE workspace_id=$1 AND kind=$2
			AND payload->>'hunterId'=$3 AND state IN ('queued','running','retry_wait'))`, value.WorkspaceID, hunter.JobKind, value.ID).Scan(&active); err != nil {
			return err
		}
		if active {
			return hunter.ErrConflict
		}
		if _, err := tx.Exec(ctx, `INSERT INTO durable_jobs
			(id,workspace_id,kind,idempotency_key,payload,max_attempts,available_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, task.ID, task.WorkspaceID, task.Kind, task.IdempotencyKey,
			task.Payload, task.MaxAttempts, task.AvailableAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE job_hunters SET last_state='queued',last_error='',
			next_run_at=now()+make_interval(mins=>interval_minutes),updated_at=now()
			WHERE workspace_id=$1 AND id=$2`, value.WorkspaceID, value.ID)
		return err
	})
}

func (r *HunterRepository) ScheduleDue(ctx context.Context, batchSize int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT schedule_due_job_hunts($1)`, batchSize).Scan(&count)
	return count, err
}

func (r *HunterRepository) MarkRunning(ctx context.Context, task workqueue.Job, hunterID string) error {
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE job_hunters SET last_state='running',last_run_at=now(),last_error='',updated_at=now()
			WHERE workspace_id=$1 AND id=$2`, task.WorkspaceID, hunterID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return hunter.ErrNotFound
		}
		return nil
	})
}

func (r *HunterRepository) Complete(ctx context.Context, task workqueue.Job, value hunter.Hunter, discovered []hunter.DiscoveredJob) (int, error) {
	created := 0
	err := withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		for _, candidate := range discovered {
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, task.WorkspaceID+"|"+candidate.SourceURL); err != nil {
				return err
			}
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(
				SELECT 1 FROM jobs WHERE workspace_id=$1 AND source_url=$2
				UNION ALL
				SELECT 1 FROM job_hunter_review_items WHERE workspace_id=$1 AND source_url=$2
			)`, task.WorkspaceID, candidate.SourceURL).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO jobs
				(id,workspace_id,title,company,location,country,city,work_mode,employment_type,source_url,
				description,status,import_state,import_error,origin,hunter_id,created_at,updated_at)
				VALUES ($1,$2,'','','','','','','',$3,'','interested','queued','','hunter',$4,now(),now())`,
				candidate.JobID, task.WorkspaceID, candidate.SourceURL, value.ID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO durable_jobs
				(id,workspace_id,kind,idempotency_key,payload,max_attempts,available_at)
				VALUES ($1,$2,$3,$4,$5,4,now())`, candidate.TaskID, task.WorkspaceID, job.ImportJobKind,
				candidate.JobID, candidate.Payload); err != nil {
				return err
			}
			created++
		}
		tag, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='succeeded',lease_owner=NULL,lease_expires_at=NULL,updated_at=now()
			WHERE id=$1 AND lease_owner=$2 AND state='running'`, task.ID, task.LeaseOwner)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete job hunt: lease lost")
		}
		_, err = tx.Exec(ctx, `UPDATE job_hunters SET last_state='succeeded',last_error='',last_found_count=$3,updated_at=now()
			WHERE workspace_id=$1 AND id=$2`, task.WorkspaceID, value.ID, created)
		return err
	})
	return created, err
}

func (r *HunterRepository) RecordFailure(ctx context.Context, task workqueue.Job, message string) error {
	var payload hunter.Payload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return err
	}
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE job_hunters SET last_state='failed',last_error=$3,updated_at=now()
			WHERE workspace_id=$1 AND id=$2`, task.WorkspaceID, payload.HunterID, message)
		return err
	})
}

func (r *HunterRepository) ListReviewItems(ctx context.Context, workspaceID, hunterID string) ([]hunter.ReviewItem, error) {
	items := make([]hunter.ReviewItem, 0)
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM job_hunters WHERE workspace_id=$1 AND id=$2)`, workspaceID, hunterID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return hunter.ErrNotFound
		}
		rows, err := tx.Query(ctx, `SELECT id,workspace_id,hunter_id,source_url,failure_code,failure_message,created_at,updated_at
			FROM job_hunter_review_items WHERE workspace_id=$1 AND hunter_id=$2 ORDER BY created_at DESC,id`, workspaceID, hunterID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item hunter.ReviewItem
			if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.HunterID, &item.SourceURL, &item.FailureCode,
				&item.FailureMessage, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *HunterRepository) DismissReviewItem(ctx context.Context, workspaceID, hunterID, reviewID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM job_hunter_review_items WHERE workspace_id=$1 AND hunter_id=$2 AND id=$3`, workspaceID, hunterID, reviewID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return hunter.ErrNotFound
		}
		return appendAudit(ctx, tx, workspaceID, "job_hunter.review.dismiss", "job_hunter_review_item", reviewID)
	})
}

func hunterFields(value *hunter.Hunter) []any {
	return []any{&value.ID, &value.WorkspaceID, &value.Name, &value.RoleQuery, &value.Location, &value.WorkMode,
		&value.EmploymentType, &value.ExperienceYears, &value.Keywords, &value.AdditionalPrompt, &value.ProfileID, &value.ConnectionID,
		&value.Model, &value.MaxResults, &value.IntervalMinutes, &value.Enabled, &value.NextRunAt, &value.LastRunAt, &value.LastState,
		&value.LastError, &value.LastFoundCount, &value.ReviewCount, &value.CreatedAt, &value.UpdatedAt}
}
