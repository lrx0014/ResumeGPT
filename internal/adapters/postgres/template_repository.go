package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

const templateConsumer = "template-extractor-v1"

type TemplateRepository struct{ pool *pgxpool.Pool }

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

const templateColumns = `id,workspace_id,name,kind,format,description,source_name,entry_file,declared_media_type,object_id,
    COALESCE(preview_object_id,''),COALESCE(content,''),state,COALESCE(job_id,''),COALESCE(error_code,''),COALESCE(error_message,''),created_at,updated_at`

func (r *TemplateRepository) List(ctx context.Context, workspaceID string) ([]resumetemplate.Template, error) {
	items := make([]resumetemplate.Template, 0)
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id,workspace_id,name,kind,format,description,source_name,entry_file,declared_media_type,object_id,
            COALESCE(preview_object_id,''),'',state,COALESCE(job_id,''),COALESCE(error_code,''),COALESCE(error_message,''),created_at,updated_at
            FROM templates WHERE workspace_id=$1 ORDER BY created_at DESC,id`, workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item resumetemplate.Template
			if err := scanTemplate(rows, &item); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *TemplateRepository) Count(ctx context.Context, workspaceID string) (resumetemplate.Counts, error) {
	var counts resumetemplate.Counts
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
            SELECT COUNT(*) FILTER (WHERE state='ready'), COUNT(*)
            FROM templates WHERE workspace_id=$1`, workspaceID).Scan(&counts.Ready, &counts.Custom)
		if err != nil {
			return fmt.Errorf("count templates: %w", err)
		}
		return nil
	})
	return counts, err
}

func (r *TemplateRepository) Get(ctx context.Context, workspaceID, templateID string) (resumetemplate.Template, error) {
	var item resumetemplate.Template
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := scanTemplate(tx.QueryRow(ctx, `SELECT `+templateColumns+` FROM templates WHERE workspace_id=$1 AND id=$2`, workspaceID, templateID), &item)
		if errors.Is(err, pgx.ErrNoRows) {
			return resumetemplate.ErrNotFound
		}
		return err
	})
	return item, err
}

func (r *TemplateRepository) Stage(ctx context.Context, item resumetemplate.Template) error {
	return withWorkspaceTx(ctx, r.pool, item.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO templates
            (id,workspace_id,name,kind,format,description,source_name,entry_file,declared_media_type,object_id,state,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, item.ID, item.WorkspaceID, item.Name, item.Kind, item.Format,
			item.Description, item.SourceName, item.EntryFile, item.DeclaredMediaType, item.ObjectID, item.State, item.CreatedAt, item.UpdatedAt)
		if err != nil {
			return fmt.Errorf("stage template: %w", err)
		}
		return appendAudit(ctx, tx, item.WorkspaceID, "template.stage", "template", item.ID)
	})
}

func (r *TemplateRepository) Restage(ctx context.Context, item resumetemplate.Template) error {
	return withWorkspaceTx(ctx, r.pool, item.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='cancelled',lease_owner=NULL,lease_expires_at=NULL,error_class='template_source_replaced',error_message='The template source was replaced.',updated_at=now()
            WHERE workspace_id=$1 AND state IN ('queued','retry_wait') AND id IN (SELECT job_id FROM templates WHERE workspace_id=$1 AND id=$2)`, item.WorkspaceID, item.ID)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE templates SET format=$3,source_name=$4,entry_file=$5,declared_media_type=$6,object_id=$7,preview_object_id=NULL,content=NULL,state='staged',job_id=NULL,error_code=NULL,error_message=NULL,updated_at=$8
            WHERE workspace_id=$1 AND id=$2`, item.WorkspaceID, item.ID, item.Format, item.SourceName, item.EntryFile, item.DeclaredMediaType, item.ObjectID, item.UpdatedAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return resumetemplate.ErrNotFound
		}
		if err := appendEvent(ctx, tx, item.WorkspaceID, "template.source_replaced.v1", "template", item.ID, map[string]string{"templateId": item.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, item.WorkspaceID, "template.source_replace", "template", item.ID)
	})
}

func (r *TemplateRepository) Queue(ctx context.Context, workspaceID, templateID string, job workqueue.Job) (resumetemplate.Template, error) {
	var item resumetemplate.Template
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := scanTemplate(tx.QueryRow(ctx, `SELECT `+templateColumns+` FROM templates WHERE workspace_id=$1 AND id=$2 FOR UPDATE`, workspaceID, templateID), &item)
		if errors.Is(err, pgx.ErrNoRows) {
			return resumetemplate.ErrNotFound
		}
		if err != nil {
			return err
		}
		if item.State != "staged" {
			if item.JobID != "" {
				return nil
			}
			return resumetemplate.ErrState
		}
		job.IdempotencyKey = item.ObjectID
		job.Payload = resumetemplatePayload(item)
		_, err = tx.Exec(ctx, `INSERT INTO durable_jobs (id,workspace_id,kind,idempotency_key,payload,max_attempts,available_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (workspace_id,kind,idempotency_key) DO NOTHING`, job.ID, job.WorkspaceID, job.Kind, job.IdempotencyKey, job.Payload, job.MaxAttempts, job.AvailableAt)
		if err != nil {
			return err
		}
		err = scanTemplate(tx.QueryRow(ctx, `UPDATE templates SET state='queued',job_id=(SELECT id FROM durable_jobs WHERE workspace_id=$1 AND kind=$2 AND idempotency_key=$3),updated_at=now()
            WHERE workspace_id=$1 AND id=$4 RETURNING `+templateColumns, workspaceID, job.Kind, job.IdempotencyKey, templateID), &item)
		if err != nil {
			return err
		}
		return appendEvent(ctx, tx, workspaceID, "template.uploaded.v1", "template", item.ID, map[string]string{"templateId": item.ID, "jobId": item.JobID})
	})
	return item, err
}

