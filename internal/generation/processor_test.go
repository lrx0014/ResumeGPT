package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

type processorRepository struct {
	run       Run
	stages    []string
	completed bool
	objectID  string
	review    string
	steps     []Step
}

func (r *processorRepository) Create(context.Context, Run, workqueue.Job) (Run, error) {
	panic("not used")
}
func (r *processorRepository) List(context.Context, string) ([]Run, error)        { panic("not used") }
func (r *processorRepository) Get(context.Context, string, string) (Run, error)   { return r.run, nil }
func (r *processorRepository) Delete(context.Context, string, string) error       { panic("not used") }
func (r *processorRepository) Reconfigure(context.Context, Run) (Run, error)      { panic("not used") }
func (r *processorRepository) Retry(context.Context, string, string) (Run, error) { panic("not used") }
func (r *processorRepository) Revise(context.Context, string, string, string) (Run, error) {
	panic("not used")
}
func (r *processorRepository) RecordStep(_ context.Context, _ workqueue.Job, step Step) error {
	r.steps = append(r.steps, step)
	return nil
}
func (r *processorRepository) ListSteps(context.Context, string, string) ([]Step, error) {
	return r.steps, nil
}
func (r *processorRepository) GetStep(context.Context, string, string, string) (Step, error) {
	panic("not used")
}
func (r *processorRepository) SetStage(_ context.Context, _ workqueue.Job, stage, _, _ string, _ int) error {
	r.stages = append(r.stages, stage)
	return nil
}

