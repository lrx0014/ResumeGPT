package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

const jobImportConsumer = "job-page-importer-v1"

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
			SELECT id,workspace_id,title,company,location,country,city,work_mode,employment_type,source_url,
				'',status,import_state,import_error,origin,COALESCE(hunter_id,''),created_at,updated_at,LENGTH(BTRIM(description)) > 0
			FROM jobs
			WHERE workspace_id = $1 AND (origin <> 'hunter' OR import_state IN ('ready','manual'))
			ORDER BY created_at DESC, id`, workspaceID)
		if err != nil {
			return fmt.Errorf("query jobs: %w", err)
		}
		defer rows.Close()
		items = make([]job.Job, 0)
		for rows.Next() {
			var item job.Job
			if err := rows.Scan(append(jobFields(&item), &item.HasDescription)...); err != nil {
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
			SELECT id,workspace_id,title,company,location,country,city,work_mode,employment_type,source_url,
				description,status,import_state,import_error,origin,COALESCE(hunter_id,''),created_at,updated_at
			FROM jobs
			WHERE workspace_id = $1 AND id = $2`, workspaceID, jobID).Scan(jobFields(&item)...)
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
	if value.Origin == "" {
		value.Origin = "manual"
	}
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO jobs
				(id,workspace_id,title,company,location,country,city,work_mode,employment_type,source_url,
				description,status,import_state,import_error,origin,hunter_id,created_at,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			value.ID, value.WorkspaceID, value.Title, value.Company, value.Location, value.Country, value.City,
			value.WorkMode, value.EmploymentType, value.SourceURL, value.Description, value.Status,
			value.ImportState, value.ImportError, value.Origin, nullableText(value.HunterID), value.CreatedAt, value.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert job: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "job.created.v1", "job", value.ID, map[string]string{"jobId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "job.create", "job", value.ID)
	})
	return value, err
}

func (r *JobRepository) Update(ctx context.Context, value job.Job) (job.Job, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE jobs SET title=$3,company=$4,location=$5,country=$6,city=$7,work_mode=$8,
			employment_type=$9,source_url=$10,description=$11,status=$12,import_state=$13,import_error='',updated_at=$14
			WHERE workspace_id=$1 AND id=$2 RETURNING created_at`, value.WorkspaceID, value.ID, value.Title,
			value.Company, value.Location, value.Country, value.City, value.WorkMode, value.EmploymentType,
			value.SourceURL, value.Description, value.Status, value.ImportState, value.UpdatedAt).Scan(&value.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return job.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("update job: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "job.updated.v1", "job", value.ID, map[string]string{"jobId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "job.update", "job", value.ID)
	})
	return value, err
}

func (r *JobRepository) UpdateStatus(ctx context.Context, workspaceID, jobID, status string, updatedAt time.Time) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE jobs SET status=$3,updated_at=$4 WHERE workspace_id=$1 AND id=$2`, workspaceID, jobID, status, updatedAt)
		if err != nil {
			return fmt.Errorf("update job status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return job.ErrNotFound
		}
		if err := appendEvent(ctx, tx, workspaceID, "job.status.updated.v1", "job", jobID, map[string]string{"jobId": jobID, "status": status}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "job.status.update", "job", jobID)
	})
}

func (r *JobRepository) Delete(ctx context.Context, workspaceID, jobID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='cancelled',lease_owner=NULL,lease_expires_at=NULL,
			error_class='job_deleted',error_message='The job was deleted before URL import completed.',updated_at=now()
			WHERE workspace_id=$1 AND kind=$2 AND payload->>'jobId'=$3
			AND state IN ('queued','retry_wait')`, workspaceID, job.ImportJobKind, jobID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM jobs WHERE workspace_id=$1 AND id=$2`, workspaceID, jobID)
		if err != nil {
			return fmt.Errorf("delete job: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return job.ErrNotFound
		}
		if err := appendEvent(ctx, tx, workspaceID, "job.deleted.v1", "job", jobID, map[string]string{"jobId": jobID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "job.delete", "job", jobID)
	})
}

func jobFields(value *job.Job) []any {
	return []any{&value.ID, &value.WorkspaceID, &value.Title, &value.Company, &value.Location, &value.Country,
		&value.City, &value.WorkMode, &value.EmploymentType, &value.SourceURL, &value.Description, &value.Status,
		&value.ImportState, &value.ImportError, &value.Origin, &value.HunterID, &value.CreatedAt, &value.UpdatedAt}
}