func resumetemplatePayload(item resumetemplate.Template) []byte {
	payload, _ := json.Marshal(resumetemplate.ExtractPayload{TemplateID: item.ID, ObjectID: item.ObjectID, SourceName: item.SourceName, EntryFile: item.EntryFile})
	return payload
}

func (r *TemplateRepository) Update(ctx context.Context, item resumetemplate.Template) (resumetemplate.Template, error) {
	err := withWorkspaceTx(ctx, r.pool, item.WorkspaceID, func(tx pgx.Tx) error {
		err := scanTemplate(tx.QueryRow(ctx, `UPDATE templates SET name=$3,kind=$4,description=$5,updated_at=$6 WHERE workspace_id=$1 AND id=$2 RETURNING `+templateColumns,
			item.WorkspaceID, item.ID, item.Name, item.Kind, item.Description, item.UpdatedAt), &item)
		if errors.Is(err, pgx.ErrNoRows) {
			return resumetemplate.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, item.WorkspaceID, "template.updated.v1", "template", item.ID, map[string]string{"templateId": item.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, item.WorkspaceID, "template.update", "template", item.ID)
	})
	return item, err
}

func (r *TemplateRepository) Delete(ctx context.Context, workspaceID, templateID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='cancelled',lease_owner=NULL,lease_expires_at=NULL,error_class='template_deleted',error_message='The template was deleted before extraction completed.',updated_at=now()
            WHERE workspace_id=$1 AND state IN ('queued','retry_wait') AND id IN (SELECT job_id FROM templates WHERE workspace_id=$1 AND id=$2)`, workspaceID, templateID)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM templates WHERE workspace_id=$1 AND id=$2`, workspaceID, templateID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return resumetemplate.ErrNotFound
		}
		if err := appendEvent(ctx, tx, workspaceID, "template.deleted.v1", "template", templateID, map[string]string{"templateId": templateID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "template.delete", "template", templateID)
	})
}

func (r *TemplateRepository) StoreExtraction(ctx context.Context, job workqueue.Job, templateID, content, previewObjectID string) (bool, error) {
	processed := false
	err := withWorkspaceTx(ctx, r.pool, job.WorkspaceID, func(tx pgx.Tx) error {
		var found string
		err := tx.QueryRow(ctx, `SELECT id FROM templates WHERE workspace_id=$1 AND id=$2 AND job_id=$3 FOR UPDATE`, job.WorkspaceID, templateID, job.ID).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return resumetemplate.ErrNotFound
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO inbox_messages (consumer,message_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, templateConsumer, job.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		tag, err = tx.Exec(ctx, `UPDATE templates SET state='ready',content=$3,preview_object_id=$4,error_code=NULL,error_message=NULL,updated_at=now() WHERE workspace_id=$1 AND id=$2 AND job_id=$5 AND state='queued'`, job.WorkspaceID, templateID, content, previewObjectID, job.ID)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete template: state changed")
		}
		tag, err = tx.Exec(ctx, `UPDATE durable_jobs SET state='succeeded',lease_owner=NULL,lease_expires_at=NULL,updated_at=now() WHERE id=$1 AND lease_owner=$2 AND state='running'`, job.ID, job.LeaseOwner)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete template job: lease lost")
		}
		if err := appendAudit(ctx, tx, job.WorkspaceID, "template.extract", "template", templateID); err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, job.WorkspaceID, "template.ready.v1", "template", templateID, map[string]string{"templateId": templateID}); err != nil {
			return err
		}
		processed = true
		return nil
	})
	return processed, err
}

func (r *TemplateRepository) RecordFailure(ctx context.Context, job workqueue.Job, state, code, message string) error {
	return withWorkspaceTx(ctx, r.pool, job.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE templates SET state=$3,error_code=$4,error_message=$5,updated_at=now() WHERE workspace_id=$1 AND job_id=$2`, job.WorkspaceID, job.ID, state, code, message)
		return err
	})
}

func scanTemplate(row scanner, item *resumetemplate.Template) error {
	return row.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Kind, &item.Format, &item.Description, &item.SourceName, &item.EntryFile, &item.DeclaredMediaType, &item.ObjectID, &item.PreviewObjectID, &item.Content, &item.State, &item.JobID, &item.ErrorCode, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt)
}