func TestProcessorReusesPersistedDraftOnRetry(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "\\documentclass{article}\\begin{document}Sample\\end{document}"})
	repository := &processorRepository{run: Run{ID: "gen_retry", WorkspaceID: "ws_test", Draft: "# Saved draft", Writer: ModelChoice{ConnectionID: "writer", Model: "writer-model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "renderer-model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	gateway := &processorGateway{responses: []string{"\\documentclass{article}\\begin{document}Draft\\end{document}", `{"approved":true,"feedback":"Layout is balanced."}`}}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: &processorBlobs{}, Documents: processorDocuments{}, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_retry"})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 2, MaxAttempts: 3})
	if !repository.completed || gateway.calls != 2 {
		t.Fatalf("retry did not reuse draft: completed=%v calls=%d", repository.completed, gateway.calls)
	}
	if len(repository.stages) == 0 || repository.stages[0] != "rendering" {
		t.Fatalf("stages = %#v, want rendering first", repository.stages)
	}
}
func (r *processorRepository) Complete(_ context.Context, _ workqueue.Job, _, _, review, objectID string, _ int) error {
	r.completed = true
	r.objectID = objectID
	r.review = review
	return nil
}
func (r *processorRepository) Fail(context.Context, workqueue.Job, string, string) error { return nil }

type processorSettings struct{}

func (processorSettings) RuntimeConnection(_ context.Context, _ string, id string) (settings.RuntimeConnection, error) {
	return settings.RuntimeConnection{Connection: settings.LLMConnection{ID: id, Provider: "ollama", BaseURL: "http://ollama"}}, nil
}

type processorGateway struct {
	responses []string
	calls     int
	failureAt map[int]error
	prompts   []string
}

func (g *processorGateway) Complete(_ context.Context, _ settings.RuntimeConnection, _, _, prompt string, _ []string, _ int) (string, error) {
	g.prompts = append(g.prompts, prompt)
	if err := g.failureAt[g.calls]; err != nil {
		g.calls++
		return "", err
	}
	result := g.responses[g.calls]
	g.calls++
	return result, nil
}

func TestProcessorAppliesFollowUpPromptToCurrentSource(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "template"})
	repository := &processorRepository{run: Run{ID: "gen_revision", WorkspaceID: "ws_test", Draft: "# Grounded draft", RenderedSource: "\\documentclass{article}\\begin{document}Current\\end{document}", Writer: ModelChoice{ConnectionID: "writer", Model: "writer-model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "renderer-model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	gateway := &processorGateway{responses: []string{"\\documentclass{article}\\begin{document}More compact\\end{document}", `{"approved":true,"feedback":"The revision is balanced."}`}}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: &processorBlobs{}, Documents: processorDocuments{}, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_revision", RevisionPrompt: "Make the experience section more compact."})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 1, MaxAttempts: 3})
	if !repository.completed || gateway.calls != 2 || !strings.Contains(gateway.prompts[0], "Make the experience section more compact.") || !strings.Contains(gateway.prompts[0], "Current") {
		t.Fatalf("revision did not use prompt and current source: repository=%#v prompts=%#v", repository, gateway.prompts)
	}
}

func TestProcessorCompletesPDFWhenReviewerDoesNotSupportImages(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "\\documentclass{article}\\begin{document}Sample\\end{document}"})
	repository := &processorRepository{run: Run{ID: "gen_no_vision", WorkspaceID: "ws_test", Writer: ModelChoice{ConnectionID: "writer", Model: "model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "text-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	gateway := &processorGateway{responses: []string{"# Draft", "\\documentclass{article}\\begin{document}Draft\\end{document}"}, failureAt: map[int]error{2: ErrVisionUnsupported}}
	blobs := &processorBlobs{}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: blobs, Documents: processorDocuments{}, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_no_vision"})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 1, MaxAttempts: 3})
	if !repository.completed || repository.objectID == "" || !bytes.Equal(blobs.stored, []byte("%PDF-result")) {
		t.Fatalf("PDF was not completed: %#v", repository)
	}
	if !strings.HasPrefix(repository.review, "Visual QA skipped:") {
		t.Fatalf("review = %q, want visual QA warning", repository.review)
	}
}

func TestProcessorUsesSafeTemplateFallbackForIncompleteModelOutput(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "\\documentclass{article}\\begin{document}Sample\\end{document}"})
	repository := &processorRepository{run: Run{ID: "gen_fallback", WorkspaceID: "ws_test", Writer: ModelChoice{ConnectionID: "writer", Model: "model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "small-model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	gateway := &processorGateway{responses: []string{
		"# Draft\n\n## Experience\n- Built reliable systems.",
		"incomplete LaTeX",
		"still incomplete",
		"not a document",
		"missing document markers",
		`{"approved":true,"feedback":"The basic layout is readable."}`,
	}}
	blobs := &processorBlobs{}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: blobs, Documents: processorDocuments{}, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_fallback"})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 1, MaxAttempts: 3})
	if !repository.completed || !strings.HasPrefix(repository.review, "Template fallback used:") {
		t.Fatalf("template fallback did not complete with a warning: %#v", repository)
	}
}

func TestProcessorUsesSafeTemplateFallbackWhenModelLatexDoesNotCompile(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "\\documentclass{article}\\begin{document}Sample\\end{document}"})
	repository := &processorRepository{run: Run{ID: "gen_compile_fallback", WorkspaceID: "ws_test", Writer: ModelChoice{ConnectionID: "writer", Model: "model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "small-model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	candidate := "\\documentclass{article}\\begin{document}Invalid package use\\end{document}"
	gateway := &processorGateway{responses: []string{"# Draft", candidate, candidate, candidate, candidate, `{"approved":true,"feedback":"The basic layout is readable."}`}}
	documents := &fallbackDocuments{}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: &processorBlobs{}, Documents: documents, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_compile_fallback"})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 1, MaxAttempts: 3})
	if !repository.completed || documents.calls != 5 || !strings.Contains(repository.review, "exhausted its repair attempts") {
		t.Fatalf("compile fallback did not complete: repository=%#v calls=%d", repository, documents.calls)
	}
	var diagnosticFound bool
	for _, step := range repository.steps {
		if step.Kind == "system_warning" && step.Content == candidate && strings.Contains(step.Feedback, "template_compile_failed") {
			diagnosticFound = true
		}
	}
	if !diagnosticFound {
		t.Fatalf("failed candidate and compiler diagnostic were not persisted: %#v", repository.steps)
	}
}

func TestProcessorCanResumeDirectlyFromSafeTemplateFallback(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "template"})
	repository := &processorRepository{run: Run{ID: "gen_resume_fallback", WorkspaceID: "ws_test", Draft: "# Saved draft", Writer: ModelChoice{ConnectionID: "writer", Model: "model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "small-model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	gateway := &processorGateway{responses: []string{`{"approved":true,"feedback":"The basic layout is readable."}`}}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: &processorBlobs{}, Documents: processorDocuments{}, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_resume_fallback", UseFallback: true})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 1, MaxAttempts: 3})
	if !repository.completed || gateway.calls != 1 || !strings.HasPrefix(repository.review, "Template fallback used:") {
		t.Fatalf("direct fallback resume failed: repository=%#v calls=%d", repository, gateway.calls)
	}
}

type processorDocuments struct{}

func (processorDocuments) PreviewTemplate(context.Context, string, string, io.Reader, int64) ([]byte, error) {
	return []byte("%PDF-result"), nil
}
func (processorDocuments) PDFPages(context.Context, []byte) (document.PDFPages, error) {
	return document.PDFPages{PageCount: 1, Images: []string{"image"}}, nil
}

type fallbackDocuments struct{ calls int }

func (d *fallbackDocuments) PreviewTemplate(context.Context, string, string, io.Reader, int64) ([]byte, error) {
	d.calls++
	if d.calls <= 4 {
		return nil, errors.New("model LaTeX did not compile")
	}
	return []byte("%PDF-fallback"), nil
}
func (*fallbackDocuments) PDFPages(context.Context, []byte) (document.PDFPages, error) {
	return document.PDFPages{PageCount: 1, Images: []string{"image"}}, nil
}

type processorBlobs struct{ stored []byte }

func (b *processorBlobs) Open(context.Context, string, string) (blobstore.Object, error) {
	return blobstore.Object{}, nil
}
func (b *processorBlobs) Put(_ context.Context, _, _, _ string, body io.Reader, _ int64) error {
	b.stored, _ = io.ReadAll(body)
	return nil
}
func TestProcessorCompletesWrittenRenderedAndReviewedPDF(t *testing.T) {
	p, _ := json.Marshal(profile.Profile{Content: "Built reliable Go services."})
	j, _ := json.Marshal(job.Job{Title: "Backend Engineer", Company: "Example", Description: "Build Go systems."})
	tpl, _ := json.Marshal(resumetemplate.Template{BuiltIn: true, Format: "latex", Kind: "resume", Content: "\\documentclass{article}\\begin{document}Sample\\end{document}"})
	repository := &processorRepository{run: Run{ID: "gen_test", WorkspaceID: "ws_test", Writer: ModelChoice{ConnectionID: "writer", Model: "writer-model"}, Renderer: ModelChoice{ConnectionID: "renderer", Model: "renderer-model"}, Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page", ProfileSnapshot: p, OpportunitySnapshot: j, TemplateSnapshot: tpl}}
	gateway := &processorGateway{responses: []string{"# Draft", "\\documentclass{article}\\begin{document}Draft\\end{document}", `{"approved":true,"feedback":"Layout is balanced."}`}}
	blobs := &processorBlobs{}
	processor := Processor{Repository: repository, Settings: processorSettings{}, Blobs: blobs, Documents: processorDocuments{}, Gateway: gateway, WorkerID: "worker", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	payload, _ := json.Marshal(Payload{RunID: "gen_test"})
	processor.handle(context.Background(), workqueue.Job{ID: "task", WorkspaceID: "ws_test", Payload: payload, LeaseOwner: "worker", Attempt: 1, MaxAttempts: 3})
	if !repository.completed || repository.objectID == "" || !bytes.Equal(blobs.stored, []byte("%PDF-result")) {
		t.Fatalf("generation not completed: %#v", repository)
	}
	if gateway.calls != 3 {
		t.Fatalf("gateway calls = %d, want 3", gateway.calls)
	}
	want := []string{"writing", "rendering", "reviewing", "finalizing"}
	if len(repository.stages) != len(want) {
		t.Fatalf("stages = %#v", repository.stages)
	}
	for i := range want {
		if repository.stages[i] != want[i] {
			t.Fatalf("stages = %#v", repository.stages)
		}
	}
}
