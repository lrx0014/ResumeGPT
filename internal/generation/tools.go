package generation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
	"github.com/tmc/langchaingo/tools"
)

const profileAvatarPlaceholder = "{{PROFILE_AVATAR}}"

type GenerationBlobs interface {
	blobstore.Reader
	blobstore.Writer
}

type staticContextTool struct {
	name        string
	description string
	content     string
	called      bool
}

var _ tools.Tool = (*staticContextTool)(nil)

func (t *staticContextTool) Name() string        { return t.name }
func (t *staticContextTool) Description() string { return t.description }
func (t *staticContextTool) Call(context.Context, string) (string, error) {
	t.called = true
	return t.content, nil
}

func (t *staticContextTool) succeeded() bool { return t.called }

type renderPDFTool struct {
	documents      DocumentRenderer
	blobs          GenerationBlobs
	workspaceID    string
	template       resumetemplate.Template
	avatarObjectID string
	avatarName     string
	avatar         []byte
	avatarLoaded   bool
	source         string
	pdf            []byte
	lastErr        error
	failures       []TemplateValidationFailure
}

type renderHTMLPDFTool struct {
	documents      DocumentRenderer
	blobs          GenerationBlobs
	workspaceID    string
	avatarObjectID string
	avatarName     string
	avatar         []byte
	avatarLoaded   bool
	source         string
	pdf            []byte
	lastErr        error
	failures       []TemplateValidationFailure
}

var _ tools.Tool = (*renderHTMLPDFTool)(nil)

func (*renderHTMLPDFTool) Name() string { return "render_html_pdf" }
func (*renderHTMLPDFTool) Description() string {
	return "Render a complete self-contained HTML and CSS document into PDF in an isolated renderer. External URLs, scripts, and local files are unavailable. The input must be the raw HTML document itself, starting with <!doctype html> or <html> — never wrap it in JSON or any other object."
}
func (t *renderHTMLPDFTool) Call(ctx context.Context, source string) (string, error) {
	t.source = cleanModelSource(source)
	if !looksLikeCompleteHTMLDocument(t.source) {
		t.pdf = nil
		t.lastErr = errors.New("candidate source is not a complete HTML document")
		t.failures = append(t.failures, TemplateValidationFailure{Source: t.source, Error: t.lastErr})
		result, _ := json.Marshal(map[string]any{"status": "render_error", "code": "incomplete_html", "error": t.lastErr.Error()})
		return string(result), nil
	}
	avatarName, avatar, err := t.loadAvatar(ctx)
	if err != nil {
		t.lastErr = err
		return "", err
	}
	renderedSource := t.source
	if strings.Contains(renderedSource, profileAvatarPlaceholder) {
		if len(avatar) == 0 {
			renderedSource = strings.ReplaceAll(renderedSource, profileAvatarPlaceholder, "")
		} else {
			mediaType := "image/png"
			if strings.HasSuffix(avatarName, ".jpg") {
				mediaType = "image/jpeg"
			}
			dataURL := "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(avatar)
			renderedSource = strings.ReplaceAll(renderedSource, profileAvatarPlaceholder, dataURL)
		}
	}
	pdf, err := t.documents.HTMLPDF(ctx, renderedSource)
	if err != nil {
		t.pdf = nil
		t.lastErr = err
		t.failures = append(t.failures, TemplateValidationFailure{Source: t.source, Error: err})
		result, _ := json.Marshal(map[string]any{"status": "render_error", "error": limit(err.Error(), 1200)})
		return string(result), nil
	}
	t.pdf = pdf
	t.lastErr = nil
	result, _ := json.Marshal(map[string]any{"status": "rendered", "bytes": len(pdf)})
	return string(result), nil
}

func (t *renderHTMLPDFTool) succeeded() bool { return len(t.pdf) > 0 && t.lastErr == nil }

// looksLikeCompleteHTMLDocument requires the source to actually start with an
// HTML document tag, not merely contain "<html"/"<body"/"</html>" somewhere
// in its text — a model that wraps its HTML in a JSON object such as
// {"html": "<!doctype html>...</html>"} would otherwise pass a naive
// substring check while still leaking the JSON scaffolding into the PDF.
func looksLikeCompleteHTMLDocument(source string) bool {
	lowered := strings.ToLower(strings.TrimSpace(source))
	if !strings.HasPrefix(lowered, "<!doctype") && !strings.HasPrefix(lowered, "<html") {
		return false
	}
	return strings.Contains(lowered, "<body") && strings.Contains(lowered, "</html>")
}

func (t *renderHTMLPDFTool) loadAvatar(ctx context.Context) (string, []byte, error) {
	if t.avatarLoaded {
		return t.avatarName, t.avatar, nil
	}
	t.avatarLoaded = true
	t.avatarName, t.avatar, t.lastErr = loadProfileAvatar(ctx, t.blobs, t.workspaceID, t.avatarObjectID)
	return t.avatarName, t.avatar, t.lastErr
}

var _ tools.Tool = (*renderPDFTool)(nil)

