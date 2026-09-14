package template_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

type signer struct{}

type previewBlobs struct{ objects map[string][]byte }

func (b *previewBlobs) Open(_ context.Context, workspaceID, objectID string) (blobstore.Object, error) {
	value, ok := b.objects[workspaceID+"/"+objectID]
	if !ok {
		return blobstore.Object{}, errors.New("object not found")
	}
	return blobstore.Object{Body: io.NopCloser(bytes.NewReader(value)), Size: int64(len(value))}, nil
}

func (b *previewBlobs) Put(_ context.Context, workspaceID, objectID, _ string, body io.Reader, _ int64) error {
	value, err := io.ReadAll(body)
	if err == nil {
		b.objects[workspaceID+"/"+objectID] = value
	}
	return err
}

type countingPreviewer struct{ calls int }

func (p *countingPreviewer) Preview(context.Context, string, io.Reader, int64) ([]byte, error) {
	p.calls++
	return []byte("%PDF-cached-preview"), nil
}

func (p *countingPreviewer) PreviewTemplate(ctx context.Context, name, _ string, body io.Reader, size int64) ([]byte, error) {
	return p.Preview(ctx, name, body, size)
}

func (signer) EnsureBucket(context.Context) error { return nil }
func (signer) PresignUpload(_ context.Context, _, objectID, contentType string, _ time.Duration) (blobstore.SignedURL, error) {
	return blobstore.SignedURL{ObjectID: objectID, URL: "https://upload.example", Headers: map[string]string{"Content-Type": contentType}}, nil
}
func (signer) PresignDownload(_ context.Context, _, objectID string, _ time.Duration) (blobstore.SignedURL, error) {
	return blobstore.SignedURL{ObjectID: objectID, URL: "https://download.example"}, nil
}

func TestBuiltInResumeRetainsAttributionAndSource(t *testing.T) {
	service := resumetemplate.NewService(memory.NewTemplateRepository(), signer{}, true)
	items, err := service.List(context.Background(), "ws_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != resumetemplate.DefaultResumeID || items[0].AuthorName != "Nanu Panchamurthy" || items[0].License != "MIT" || items[0].SourceURL != "https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs" {
		t.Fatalf("unexpected default template: %#v", items)
	}
	detail, err := service.Get(context.Background(), "ws_test", resumetemplate.DefaultResumeID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Content == "" {
		t.Fatal("built-in template source is empty")
	}
	encoded, err := json.Marshal(items[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || strings.Contains(string(encoded), "0001-01-01") {
		t.Fatalf("built-in JSON contains zero timestamps: %s", encoded)
	}
	if _, err := service.Update(context.Background(), "ws_test", detail.ID, resumetemplate.UpdateInput{Name: "Changed", Kind: "resume"}); !errors.Is(err, resumetemplate.ErrBuiltIn) {
		t.Fatalf("update error = %v, want built-in error", err)
	}
}

func TestStageValidatesAndStoresSimpleTemplateMetadata(t *testing.T) {
	repository := memory.NewTemplateRepository()
	service := resumetemplate.NewService(repository, signer{}, true)
	result, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{Name: "Cover letter", Kind: "cover_letter", Description: "Simple letter", SourceName: "letter.docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Template.Format != "docx" || result.Template.State != "staged" || result.Target.ObjectID != result.Template.ObjectID {
		t.Fatalf("unexpected staged template: %#v", result)
	}
	if _, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{Name: "Bad", Kind: "resume", SourceName: "template.pdf", ContentType: "application/pdf"}); !errors.Is(err, resumetemplate.ErrInvalid) {
		t.Fatalf("unsupported extension error = %v", err)
	}
}

func TestReplaceSourceRestagesTemplateWithoutChangingMetadata(t *testing.T) {
	repository := memory.NewTemplateRepository()
	service := resumetemplate.NewService(repository, signer{}, true)
	staged, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{Name: "Resume", Kind: "resume", Description: "Primary", SourceName: "old.tex", ContentType: "application/x-tex"})
	if err != nil {
		t.Fatal(err)
	}
	replaced, err := service.ReplaceSource(context.Background(), "ws_test", staged.Template.ID, resumetemplate.SourceInput{SourceName: "new.docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Template.Name != "Resume" || replaced.Template.Description != "Primary" || replaced.Template.Format != "docx" || replaced.Template.SourceName != "new.docx" || replaced.Template.State != "staged" || replaced.Template.ObjectID == staged.Template.ObjectID {
		t.Fatalf("unexpected replacement: %#v", replaced.Template)
	}
}

func TestStageSupportsLatexZipWithOptionalEntryFile(t *testing.T) {
	service := resumetemplate.NewService(memory.NewTemplateRepository(), signer{}, true)
	result, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{Name: "Multi-file resume", Kind: "resume", SourceName: "resume.zip", ContentType: "application/zip", EntryFile: "src/resume.tex"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Template.Format != "latex" || result.Template.EntryFile != "src/resume.tex" {
		t.Fatalf("unexpected ZIP template: %#v", result.Template)
	}
	if _, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{Name: "Unsafe", Kind: "resume", SourceName: "resume.zip", ContentType: "application/zip", EntryFile: "../main.tex"}); !errors.Is(err, resumetemplate.ErrInvalid) {
		t.Fatalf("unsafe entry error = %v", err)
	}
	if _, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{Name: "Wrong", Kind: "resume", SourceName: "resume.docx", ContentType: "application/octet-stream", EntryFile: "main.tex"}); !errors.Is(err, resumetemplate.ErrInvalid) {
		t.Fatalf("Word entry error = %v", err)
	}
}

func TestDefaultPreviewIsGeneratedOnceAndThenLoadedFromCache(t *testing.T) {
	blobs := &previewBlobs{objects: make(map[string][]byte)}
	previewer := &countingPreviewer{}
	firstID, err := resumetemplate.PrepareDefaultPreview(context.Background(), blobs, previewer)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := resumetemplate.PrepareDefaultPreview(context.Background(), blobs, previewer)
	if err != nil {
		t.Fatal(err)
	}
	if firstID != secondID || previewer.calls != 1 {
		t.Fatalf("preview IDs = %q/%q, render calls = %d", firstID, secondID, previewer.calls)
	}
	service := resumetemplate.NewService(memory.NewTemplateRepository(), signer{}, true)
	service.ConfigurePreview(blobs, firstID)
	preview, err := service.Preview(context.Background(), "ws_test", resumetemplate.DefaultResumeID)
	if err != nil || string(preview) != "%PDF-cached-preview" || previewer.calls != 1 {
		t.Fatalf("preview = %q, error = %v, render calls = %d", preview, err, previewer.calls)
	}
}

func TestUploadsCanBeDisabledWithoutHidingBuiltInTemplate(t *testing.T) {
	service := resumetemplate.NewService(memory.NewTemplateRepository(), signer{}, false)
	if service.UploadsEnabled() {
		t.Fatal("uploads unexpectedly enabled")
	}
	if _, err := service.Stage(context.Background(), "ws_test", resumetemplate.StageInput{}); !errors.Is(err, resumetemplate.ErrState) {
		t.Fatalf("stage error = %v", err)
	}
	items, err := service.List(context.Background(), "ws_test")
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %#v, error = %v", items, err)
	}
}
