package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

const documentConsumer = "profile-document-extractor-v1"

type DocumentRepository struct{ pool *pgxpool.Pool }

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

func (r *DocumentRepository) Stage(ctx context.Context, upload document.Upload) error {
	return withWorkspaceTx(ctx, r.pool, upload.WorkspaceID, func(tx pgx.Tx) error {
		var profileID string
		if err := tx.QueryRow(ctx, `SELECT id FROM profiles WHERE workspace_id=$1 AND id=$2`, upload.WorkspaceID, upload.ProfileID).Scan(&profileID); errors.Is(err, pgx.ErrNoRows) {
			return document.ErrNotFound
		} else if err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO document_uploads
			(workspace_id,profile_id,id,object_id,name,declared_media_type,state,created_at,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, upload.WorkspaceID, upload.ProfileID, upload.ID,
			upload.ObjectID, upload.Name, upload.DeclaredMediaType, upload.State, upload.CreatedAt, upload.UpdatedAt)
		if err != nil {
			return fmt.Errorf("stage document upload: %w", err)
		}
		return appendAudit(ctx, tx, upload.WorkspaceID, "document.upload.stage", "document_upload", upload.ID)
	})
}

func (r *DocumentRepository) Queue(ctx context.Context, workspaceID, profileID, uploadID string, job workqueue.Job) (document.Upload, error) {
	var upload document.Upload
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := scanUpload(tx.QueryRow(ctx, `SELECT id,workspace_id,profile_id,object_id,name,declared_media_type,state,
			COALESCE(job_id,''),COALESCE(extracted_text,''),COALESCE(error_code,''),COALESCE(error_message,''),created_at,updated_at
			FROM document_uploads WHERE workspace_id=$1 AND profile_id=$2 AND id=$3 FOR UPDATE`, workspaceID, profileID, uploadID), &upload)
		if errors.Is(err, pgx.ErrNoRows) {
			return document.ErrNotFound
		}
		if err != nil {
			return err
		}
		if upload.State != "staged" {
			if upload.JobID != "" {
				return nil
			}
			return document.ErrState
		}
		job.Payload, _ = marshalDocumentPayload(upload)
		if _, err := tx.Exec(ctx, `INSERT INTO durable_jobs
			(id,workspace_id,kind,idempotency_key,payload,max_attempts,available_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (workspace_id,kind,idempotency_key) DO NOTHING`,
			job.ID, job.WorkspaceID, job.Kind, job.IdempotencyKey, job.Payload, job.MaxAttempts, job.AvailableAt); err != nil {
			return fmt.Errorf("enqueue document extraction: %w", err)
		}
		if err := tx.QueryRow(ctx, `UPDATE document_uploads SET state='queued',job_id=(SELECT id FROM durable_jobs
			WHERE workspace_id=$1 AND kind=$2 AND idempotency_key=$3),updated_at=now() WHERE workspace_id=$1 AND id=$4
			RETURNING id,workspace_id,profile_id,object_id,name,declared_media_type,state,COALESCE(job_id,''),
			COALESCE(extracted_text,''),COALESCE(error_code,''),COALESCE(error_message,''),created_at,updated_at`,
			workspaceID, job.Kind, job.IdempotencyKey, uploadID).Scan(uploadFields(&upload)...); err != nil {
			return err
		}
		return appendEvent(ctx, tx, workspaceID, "profile.document.uploaded.v1", "document_upload", upload.ID,
			map[string]string{"profileId": upload.ProfileID, "uploadId": upload.ID, "jobId": upload.JobID})
	})
	return upload, err
}

func marshalDocumentPayload(upload document.Upload) ([]byte, error) {
	return json.Marshal(document.ExtractPayload{UploadID: upload.ID, ProfileID: upload.ProfileID, ObjectID: upload.ObjectID, Name: upload.Name})
}

