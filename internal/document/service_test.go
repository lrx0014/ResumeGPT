package document_test

import (
	"context"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type repositoryStub struct {
	staged document.Upload
	job    workqueue.Job
}

func (r *repositoryStub) Stage(_ context.Context, upload document.Upload) error {
	r.staged = upload
	return nil
}

func (r *repositoryStub) Queue(_ context.Context, workspaceID, profileID, uploadID string, job workqueue.Job) (document.Upload, error) {
	r.job = job
	upload := r.staged
	if upload.WorkspaceID != workspaceID || upload.ProfileID != profileID || upload.ID != uploadID {
		return document.Upload{}, document.ErrNotFound
	}
	upload.State = "queued"
	upload.JobID = job.ID
	return upload, nil
}

func (r *repositoryStub) Get(context.Context, string, string, string) (document.Upload, error) {
	return r.staged, nil
}

func (r *repositoryStub) DownloadAllowed(context.Context, string, string) (bool, error) {
	return r.staged.State == "ready", nil
}

func (r *repositoryStub) StoreExtraction(context.Context, workqueue.Job, string, string) (bool, error) {
	return true, nil
}

func (r *repositoryStub) RecordFailure(context.Context, workqueue.Job, string, string, string) error {
	return nil
}

func TestStagesAndQueuesSupportedDocument(t *testing.T) {
	repository := &repositoryStub{}
	service := document.NewService(repository, memory.BlobSigner{})
	staged, err := service.Stage(context.Background(), "ws_test", "prof_test", document.StageInput{Name: "resume.PDF", ContentType: "application/pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if staged.Upload.State != "staged" || staged.Target.ObjectID != staged.Upload.ObjectID {
		t.Fatalf("unexpected staged upload: %#v", staged)
	}
	queued, err := service.Queue(context.Background(), "ws_test", "prof_test", staged.Upload.ID)
	if err != nil {
		t.Fatal(err)
	}
	if queued.State != "queued" || repository.job.Kind != document.ExtractJobKind || repository.job.IdempotencyKey != staged.Upload.ID {
		t.Fatalf("unexpected queued upload: %#v, job: %#v", queued, repository.job)
	}
}

func TestRejectsUnsupportedDocumentExtension(t *testing.T) {
	service := document.NewService(&repositoryStub{}, memory.BlobSigner{})
	if _, err := service.Stage(context.Background(), "ws_test", "prof_test", document.StageInput{Name: "resume.zip", ContentType: "application/zip"}); err != document.ErrInvalid {
		t.Fatalf("error = %v, want %v", err, document.ErrInvalid)
	}
}