func (r *JobRepository) QueueImport(ctx context.Context, value job.Job, task workqueue.Job) (job.Job, error) {
	if value.Origin == "" {
		value.Origin = "url_import"
	}
	var result job.Job
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, value.WorkspaceID+"|"+value.SourceURL); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `SELECT id,workspace_id,title,company,location,country,city,work_mode,employment_type,
			source_url,description,status,import_state,import_error,origin,COALESCE(hunter_id,''),created_at,updated_at
			FROM jobs WHERE workspace_id=$1 AND source_url=$2 ORDER BY created_at LIMIT 1`, value.WorkspaceID, value.SourceURL).Scan(jobFields(&result)...)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if len(task.Payload) == 0 {
			payload, err := json.Marshal(job.ImportPayload{JobID: value.ID, SourceURL: value.SourceURL, Mode: "standard"})
			if err != nil {
				return err
			}
			task.Payload = payload
		}
		if _, err := tx.Exec(ctx, `INSERT INTO jobs
			(id,workspace_id,title,company,location,country,city,work_mode,employment_type,source_url,description,
			status,import_state,import_error,origin,hunter_id,created_at,updated_at)
			VALUES ($1,$2,'','','','','','', '',$3,'',$4,$5,'',$6,$7,$8,$9)`, value.ID, value.WorkspaceID,
			value.SourceURL, value.Status, value.ImportState, value.Origin, nullableText(value.HunterID), value.CreatedAt, value.UpdatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO durable_jobs
			(id,workspace_id,kind,idempotency_key,payload,max_attempts,available_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, task.ID, task.WorkspaceID, task.Kind, task.IdempotencyKey,
			task.Payload, task.MaxAttempts, task.AvailableAt); err != nil {
			return err
		}
		result = value
		if err := appendEvent(ctx, tx, value.WorkspaceID, "job.import.queued.v1", "job", value.ID,
			map[string]string{"jobId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "job.import.queue", "job", value.ID)
	})
	return result, err
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *JobRepository) StoreImport(ctx context.Context, task workqueue.Job, parsed job.ParsedJob) (bool, error) {
	processed := false
	importState, importError := "ready", ""
	if parsed.Title == "" || parsed.Company == "" {
		importState = "needs_user_action"
		importError = "Some job details could not be detected. Review and complete the fields manually."
	}
	err := withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		var payload job.ImportPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO inbox_messages (consumer,message_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, jobImportConsumer, task.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		if importState == "needs_user_action" {
			quarantined, err := quarantineHunterJob(ctx, tx, task.WorkspaceID, payload.JobID, "incomplete_job_details", importError)
			if err != nil {
				return err
			}
			if quarantined {
				if err := completeImportTask(ctx, tx, task); err != nil {
					return err
				}
				processed = true
				return nil
			}
		}
		tag, err = tx.Exec(ctx, `UPDATE jobs SET title=$3,company=$4,location=$5,country=$6,city=$7,work_mode=$8,
			employment_type=$9,description=$10,import_state=$11,import_error=$12,updated_at=now()
			WHERE workspace_id=$1 AND id=$2 AND import_state IN ('queued','fetching','analyzing')`, task.WorkspaceID, payload.JobID,
			parsed.Title, parsed.Company, parsed.Location, parsed.Country, parsed.City, parsed.WorkMode,
			parsed.EmploymentType, parsed.Description, importState, importError)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE workspace_id=$1 AND id=$2)`, task.WorkspaceID, payload.JobID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return job.ErrNotFound
			}
		}
		if err := completeImportTask(ctx, tx, task); err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, task.WorkspaceID, "job.import.completed.v1", "job", payload.JobID,
			map[string]string{"jobId": payload.JobID}); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, task.WorkspaceID, "job.import.complete", "job", payload.JobID); err != nil {
			return err
		}
		processed = true
		return nil
	})
	return processed, err
}

func completeImportTask(ctx context.Context, tx pgx.Tx, task workqueue.Job) error {
	tag, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='succeeded',lease_owner=NULL,lease_expires_at=NULL,updated_at=now()
		WHERE id=$1 AND lease_owner=$2 AND state='running'`, task.ID, task.LeaseOwner)
	if err != nil || tag.RowsAffected() != 1 {
		return fmt.Errorf("complete job import: lease lost")
	}
	return nil
}

func (r *JobRepository) QuarantineImport(ctx context.Context, task workqueue.Job, code, message string) (bool, error) {
	var payload job.ImportPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return false, err
	}
	quarantined := false
	err := withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		var err error
		quarantined, err = quarantineHunterJob(ctx, tx, task.WorkspaceID, payload.JobID, code, message)
		return err
	})
	return quarantined, err
}

func quarantineHunterJob(ctx context.Context, tx pgx.Tx, workspaceID, jobID, code, message string) (bool, error) {
	var origin, hunterID, sourceURL string
	err := tx.QueryRow(ctx, `SELECT origin,COALESCE(hunter_id,''),source_url FROM jobs WHERE workspace_id=$1 AND id=$2`, workspaceID, jobID).
		Scan(&origin, &hunterID, &sourceURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, job.ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if origin != "hunter" || hunterID == "" {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `INSERT INTO job_hunter_review_items
		(id,workspace_id,hunter_id,source_url,failure_code,failure_message,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,now(),now())
		ON CONFLICT (workspace_id,hunter_id,source_url) DO UPDATE
		SET failure_code=EXCLUDED.failure_code,failure_message=EXCLUDED.failure_message,updated_at=now()`,
		id.New("review"), workspaceID, hunterID, sourceURL, code, message); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM jobs WHERE workspace_id=$1 AND id=$2`, workspaceID, jobID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE job_hunters SET last_found_count=GREATEST(0,last_found_count-1),updated_at=now()
		WHERE workspace_id=$1 AND id=$2`, workspaceID, hunterID); err != nil {
		return false, err
	}
	if err := appendEvent(ctx, tx, workspaceID, "job.hunter.review_required.v1", "job_hunter", hunterID,
		map[string]string{"hunterId": hunterID, "sourceUrl": sourceURL}); err != nil {
		return false, err
	}
	return true, appendAudit(ctx, tx, workspaceID, "job_hunter.review.require", "job_hunter", hunterID)
}

func (r *JobRepository) SetImportState(ctx context.Context, task workqueue.Job, state, message string) error {
	var payload job.ImportPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return err
	}
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE jobs SET import_state=$3,import_error=$4,updated_at=now()
			WHERE workspace_id=$1 AND id=$2 AND import_state IN ('queued','fetching','analyzing')`, task.WorkspaceID, payload.JobID, state, message)
		if err != nil || tag.RowsAffected() > 0 {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE workspace_id=$1 AND id=$2)`, task.WorkspaceID, payload.JobID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return job.ErrNotFound
		}
		return nil
	})
}