func (r *DocumentRepository) Get(ctx context.Context, workspaceID, profileID, uploadID string) (document.Upload, error) {
	var upload document.Upload
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := scanUpload(tx.QueryRow(ctx, `SELECT d.id,d.workspace_id,d.profile_id,d.object_id,d.name,d.declared_media_type,
			CASE WHEN d.state='queued' AND j.state IN ('failed','cancelled') THEN 'failed' ELSE d.state END,
			COALESCE(d.job_id,''),COALESCE(d.extracted_text,''),COALESCE(d.error_code,j.error_class,''),
			COALESCE(d.error_message,j.error_message,''),d.created_at,d.updated_at
			FROM document_uploads d LEFT JOIN durable_jobs j ON j.id=d.job_id
			WHERE d.workspace_id=$1 AND d.profile_id=$2 AND d.id=$3`, workspaceID, profileID, uploadID), &upload)
		if errors.Is(err, pgx.ErrNoRows) {
			return document.ErrNotFound
		}
		return err
	})
	return upload, err
}

func (r *DocumentRepository) DownloadAllowed(ctx context.Context, workspaceID, objectID string) (bool, error) {
	allowed := false
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		var state string
		err := tx.QueryRow(ctx, `SELECT state FROM document_uploads WHERE workspace_id=$1 AND object_id=$2`, workspaceID, objectID).Scan(&state)
		if errors.Is(err, pgx.ErrNoRows) {
			allowed = true
			return nil
		}
		if err != nil {
			return err
		}
		allowed = state == "ready"
		return nil
	})
	return allowed, err
}

func uploadFields(upload *document.Upload) []any {
	return []any{&upload.ID, &upload.WorkspaceID, &upload.ProfileID, &upload.ObjectID, &upload.Name,
		&upload.DeclaredMediaType, &upload.State, &upload.JobID, &upload.ExtractedText, &upload.ErrorCode,
		&upload.ErrorMessage, &upload.CreatedAt, &upload.UpdatedAt}
}

func scanUpload(row scanner, upload *document.Upload) error {
	return row.Scan(uploadFields(upload)...)
}

func (r *DocumentRepository) StoreExtraction(ctx context.Context, job workqueue.Job, uploadID, extractedText string) (bool, error) {
	processed := false
	err := withWorkspaceTx(ctx, r.pool, job.WorkspaceID, func(tx pgx.Tx) error {
		var found string
		if err := tx.QueryRow(ctx, `SELECT id FROM document_uploads WHERE workspace_id=$1 AND id=$2 AND job_id=$3 FOR UPDATE`,
			job.WorkspaceID, uploadID, job.ID).Scan(&found); errors.Is(err, pgx.ErrNoRows) {
			return document.ErrNotFound
		} else if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO inbox_messages (consumer,message_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, documentConsumer, job.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		tag, err = tx.Exec(ctx, `UPDATE document_uploads SET state='ready',extracted_text=$3,error_code=NULL,error_message=NULL,updated_at=now()
			WHERE workspace_id=$1 AND id=$2 AND job_id=$4 AND state='queued'`, job.WorkspaceID, uploadID, extractedText, job.ID)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete document upload: state changed")
		}
		tag, err = tx.Exec(ctx, `UPDATE durable_jobs SET state='succeeded',lease_owner=NULL,lease_expires_at=NULL,updated_at=now()
			WHERE id=$1 AND lease_owner=$2 AND state='running'`, job.ID, job.LeaseOwner)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete document job: lease lost")
		}
		if err := appendAudit(ctx, tx, job.WorkspaceID, "document.upload.extract", "document_upload", uploadID); err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, job.WorkspaceID, "profile.document.extracted.v1", "document_upload", uploadID,
			map[string]string{"uploadId": uploadID, "jobId": job.ID}); err != nil {
			return err
		}
		processed = true
		return nil
	})
	return processed, err
}

func (r *DocumentRepository) RecordFailure(ctx context.Context, job workqueue.Job, state, code, message string) error {
	return withWorkspaceTx(ctx, r.pool, job.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE document_uploads SET state=$3,error_code=$4,error_message=$5,updated_at=now()
			WHERE workspace_id=$1 AND job_id=$2`, job.WorkspaceID, job.ID, state, code, message)
		return err
	})
}