func (*renderPDFTool) Name() string { return "render_pdf" }
func (*renderPDFTool) Description() string {
	return "Compile a complete LaTeX entry source into a sandboxed PDF with network access and shell escape disabled."
}
func (t *renderPDFTool) Call(ctx context.Context, source string) (string, error) {
	t.source = cleanModelSource(source)
	if !strings.Contains(t.source, "\\begin{document}") || !strings.Contains(t.source, "\\end{document}") {
		t.pdf = nil
		t.lastErr = errors.New("candidate source is not a complete LaTeX document")
		t.failures = append(t.failures, TemplateValidationFailure{Source: t.source, Error: t.lastErr})
		result, _ := json.Marshal(map[string]any{"status": "compile_error", "code": "incomplete_latex", "error": t.lastErr.Error()})
		return string(result), nil
	}
	pdf, err := t.render(ctx, t.source)
	if err != nil {
		t.pdf = nil
		t.lastErr = err
		t.failures = append(t.failures, TemplateValidationFailure{Source: t.source, Error: err})
		result, _ := json.Marshal(map[string]any{"status": "compile_error", "error": limit(err.Error(), 1200)})
		return string(result), nil
	}
	t.pdf = pdf
	t.lastErr = nil
	result, _ := json.Marshal(map[string]any{"status": "rendered", "bytes": len(pdf)})
	return string(result), nil
}

func (t *renderPDFTool) succeeded() bool {
	return len(t.pdf) > 0 && t.lastErr == nil
}
func (t *renderPDFTool) render(ctx context.Context, source string) ([]byte, error) {
	avatarName, avatar, err := t.loadAvatar(ctx)
	if err != nil {
		return nil, err
	}
	if t.template.BuiltIn || strings.HasSuffix(strings.ToLower(t.template.SourceName), ".tex") {
		if len(avatar) > 0 {
			archive, err := latexArchive(source, avatarName, avatar)
			if err != nil {
				return nil, err
			}
			return t.documents.PreviewTemplate(ctx, "generated.zip", "main.tex", bytes.NewReader(archive), int64(len(archive)))
		}
		return t.documents.PreviewTemplate(ctx, "generated.tex", "", strings.NewReader(source), int64(len(source)))
	}
	if !strings.HasSuffix(strings.ToLower(t.template.SourceName), ".zip") {
		return nil, errors.New("only LaTeX templates are supported")
	}
	object, err := t.blobs.Open(ctx, t.workspaceID, t.template.ObjectID)
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()
	data, err := io.ReadAll(io.LimitReader(object.Body, 10*1024*1024+1))
	if err != nil {
		return nil, err
	}
	archive, entry, err := replaceArchiveEntry(data, t.template.EntryFile, source)
	if err != nil {
		return nil, err
	}
	if len(avatar) > 0 {
		archive, err = addArchiveFile(archive, archiveAssetPath(entry, avatarName), avatar)
		if err != nil {
			return nil, err
		}
	}
	return t.documents.PreviewTemplate(ctx, "generated.zip", entry, bytes.NewReader(archive), int64(len(archive)))
}

func (t *renderPDFTool) loadAvatar(ctx context.Context) (string, []byte, error) {
	if t.avatarLoaded {
		return t.avatarName, t.avatar, nil
	}
	t.avatarLoaded = true
	t.avatarName, t.avatar, t.lastErr = loadProfileAvatar(ctx, t.blobs, t.workspaceID, t.avatarObjectID)
	return t.avatarName, t.avatar, t.lastErr
}

func loadProfileAvatar(ctx context.Context, blobs GenerationBlobs, workspaceID, avatarObjectID string) (string, []byte, error) {
	if strings.TrimSpace(avatarObjectID) == "" {
		return "", nil, nil
	}
	object, err := blobs.Open(ctx, workspaceID, avatarObjectID)
	if err != nil {
		return "", nil, errors.New("the profile avatar is unavailable")
	}
	defer object.Body.Close()
	if object.Size < 1 || object.Size > 5*1024*1024 {
		return "", nil, errors.New("the profile avatar exceeds the 5 MiB generation limit")
	}
	data, err := io.ReadAll(io.LimitReader(object.Body, 5*1024*1024+1))
	if err != nil {
		return "", nil, errors.New("the profile avatar could not be read")
	}
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "resumegpt-avatar.png", data, nil
	case len(data) >= 3 && bytes.Equal(data[:3], []byte("\xff\xd8\xff")):
		return "resumegpt-avatar.jpg", data, nil
	default:
		return "", nil, errors.New("the profile avatar is not a valid PNG or JPEG image")
	}
}

type pdfToImagesTool struct {
	documents DocumentRenderer
	pdf       []byte
	pages     document.PDFPages
}

var _ tools.Tool = (*pdfToImagesTool)(nil)

func (*pdfToImagesTool) Name() string { return "pdf_to_images" }
func (*pdfToImagesTool) Description() string {
	return "Rasterize the current bounded PDF into page images for visual inspection."
}
func (t *pdfToImagesTool) Call(ctx context.Context, _ string) (string, error) {
	pages, err := t.documents.PDFPages(ctx, t.pdf)
	if err != nil {
		return "", err
	}
	t.pages = pages
	result, _ := json.Marshal(map[string]any{"status": "rasterized", "page_count": pages.PageCount})
	return string(result), nil
}

type storePDFArtifactTool struct {
	blobs       GenerationBlobs
	workspaceID string
	pdf         []byte
	objectID    string
}

var _ tools.Tool = (*storePDFArtifactTool)(nil)

func (*storePDFArtifactTool) Name() string { return "store_pdf_artifact" }
func (*storePDFArtifactTool) Description() string {
	return "Store the final PDF as a workspace-scoped immutable artifact."
}
func (t *storePDFArtifactTool) Call(ctx context.Context, _ string) (string, error) {
	objectID := blobstore.NewObjectID()
	if err := t.blobs.Put(ctx, t.workspaceID, objectID, "application/pdf", bytes.NewReader(t.pdf), int64(len(t.pdf))); err != nil {
		return "", err
	}
	t.objectID = objectID
	result, _ := json.Marshal(map[string]string{"status": "stored", "object_id": objectID})
	return string(result), nil
}
