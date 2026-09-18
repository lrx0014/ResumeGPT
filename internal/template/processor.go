package template

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type Processor struct {
	Queue      workqueue.ClaimQueue
	Repository Repository
	Blobs      TemplateBlobs
	Extractor  *document.HTTPExtractor
	Previewer  Previewer
	WorkerID   string
	Logger     *slog.Logger
}

type TemplateBlobs interface {
	blobstore.Reader
	blobstore.Writer
}

func (p *Processor) Run(ctx context.Context) error {
	return workqueue.RunLoop(ctx, p.Queue, p.WorkerID, ExtractJobKind, 5*time.Minute, 4*time.Minute+30*time.Second,
		p.Logger, "claim template job", p.handle)
}

func (p *Processor) handle(ctx context.Context, job workqueue.Job) {
	var payload ExtractPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil || payload.TemplateID == "" || payload.ObjectID == "" || payload.SourceName == "" {
		p.fail(ctx, job, "failed", "invalid_job_payload", "The template job payload is invalid.", false)
		return
	}
	object, err := p.Blobs.Open(ctx, job.WorkspaceID, payload.ObjectID)
	if err != nil {
		p.fail(ctx, job, "queued", "object_unavailable", "The staged template is not available yet.", true)
		return
	}
	defer object.Body.Close()
	if object.Size > maxTemplateBytes {
		p.fail(ctx, job, "needs_user_action", "template_too_large", "Template files must be 5 MiB or smaller.", false)
		return
	}
	result, err := p.Extractor.ExtractTemplate(ctx, payload.SourceName, payload.EntryFile, object.Body, object.Size)
	_ = object.Body.Close()
	if err != nil {
		var failure *document.ExtractionFailure
		if !errors.As(err, &failure) {
			failure = &document.ExtractionFailure{Code: "extraction_failed", Message: "Template extraction failed unexpectedly.", Retryable: true}
		}
		state := "needs_user_action"
		if failure.Retryable {
			state = "queued"
		}
		if failure.Code == "malware_detected" {
			state = "security_quarantine"
		}
		p.fail(ctx, job, state, failure.Code, failure.Message, failure.Retryable)
		return
	}
	if result.MalwareStatus != "clean" {
		p.fail(ctx, job, "security_quarantine", "scan_not_clean", "The template was not released by malware scanning.", false)
		return
	}
	parts := make([]string, 0, len(result.Segments))
	for _, segment := range result.Segments {
		if value := strings.TrimSpace(segment.Text); value != "" {
			parts = append(parts, value)
		}
	}
	content := strings.Join(parts, "\n\n")
	if content == "" || len(content) > 1024*1024 {
		p.fail(ctx, job, "needs_user_action", "invalid_template_text", "The extracted template text is empty or exceeds 1 MiB.", false)
		return
	}
	previewSource, err := p.Blobs.Open(ctx, job.WorkspaceID, payload.ObjectID)
	if err != nil {
		p.fail(ctx, job, "queued", "object_unavailable", "The template source is not available for PDF preview generation.", true)
		return
	}
	preview, previewErr := p.Previewer.PreviewTemplate(ctx, payload.SourceName, payload.EntryFile, previewSource.Body, previewSource.Size)
	_ = previewSource.Body.Close()
	if previewErr != nil {
		var failure *document.ExtractionFailure
		if !errors.As(previewErr, &failure) {
			failure = &document.ExtractionFailure{Code: "preview_failed", Message: "PDF preview generation failed unexpectedly.", Retryable: true}
		}
		state := "needs_user_action"
		if failure.Retryable {
			state = "queued"
		}
		p.fail(ctx, job, state, failure.Code, failure.Message, failure.Retryable)
		return
	}
	previewObjectID := previewObjectID(payload.ObjectID)
	if err := p.Blobs.Put(ctx, job.WorkspaceID, previewObjectID, "application/pdf", bytes.NewReader(preview), int64(len(preview))); err != nil {
		p.fail(ctx, job, "queued", "preview_persistence_failed", "Could not store the generated PDF preview.", true)
		return
	}
	processed, err := p.Repository.StoreExtraction(ctx, job, payload.TemplateID, content, previewObjectID)
	if errors.Is(err, ErrNotFound) {
		p.fail(ctx, job, "failed", "template_deleted", "The template was deleted during extraction.", false)
		return
	}
	if err != nil {
		p.fail(ctx, job, "queued", "persistence_failed", "Could not store the extracted template text.", true)
		return
	}
	if processed {
		p.Logger.Info("template extraction completed", "job_id", job.ID, "template_id", payload.TemplateID)
	}
}

func previewObjectID(sourceObjectID string) string {
	return "obj_template_preview_" + strings.TrimPrefix(sourceObjectID, "obj_")
}

func (p *Processor) fail(ctx context.Context, job workqueue.Job, state, code, message string, retryable bool) {
	if retryable && job.Attempt >= job.MaxAttempts {
		retryable = false
		if state == "queued" {
			state = "needs_user_action"
		}
	}
	if err := p.Repository.RecordFailure(ctx, job, state, code, message); err != nil {
		p.Logger.Error("record template failure", "job_id", job.ID, "error", err)
	}
	var err error
	if retryable {
		err = p.Queue.Retry(ctx, job, p.WorkerID, code, message, time.Duration(job.Attempt*job.Attempt)*5*time.Second)
	} else {
		err = p.Queue.Fail(ctx, job.ID, p.WorkerID, code, message)
	}
	if err != nil {
		p.Logger.Error("finalize failed template job", "job_id", job.ID, "error", err)
	}
	p.Logger.Warn("template extraction failed", "job_id", job.ID, "code", code, "retryable", retryable, "error", message)
}
