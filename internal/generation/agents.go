package generation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/prompts"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/tools"
)

var ErrAgentConnection = errors.New("agent model connection unavailable")

type GenerationAgentTeam struct {
	connections RuntimeResolver
	gateway     Gateway
	documents   DocumentRenderer
	blobs       GenerationBlobs
}

func NewGenerationAgentTeam(connections RuntimeResolver, gateway Gateway, documents DocumentRenderer, blobs GenerationBlobs) GenerationAgentTeam {
	return GenerationAgentTeam{connections: connections, gateway: gateway, documents: documents, blobs: blobs}
}

func (t GenerationAgentTeam) run(ctx context.Context, workspaceID string, choice ModelChoice, systemPrompt, input string, agentTools []tools.Tool, imageSource func() []string, maxTokens int, requiredTools []requiredAgentTool) (string, error) {
	runtime, err := t.connections.RuntimeConnection(ctx, workspaceID, choice.ConnectionID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAgentConnection, err)
	}
	model := &gatewayModel{gateway: t.gateway, runtime: runtime, model: choice.Model, systemPrompt: systemPrompt, imageSource: imageSource, maxTokens: maxTokens, agentOutput: true, requiredTools: requiredTools}
	agent := agents.NewOneShotAgent(
		model,
		agentTools,
		agents.WithPromptPrefix(prompts.WithScopedTools(systemPrompt)),
	)
	executor := agents.NewExecutor(agent, agents.WithMaxIterations(5))
	return chains.Run(ctx, executor, input)
}

func (t GenerationAgentTeam) Write(ctx context.Context, workspaceID string, run Run, profileValue profile.Profile, opportunity job.Job) (string, error) {
	contextValue := writerPrompt(run, profileValue, opportunity)
	contextTool := &staticContextTool{name: "read_generation_context", description: "Read the immutable Profile, Opportunity, and generation requirements for this run.", content: contextValue}
	return t.run(ctx, workspaceID, run.Writer, prompts.WriterSystem(run.DocumentType), contextValue, []tools.Tool{contextTool}, nil, writerMaxTokens, nil)
}

type TemplateApplyResult struct {
	Source          string
	FallbackReason  string
	ValidationError error
	Failures        []TemplateValidationFailure
}

type TemplateValidationFailure struct {
	Source string
	Error  error
}

func (t GenerationAgentTeam) DesignDocument(ctx context.Context, workspaceID string, run Run, avatarObjectID, draft, revisionPromptValue string, useFallback bool) (TemplateApplyResult, error) {
	if useFallback {
		return TemplateApplyResult{Source: fallbackHTML(draft), FallbackReason: "the previous safe-layout render could not complete and this retry resumed directly from the corrected fallback"}, nil
	}
	avatarName, avatar, err := loadProfileAvatar(ctx, t.blobs, workspaceID, avatarObjectID)
	if err != nil {
		return TemplateApplyResult{}, err
	}
	avatarInstruction := prompts.DesignerNoAvatarInstruction
	if avatarName != "" {
		avatarInstruction = prompts.DesignerAvatarInstruction(profileAvatarPlaceholder)
	}
	designBrief := prompts.DesignBrief(run.DocumentType, run.Language, run.PageTarget, run.CustomInstructions, avatarInstruction, draft)
	prompt := prompts.DesignTask
	if revisionPromptValue != "" {
		currentSource := strings.TrimSpace(run.RenderedSource)
		if currentSource == "" {
			currentSource = fallbackHTML(draft)
		}
		prompt = prompts.DesignRevisionTask(revisionPromptValue, currentSource)
	}
	briefTool := &staticContextTool{name: "read_document_design_brief", description: "Read the immutable document requirements, grounded draft, page target, and available profile assets before designing.", content: designBrief}
	renderTool := &renderHTMLPDFTool{documents: t.documents, blobs: t.blobs, workspaceID: workspaceID, avatarObjectID: avatarObjectID, avatarName: avatarName, avatar: avatar, avatarLoaded: true}
	requirements := []requiredAgentTool{{name: briefTool.Name(), satisfied: briefTool.succeeded}, {name: renderTool.Name(), satisfied: renderTool.succeeded}}
	source, err := t.run(ctx, workspaceID, run.Renderer, prompts.DesignerSystem, prompt, []tools.Tool{briefTool, renderTool}, nil, rendererMaxTokens, requirements)
	if err != nil {
		if renderTool.succeeded() {
			err = nil
		} else if renderTool.lastErr != nil && agentIterationsExhausted(err) {
			return TemplateApplyResult{Source: renderTool.source, FallbackReason: "the Document Designer exhausted its repair attempts without producing valid HTML", ValidationError: renderTool.lastErr, Failures: renderTool.failures}, nil
		} else {
			return TemplateApplyResult{}, err
		}
	}
	if renderTool.source != "" {
		source = renderTool.source
	}
	source = cleanModelSource(source)
	if renderTool.succeeded() && strings.Contains(strings.ToLower(source), "</html>") {
		return TemplateApplyResult{Source: source, Failures: renderTool.failures}, nil
	}
	validationErr := renderTool.lastErr
	if validationErr == nil {
		validationErr = errors.New("the Document Designer did not return a complete validated HTML document")
	}
	return TemplateApplyResult{Source: source, FallbackReason: validationErr.Error(), ValidationError: validationErr, Failures: renderTool.failures}, nil
}

