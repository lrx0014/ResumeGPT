package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/prompts"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

const (
	maxRepairs        = 2
	writerMaxTokens   = 4096
	rendererMaxTokens = 6144
	reviewerMaxTokens = 512
)

type RuntimeResolver interface {
	RuntimeConnection(context.Context, string, string) (settings.RuntimeConnection, error)
}

type DocumentRenderer interface {
	PreviewTemplate(context.Context, string, string, io.Reader, int64) ([]byte, error)
	HTMLPDF(context.Context, string) ([]byte, error)
	PDFPages(context.Context, []byte) (document.PDFPages, error)
}

type Processor struct {
	Queue      workqueue.ClaimQueue
	Repository Repository
	Settings   RuntimeResolver
	Blobs      interface {
		blobstore.Reader
		blobstore.Writer
	}
	Documents DocumentRenderer
	Gateway   Gateway
	WorkerID  string
	Logger    *slog.Logger
}

func (p *Processor) Run(ctx context.Context) error {
	return workqueue.RunLoop(ctx, p.Queue, p.WorkerID, JobKind, 45*time.Minute, 44*time.Minute,
		p.Logger, "claim generation job", p.handle)
}

func (p *Processor) handle(ctx context.Context, task workqueue.Job) {
	var payload Payload
	if json.Unmarshal(task.Payload, &payload) != nil || payload.RunID == "" {
		p.fail(ctx, task, "invalid_job_payload", "The generation task is invalid.", false)
		return
	}
	run, err := p.Repository.Get(ctx, task.WorkspaceID, payload.RunID)
	if err != nil {
		p.fail(ctx, task, "generation_missing", "The generation no longer exists.", false)
		return
	}
	var profileValue profile.Profile
	var opportunity job.Job
	var templateValue resumetemplate.Template
	templateSnapshot := bytes.TrimSpace(run.TemplateSnapshot)
	usesTemplate := run.TemplateID != "" || len(templateSnapshot) > 2 && string(templateSnapshot) != "null"
	if json.Unmarshal(run.ProfileSnapshot, &profileValue) != nil || json.Unmarshal(run.OpportunitySnapshot, &opportunity) != nil || usesTemplate && json.Unmarshal(run.TemplateSnapshot, &templateValue) != nil {
		p.fail(ctx, task, "invalid_snapshot", "The saved generation inputs are invalid.", false)
		return
	}
	agentTeam := NewGenerationAgentTeam(p.Settings, p.Gateway, p.Documents, p.Blobs)
	draft := strings.TrimSpace(run.Draft)
	if draft == "" {
		_ = p.Repository.SetStage(ctx, task, "writing", "", "", 0)
		draft, err = agentTeam.Write(ctx, task.WorkspaceID, run, profileValue, opportunity)
		if err != nil {
			if errors.Is(err, ErrAgentConnection) {
				p.fail(ctx, task, "writer_unavailable", "The selected writer provider is unavailable.", false)
				return
			}
			p.fail(ctx, task, "writing_failed", err.Error(), true)
			return
		}
		draft = strings.TrimSpace(draft)
		if err := p.Repository.RecordStep(ctx, task, Step{ID: id.New("step"), Kind: "writer_draft", Content: draft, CreatedAt: time.Now().UTC()}); err != nil {
			p.fail(ctx, task, "step_store_failed", "The writing result could not be saved.", true)
			return
		}
	}
	if draft == "" || len(draft) > 1024*1024 {
		p.fail(ctx, task, "invalid_draft", "The writer returned an empty or oversized draft.", false)
		return
	}
	_ = p.Repository.SetStage(ctx, task, "rendering", draft, "", 0)
	var application TemplateApplyResult
	if !usesTemplate {
		application, err = agentTeam.DesignDocument(ctx, task.WorkspaceID, run, profileValue.AvatarObjectID, draft, payload.RevisionPrompt, payload.UseFallback)
	} else {
		application, err = agentTeam.ApplyTemplate(ctx, task.WorkspaceID, run, templateValue, profileValue.AvatarObjectID, draft, payload.RevisionPrompt, payload.UseFallback)
	}
	if err != nil {
		if errors.Is(err, ErrAgentConnection) {
			p.fail(ctx, task, "renderer_unavailable", "The selected renderer provider is unavailable.", false)
			return
		}
		code := "rendering_failed"
		if payload.RevisionPrompt != "" {
			code = "revision_failed"
		}
		p.fail(ctx, task, code, err.Error(), true)
		return
	}
	source, fallbackReason := application.Source, application.FallbackReason
	var reviewText string
	repairCount := 0
	var pdf []byte
	var artifactObjectID string
	for _, failure := range application.Failures {
		p.recordTemplateFailure(ctx, task, failure.Source, failure.Error, repairCount)
	}
	if application.ValidationError != nil {
		if len(application.Failures) == 0 {
			p.recordTemplateFailure(ctx, task, source, application.ValidationError, repairCount)
		}
		if !usesTemplate {
			source = fallbackHTML(draft)
			reviewText = "The Document Designer could not produce valid HTML within its repair limit. Switching to the safe basic layout."
		} else {
			source = fallbackLatex(draft)
			reviewText = "The Template Applying agent could not produce compilable LaTeX within its repair limit. Switching to the safe basic layout."
		}
		_ = p.Repository.SetStage(ctx, task, "rendering", draft, reviewText, repairCount)
	}
	for {
		if !usesTemplate {
			pdf, err = agentTeam.RenderHTMLPDF(ctx, task.WorkspaceID, profileValue.AvatarObjectID, source)
		} else {
			pdf, err = agentTeam.RenderPDF(ctx, task.WorkspaceID, templateValue, profileValue.AvatarObjectID, source)
		}
		if err != nil {
			if fallbackReason == "" {
				p.recordTemplateFailure(ctx, task, source, err, repairCount)
				if !usesTemplate {
					source = fallbackHTML(draft)
					fallbackReason = "the validated HTML design could not be rendered again while creating the artifact"
					reviewText = "The model-generated HTML failed rendering. Switching to the safe basic layout."
				} else {
					source = fallbackLatex(draft)
					fallbackReason = "the validated template candidate could not be compiled again while creating the artifact"
					reviewText = "The model-generated LaTeX failed compilation. Switching to the safe basic layout."
				}
				_ = p.Repository.SetStage(ctx, task, "rendering", draft, reviewText, repairCount)
				continue
			}
			p.fail(ctx, task, "fallback_render_failed", "The safe basic layout could not be rendered as PDF: "+err.Error(), false)
			return
		}
		artifactObjectID, err = agentTeam.StorePDF(ctx, task.WorkspaceID, pdf)
		if err != nil {
			p.fail(ctx, task, "artifact_store_failed", "The rendered PDF could not be stored.", true)
			return
		}
		if err := p.Repository.RecordStep(ctx, task, Step{ID: id.New("step"), Kind: "rendered_pdf", Content: source, ArtifactObjectID: artifactObjectID, RepairCount: repairCount, CreatedAt: time.Now().UTC()}); err != nil {
			p.fail(ctx, task, "step_store_failed", "The rendered PDF stage could not be saved.", true)
			return
		}
		_ = p.Repository.SetStage(ctx, task, "reviewing", draft, reviewText, repairCount)
		pages, approved, feedback, response, reviewErr := agentTeam.Review(ctx, task.WorkspaceID, run, pdf)
		if reviewErr != nil && !errors.Is(reviewErr, ErrVisionUnsupported) {
			if errors.Is(reviewErr, ErrAgentConnection) {
				p.fail(ctx, task, "reviewer_unavailable", "The selected reviewer provider is unavailable.", false)
				return
			}
			p.fail(ctx, task, "visual_review_failed", reviewErr.Error(), true)
			return
		}
		if expected := expectedPages(run.PageTarget); expected > 0 && pages.PageCount != expected {
			reviewText = fmt.Sprintf("The page target is %d page(s), but the rendered PDF has %d page(s). Adjust content density, spacing, and safe font sizing without removing important evidence.", expected, pages.PageCount)
			_ = p.Repository.RecordStep(ctx, task, Step{ID: id.New("step"), Kind: "reviewer_feedback", Feedback: reviewText, RepairCount: repairCount, CreatedAt: time.Now().UTC()})
			if repairCount >= maxRepairs {
				p.fail(ctx, task, "page_target_failed", reviewText, false)
				return
			}
			repairCount++
			_ = p.Repository.SetStage(ctx, task, "repairing", draft, reviewText, repairCount)
			if !usesTemplate {
				source, err = agentTeam.PolishDesign(ctx, task.WorkspaceID, run, profileValue.AvatarObjectID, draft, source, reviewText)
			} else {
				source, err = agentTeam.Polish(ctx, task.WorkspaceID, run, templateValue, profileValue.AvatarObjectID, draft, source, reviewText)
			}
			if err != nil {
				p.recordTemplateFailure(ctx, task, source, err, repairCount)
				p.fail(ctx, task, "repair_failed", err.Error(), true)
				return
			}
			continue
		}
		if reviewErr != nil {
			if errors.Is(reviewErr, ErrVisionUnsupported) {
				reviewText = "Visual QA skipped: the selected reviewer model does not support image input. The PDF was generated and deterministic page-count checks passed."
				_ = p.Repository.RecordStep(ctx, task, Step{ID: id.New("step"), Kind: "system_warning", Feedback: reviewText, RepairCount: repairCount, CreatedAt: time.Now().UTC()})
				break
			}
		}
		reviewText = feedback
		_ = p.Repository.RecordStep(ctx, task, Step{ID: id.New("step"), Kind: "reviewer_feedback", Content: response, Feedback: feedback, RepairCount: repairCount, CreatedAt: time.Now().UTC()})
		if approved {
			break
		}
		if repairCount >= maxRepairs {
			p.fail(ctx, task, "quality_review_failed", "Visual review still found layout issues after two repairs: "+feedback, false)
			return
		}
		repairCount++
		_ = p.Repository.SetStage(ctx, task, "repairing", draft, reviewText, repairCount)
		if !usesTemplate {
			source, err = agentTeam.PolishDesign(ctx, task.WorkspaceID, run, profileValue.AvatarObjectID, draft, source, reviewText)
		} else {
			source, err = agentTeam.Polish(ctx, task.WorkspaceID, run, templateValue, profileValue.AvatarObjectID, draft, source, reviewText)
		}
		if err != nil {
			p.recordTemplateFailure(ctx, task, source, err, repairCount)
			p.fail(ctx, task, "repair_failed", err.Error(), true)
			return
		}
	}
	if fallbackReason != "" {
		warning := "Document design fallback used: " + fallbackReason + ", so ResumeGPT generated a safe basic layout."
		if usesTemplate {
			warning = "Template fallback used: " + fallbackReason + ", so ResumeGPT generated a safe basic layout instead of the selected template."
		}
		if reviewText != "" {
			reviewText = warning + " " + reviewText
		} else {
			reviewText = warning
		}
		_ = p.Repository.RecordStep(ctx, task, Step{ID: id.New("step"), Kind: "system_warning", Feedback: warning, RepairCount: repairCount, CreatedAt: time.Now().UTC()})
	}
	_ = p.Repository.SetStage(ctx, task, "finalizing", draft, reviewText, repairCount)
	if err := p.Repository.Complete(ctx, task, source, draft, reviewText, artifactObjectID, repairCount); err != nil {
		p.Logger.Error("complete generation", "run_id", run.ID, "error", err)
		return
	}
	p.Logger.Info("generation completed", "run_id", run.ID, "repairs", repairCount)
}

