package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type Processor struct {
	Queue      ProcessorQueue
	Repository Repository
	Blobs      blobstore.Reader
	Extractor  *HTTPExtractor
	WorkerID   string
	Logger     *slog.Logger
}

type ProcessorQueue interface {
	ClaimKind(context.Context, string, string, time.Duration) (workqueue.Job, error)
	Retry(context.Context, workqueue.Job, string, string, string, time.Duration) error
	Fail(context.Context, string, string, string, string) error
}

func (p *Processor) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		job, err := p.Queue.ClaimKind(ctx, p.WorkerID, ExtractJobKind, 5*time.Minute)
		if err == nil {
			jobContext, cancel := context.WithTimeout(ctx, 4*time.Minute+30*time.Second)
			p.handle(jobContext, job)
			cancel()
		} else if !errors.Is(err, workqueue.ErrEmpty) {
			p.Logger.Error("claim document job", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (p *Processor) handle(ctx context.Context, job workqueue.Job) {
	var payload ExtractPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil || payload.UploadID == "" || payload.ObjectID == "" || payload.Name == "" {
		p.fail(ctx, job, payload.UploadID, "failed", "invalid_job_payload", "The document job payload is invalid.", false)
		return
	}
	object, err := p.Blobs.Open(ctx, job.WorkspaceID, payload.ObjectID)
	if err != nil {
		p.fail(ctx, job, payload.UploadID, "queued", "object_unavailable", "The staged document is not available yet.", true)
		return
	}
	defer object.Body.Close()
	result, err := p.Extractor.Extract(ctx, payload.Name, object.Body, object.Size)
	if err != nil {
		var failure *ExtractionFailure
		if !errors.As(err, &failure) {
			failure = &ExtractionFailure{Code: "extraction_failed", Message: "Document extraction failed unexpectedly.", Retryable: true}
		}
		state := "needs_user_action"
		if failure.Retryable {
			state = "queued"
		}
		if failure.Code == "malware_detected" {
			state = "security_quarantine"
		}
		p.fail(ctx, job, payload.UploadID, state, failure.Code, failure.Message, failure.Retryable)
		return
	}
	if result.MalwareStatus != "clean" {
		p.fail(ctx, job, payload.UploadID, "security_quarantine", "scan_not_clean", "The document was not released by malware scanning.", false)
		return
	}
	paragraphs := make([]string, 0, len(result.Segments))
	for _, segment := range result.Segments {
		text := strings.TrimSpace(segment.Text)
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}
	extractedText := strings.Join(paragraphs, "\n\n")
	if extractedText == "" || len(extractedText) > 1024*1024 {
		p.fail(ctx, job, payload.UploadID, "failed", "invalid_extraction_result", "The extracted document exceeds the profile text limit.", false)
		return
	}
	processed, err := p.Repository.StoreExtraction(ctx, job, payload.UploadID, extractedText)
	if errors.Is(err, ErrNotFound) {
		p.fail(ctx, job, payload.UploadID, "failed", "upload_deleted", "The profile or document upload was deleted during extraction.", false)
		return
	}
	if err != nil {
		p.fail(ctx, job, payload.UploadID, "queued", "persistence_failed", "Could not store the extracted text.", true)
		return
	}
	if processed {
		p.Logger.Info("document extraction completed", "job_id", job.ID, "upload_id", payload.UploadID)
	}
}

func (p *Processor) fail(ctx context.Context, job workqueue.Job, uploadID, state, code, message string, retryable bool) {
	if retryable && job.Attempt >= job.MaxAttempts {
		retryable = false
		if state == "queued" {
			state = "needs_user_action"
		}
	}
	if err := p.Repository.RecordFailure(ctx, job, state, code, message); err != nil {
		p.Logger.Error("record document failure", "job_id", job.ID, "error", err)
	}
	var err error
	if retryable {
		delay := time.Duration(job.Attempt*job.Attempt) * 5 * time.Second
		err = p.Queue.Retry(ctx, job, p.WorkerID, code, message, delay)
	} else {
		err = p.Queue.Fail(ctx, job.ID, p.WorkerID, code, message)
	}
	if err != nil {
		p.Logger.Error("finalize failed document job", "job_id", job.ID, "error", err)
	}
	p.Logger.Warn("document extraction failed", "job_id", job.ID, "code", code, "retryable", retryable, "error", fmt.Errorf("%s", message))
}