func (t GenerationAgentTeam) ApplyTemplate(ctx context.Context, workspaceID string, run Run, templateValue resumetemplate.Template, avatarObjectID, draft, revisionPromptValue string, useFallback bool) (TemplateApplyResult, error) {
	if useFallback {
		return TemplateApplyResult{Source: fallbackLatex(draft), FallbackReason: "the previous safe-layout render could not complete and this retry resumed directly from the corrected fallback"}, nil
	}
	avatarName, avatar, err := loadProfileAvatar(ctx, t.blobs, workspaceID, avatarObjectID)
	if err != nil {
		return TemplateApplyResult{}, err
	}
	prompt := rendererPrompt(run, templateValue, draft, avatarName)
	if revisionPromptValue != "" {
		currentSource := run.RenderedSource
		if strings.TrimSpace(currentSource) == "" {
			currentSource = fallbackLatex(draft)
		}
		prompt = revisionPrompt(run, draft, currentSource, revisionPromptValue)
	}
	templateTool := &staticContextTool{name: "read_template_source", description: "Read the complete extracted template project before applying it. Identify the entry structure, custom macros, usage examples, local assets, and any supported avatar or photo command.", content: limit(templateValue.Content, 100000)}
	renderTool := &renderPDFTool{documents: t.documents, blobs: t.blobs, workspaceID: workspaceID, template: templateValue, avatarObjectID: avatarObjectID, avatarName: avatarName, avatar: avatar, avatarLoaded: true}
	requirements := []requiredAgentTool{{name: templateTool.Name(), satisfied: templateTool.succeeded}, {name: renderTool.Name(), satisfied: renderTool.succeeded}}
	source, err := t.run(ctx, workspaceID, run.Renderer, prompts.TemplateApplierSystem, prompt, []tools.Tool{templateTool, renderTool}, nil, rendererMaxTokens, requirements)
	if err != nil {
		if renderTool.succeeded() {
			err = nil
		} else if renderTool.lastErr != nil && agentIterationsExhausted(err) {
			return TemplateApplyResult{
				Source:          renderTool.source,
				FallbackReason:  "the Template Applying agent exhausted its repair attempts without producing compilable LaTeX",
				ValidationError: renderTool.lastErr,
				Failures:        renderTool.failures,
			}, nil
		} else {
			return TemplateApplyResult{}, err
		}
	}
	if renderTool.source != "" {
		source = renderTool.source
	}
	source = cleanModelSource(source)
	if renderTool.succeeded() && strings.Contains(source, "\\begin{document}") && strings.Contains(source, "\\end{document}") {
		return TemplateApplyResult{Source: source, Failures: renderTool.failures}, nil
	}
	reason := "the selected renderer model did not return a complete LaTeX document"
	if revisionPromptValue != "" {
		reason = "the renderer model returned an incomplete document for the requested revision"
	}
	validationErr := renderTool.lastErr
	if validationErr == nil {
		validationErr = errors.New(reason)
	}
	return TemplateApplyResult{Source: source, FallbackReason: reason, ValidationError: validationErr, Failures: renderTool.failures}, nil
}

func (t GenerationAgentTeam) Polish(ctx context.Context, workspaceID string, run Run, templateValue resumetemplate.Template, avatarObjectID, draft, source, feedback string) (string, error) {
	prompt := prompts.PolishTask(feedback, draft, source)
	avatarName, avatar, err := loadProfileAvatar(ctx, t.blobs, workspaceID, avatarObjectID)
	if err != nil {
		return "", err
	}
	renderTool := &renderPDFTool{documents: t.documents, blobs: t.blobs, workspaceID: workspaceID, template: templateValue, avatarObjectID: avatarObjectID, avatarName: avatarName, avatar: avatar, avatarLoaded: true}
	result, err := t.run(ctx, workspaceID, run.Renderer, prompts.LayoutPolishSystem, prompt, []tools.Tool{renderTool}, nil, rendererMaxTokens, []requiredAgentTool{{name: renderTool.Name(), satisfied: renderTool.succeeded}})
	if renderTool.source != "" {
		result = renderTool.source
	}
	if err != nil && renderTool.succeeded() {
		err = nil
	}
	if err == nil && !renderTool.succeeded() {
		err = renderTool.lastErr
		if err == nil {
			err = errors.New("the Layout Polishing agent did not validate its LaTeX with render_pdf")
		}
	}
	return cleanModelSource(result), err
}