func (p *Processor) recordTemplateFailure(ctx context.Context, task workqueue.Job, source string, renderErr error, repairCount int) {
	code := "template_compile_failed"
	message := renderErr.Error()
	var failure *document.ExtractionFailure
	if errors.As(renderErr, &failure) {
		code = failure.Code
		message = failure.Message
	}
	feedback := fmt.Sprintf("Document rendering validation failed [%s]: %s", code, message)
	_ = p.Repository.RecordStep(ctx, task, Step{
		ID:          id.New("step"),
		Kind:        "system_warning",
		Content:     limit(source, 100000),
		Feedback:    limit(feedback, 4000),
		RepairCount: repairCount,
		CreatedAt:   time.Now().UTC(),
	})
}

func (p *Processor) fail(ctx context.Context, task workqueue.Job, code, message string, retryable bool) {
	if ctx.Err() != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
	}
	message = limit(message, 2000)
	_ = p.Repository.Fail(ctx, task, code, message)
	var err error
	if retryable && task.Attempt < task.MaxAttempts {
		err = p.Queue.Retry(ctx, task, p.WorkerID, code, message, time.Duration(task.Attempt*task.Attempt)*5*time.Second)
	} else {
		err = p.Queue.Fail(ctx, task.ID, p.WorkerID, code, message)
	}
	if err != nil {
		p.Logger.Error("finalize generation failure", "error", err)
	}
}