func (t GenerationAgentTeam) PolishDesign(ctx context.Context, workspaceID string, run Run, avatarObjectID, draft, source, feedback string) (string, error) {
	prompt := prompts.PolishDesignTask(feedback, draft, source)
	avatarName, avatar, err := loadProfileAvatar(ctx, t.blobs, workspaceID, avatarObjectID)
	if err != nil {
		return "", err
	}
	renderTool := &renderHTMLPDFTool{documents: t.documents, blobs: t.blobs, workspaceID: workspaceID, avatarObjectID: avatarObjectID, avatarName: avatarName, avatar: avatar, avatarLoaded: true}
	result, err := t.run(ctx, workspaceID, run.Renderer, prompts.DesignerRepairSystem, prompt, []tools.Tool{renderTool}, nil, rendererMaxTokens, []requiredAgentTool{{name: renderTool.Name(), satisfied: renderTool.succeeded}})
	if renderTool.source != "" {
		result = renderTool.source
	}
	if err != nil && renderTool.succeeded() {
		err = nil
	}
	if err == nil && !renderTool.succeeded() {
		err = renderTool.lastErr
		if err == nil {
			err = errors.New("the Document Designer did not validate its HTML with render_html_pdf")
		}
	}
	return cleanModelSource(result), err
}

func (t GenerationAgentTeam) Review(ctx context.Context, workspaceID string, run Run, pdf []byte) (document.PDFPages, bool, string, string, error) {
	pageTool := &pdfToImagesTool{documents: t.documents, pdf: pdf}
	if _, err := pageTool.Call(ctx, "current PDF"); err != nil {
		return document.PDFPages{}, false, "", "", err
	}
	response, err := t.run(ctx, workspaceID, run.Reviewer, prompts.VisualReviewerSystem, reviewerPrompt(run, pageTool.pages.PageCount), []tools.Tool{pageTool}, func() []string { return pageTool.pages.Images }, reviewerMaxTokens, nil)
	if err != nil {
		return pageTool.pages, false, "", "", err
	}
	approved, feedback, visionUnsupported := parseReview(response)
	if visionUnsupported {
		// The provider returned a normal 200 response, but the model itself
		// reported it never received a usable image (e.g. a non-multimodal
		// model that silently ignores image content instead of the provider
		// rejecting the request outright). Route through the same fallback
		// as a provider-level rejection rather than treating a blind guess
		// as a real review verdict.
		return pageTool.pages, false, feedback, response, ErrVisionUnsupported
	}
	return pageTool.pages, approved, feedback, response, nil
}

func agentIterationsExhausted(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "iteration") || strings.Contains(message, "stopped") || strings.Contains(message, "finished")
}

func (t GenerationAgentTeam) RenderPDF(ctx context.Context, workspaceID string, templateValue resumetemplate.Template, avatarObjectID, source string) ([]byte, error) {
	tool := &renderPDFTool{documents: t.documents, blobs: t.blobs, workspaceID: workspaceID, template: templateValue, avatarObjectID: avatarObjectID}
	if _, err := tool.Call(ctx, source); err != nil {
		return nil, err
	}
	if tool.lastErr != nil {
		return nil, tool.lastErr
	}
	return tool.pdf, nil
}

func (t GenerationAgentTeam) RenderHTMLPDF(ctx context.Context, workspaceID, avatarObjectID, source string) ([]byte, error) {
	tool := &renderHTMLPDFTool{documents: t.documents, blobs: t.blobs, workspaceID: workspaceID, avatarObjectID: avatarObjectID}
	if _, err := tool.Call(ctx, source); err != nil {
		return nil, err
	}
	if tool.lastErr != nil {
		return nil, tool.lastErr
	}
	return tool.pdf, nil
}

func (t GenerationAgentTeam) StorePDF(ctx context.Context, workspaceID string, pdf []byte) (string, error) {
	tool := &storePDFArtifactTool{blobs: t.blobs, workspaceID: workspaceID, pdf: pdf}
	if _, err := tool.Call(ctx, "final PDF"); err != nil {
		return "", err
	}
	return tool.objectID, nil
}