func writerPrompt(run Run, p profile.Profile, j job.Job) string {
	return prompts.WriterTask(run.DocumentType, run.Language, p.TargetRole, run.PageTarget, run.CustomInstructions, p.Content, j.Title, j.Company, j.Location, j.Description)
}
func rendererPrompt(run Run, t resumetemplate.Template, draft, avatarName string) string {
	assetInstruction := prompts.TemplateApplierNoAvatarInstruction
	if avatarName != "" {
		assetInstruction = prompts.TemplateApplierAvatarInstruction(avatarName)
	}
	return prompts.RendererTask(run.DocumentType, run.PageTarget, t.Name, t.SourceName, t.EntryFile, assetInstruction, draft)
}
func revisionPrompt(run Run, draft, source, instruction string) string {
	return prompts.RendererRevisionTask(run.DocumentType, instruction, draft, source)
}
func reviewerPrompt(run Run, pageCount int) string {
	return prompts.ReviewerTask(run.DocumentType, run.PageTarget, pageCount)
}

// parseReview parses the Visual Reviewer's JSON response. visionUnsupported
// reports the model's own self-declared "I was not given a usable image"
// signal (see prompts.VisualReviewerSystem) — distinct from a provider-level
// rejection, which surfaces as ErrVisionUnsupported from the HTTP gateway
// instead. Both end up handled the same way by the caller.
//
// Models don't reliably follow the exact {"visionUnsupported":true} escape
// hatch the prompt asks for — some paraphrase the same complaint in their
// own words instead (e.g. "no rasterized page images ... were available for
// visual review"). reviewerReportsNoImage catches that free-text case too,
// the same way gateway.go's visionUnsupported() pattern-matches a provider's
// own error text rather than relying on a fixed error code.
func parseReview(value string) (approved bool, feedback string, visionUnsupported bool) {
	clean := cleanModelSource(value)
	var result struct {
		Approved          bool   `json:"approved"`
		Feedback          string `json:"feedback"`
		VisionUnsupported bool   `json:"visionUnsupported"`
	}
	if json.Unmarshal([]byte(clean), &result) == nil {
		feedback = strings.TrimSpace(result.Feedback)
		return result.Approved, feedback, result.VisionUnsupported || reviewerReportsNoImage(feedback)
	}
	return false, limit(clean, 4000), false
}

func reviewerReportsNoImage(feedback string) bool {
	value := strings.ToLower(feedback)
	if !strings.Contains(value, "image") && !strings.Contains(value, "pdf") {
		return false
	}
	patterns := []string{
		"no image", "no page image", "no rasterized", "no inspectable",
		"not available for visual review", "no usable image", "no visual content",
		"cannot inspect", "unable to inspect", "cannot see", "unable to see", "unable to view",
		"was not provided", "were not available", "not supplied", "no supported pdf",
		"did not receive", "no image data", "missing image",
	}
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}
func expectedPages(target string) int {
	if target == "one_page" {
		return 1
	}
	if target == "two_pages" {
		return 2
	}
	return 0
}
func limit(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n[truncated]"
}
