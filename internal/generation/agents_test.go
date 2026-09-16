package generation

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

func TestWriterAgentCanReadGenerationContextTool(t *testing.T) {
	gateway := &processorGateway{responses: []string{
		"Action: read_generation_context\nAction Input: current run",
		"Final Answer: # Tailored resume",
	}}
	team := NewGenerationAgentTeam(processorSettings{}, gateway, processorDocuments{}, &processorBlobs{})
	run := Run{Writer: ModelChoice{ConnectionID: "writer", Model: "writer-model"}, DocumentType: "resume", PageTarget: "one_page"}

	draft, err := team.Write(context.Background(), "ws_test", run, profile.Profile{Content: "Built Go services."}, job.Job{Title: "Backend Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(draft) != "# Tailored resume" || gateway.calls != 2 {
		t.Fatalf("draft=%q calls=%d", draft, gateway.calls)
	}
}

func TestTemplateAgentCanValidateLatexWithRenderTool(t *testing.T) {
	source := "\\documentclass{article}\\begin{document}Validated\\end{document}"
	gateway := &processorGateway{responses: []string{
		"Action: render_pdf\nAction Input: " + source,
	}}
	team := NewGenerationAgentTeam(processorSettings{}, gateway, processorDocuments{}, &processorBlobs{})
	run := Run{Renderer: ModelChoice{ConnectionID: "renderer", Model: "renderer-model"}, DocumentType: "resume", PageTarget: "one_page"}

	result, err := team.ApplyTemplate(context.Background(), "ws_test", run, resumetemplate.Template{BuiltIn: true, Format: "latex", Content: "\\newcommand{\\name}[1]{#1}"}, "", "# Draft", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != source || gateway.calls != 1 || !strings.Contains(gateway.prompts[0], `\newcommand{\name}`) {
		t.Fatalf("source=%q calls=%d", result.Source, gateway.calls)
	}
}

func TestRenderPDFToolAddsProfileAvatarBesideNestedZipEntry(t *testing.T) {
	var templateArchive bytes.Buffer
	writer := zip.NewWriter(&templateArchive)
	entry, _ := writer.Create("project/main.tex")
	_, _ = entry.Write([]byte("original"))
	style, _ := writer.Create("project/style.sty")
	_, _ = style.Write([]byte("style"))
	_ = writer.Close()

	avatar := append([]byte("\x89PNG\r\n\x1a\n"), []byte("avatar")...)
	blobs := &agentAssetBlobs{objects: map[string][]byte{"obj_template": templateArchive.Bytes(), "obj_avatar": avatar}}
	documents := &archiveCaptureDocuments{}
	tool := &renderPDFTool{
		documents: documents, blobs: blobs, workspaceID: "ws_test", avatarObjectID: "obj_avatar",
		template: resumetemplate.Template{SourceName: "resume.zip", ObjectID: "obj_template", EntryFile: "project/main.tex"},
	}
	source := "\\documentclass{article}\\begin{document}\\includegraphics{resumegpt-avatar.png}\\end{document}"

	if _, err := tool.Call(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	if tool.lastErr != nil || documents.entry != "project/main.tex" || string(documents.files["project/main.tex"]) != source || !bytes.Equal(documents.files["project/resumegpt-avatar.png"], avatar) {
		t.Fatalf("render error=%v entry=%q files=%v", tool.lastErr, documents.entry, documents.files)
	}
}

func TestTemplateAgentRepairsCompilationFailureBeforeFinishing(t *testing.T) {
	invalid := "\\documentclass{article}\\begin{document}\\invalidcommand\\end{document}"
	repaired := "\\documentclass{article}\\begin{document}Repaired\\end{document}"
	gateway := &processorGateway{responses: []string{invalid, repaired}}
	documents := &agentRepairDocuments{}
	team := NewGenerationAgentTeam(processorSettings{}, gateway, documents, &processorBlobs{})
	run := Run{Renderer: ModelChoice{ConnectionID: "renderer", Model: "renderer-model"}, DocumentType: "resume", PageTarget: "one_page"}

	result, err := team.ApplyTemplate(context.Background(), "ws_test", run, resumetemplate.Template{BuiltIn: true, Format: "latex", Content: "\\newcommand{\\name}[1]{#1}"}, "", "# Draft", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != repaired || result.ValidationError != nil || len(result.Failures) != 1 || result.Failures[0].Source != invalid || documents.calls != 2 || gateway.calls != 2 {
		t.Fatalf("result=%+v document_calls=%d gateway_calls=%d", result, documents.calls, gateway.calls)
	}
}

func TestTemplateAgentFallsBackOnlyAfterRepairIterationsAreExhausted(t *testing.T) {
	source := "\\documentclass{article}\\begin{document}\\invalidcommand\\end{document}"
	gateway := &processorGateway{responses: []string{source, source, source, source}}
	documents := &agentRepairDocuments{alwaysFail: true}
	team := NewGenerationAgentTeam(processorSettings{}, gateway, documents, &processorBlobs{})
	run := Run{Renderer: ModelChoice{ConnectionID: "renderer", Model: "renderer-model"}, DocumentType: "resume", PageTarget: "one_page"}

	result, err := team.ApplyTemplate(context.Background(), "ws_test", run, resumetemplate.Template{BuiltIn: true, Format: "latex", Content: "\\newcommand{\\name}[1]{#1}"}, "", "# Draft", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidationError == nil || result.Source != source || documents.calls != 4 || gateway.calls != 4 {
		t.Fatalf("result=%+v document_calls=%d gateway_calls=%d", result, documents.calls, gateway.calls)
	}
	if !strings.Contains(result.FallbackReason, "exhausted") {
		t.Fatalf("fallback reason=%q", result.FallbackReason)
	}
}

type agentRepairDocuments struct {
	calls      int
	alwaysFail bool
}

type agentAssetBlobs struct {
	objects map[string][]byte
}

func (b *agentAssetBlobs) Open(_ context.Context, _, objectID string) (blobstore.Object, error) {
	data, ok := b.objects[objectID]
	if !ok {
		return blobstore.Object{}, errors.New("object not found")
	}
	return blobstore.Object{Body: io.NopCloser(bytes.NewReader(data)), Size: int64(len(data))}, nil
}

func (*agentAssetBlobs) Put(context.Context, string, string, string, io.Reader, int64) error {
	return nil
}

type archiveCaptureDocuments struct {
	entry string
	files map[string][]byte
}

func (d *archiveCaptureDocuments) PreviewTemplate(_ context.Context, name, entry string, body io.Reader, size int64) ([]byte, error) {
	if name != "generated.zip" {
		return nil, errors.New("expected generated ZIP")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), size)
	if err != nil {
		return nil, err
	}
	d.entry = entry
	d.files = map[string][]byte{}
	for _, file := range reader.File {
		input, err := file.Open()
		if err != nil {
			return nil, err
		}
		d.files[file.Name], err = io.ReadAll(input)
		_ = input.Close()
		if err != nil {
			return nil, err
		}
	}
	return []byte("%PDF-result"), nil
}

func (*archiveCaptureDocuments) PDFPages(context.Context, []byte) (document.PDFPages, error) {
	return document.PDFPages{}, errors.New("not used")
}

func (d *agentRepairDocuments) PreviewTemplate(context.Context, string, string, io.Reader, int64) ([]byte, error) {
	d.calls++
	if d.alwaysFail || d.calls == 1 {
		return nil, &document.ExtractionFailure{Code: "preview_render_failed", Message: "LaTeX compilation failed: Undefined control sequence."}
	}
	return []byte("%PDF-result"), nil
}

func (*agentRepairDocuments) PDFPages(context.Context, []byte) (document.PDFPages, error) {
	return document.PDFPages{}, errors.New("not used")
}

func TestReviewerAgentCanInvokePDFToImagesTool(t *testing.T) {
	gateway := &processorGateway{responses: []string{
		"Action: pdf_to_images\nAction Input: current PDF",
		`Final Answer: {"approved":true,"feedback":"Layout is balanced."}`,
	}}
	team := NewGenerationAgentTeam(processorSettings{}, gateway, processorDocuments{}, &processorBlobs{})
	run := Run{Reviewer: ModelChoice{ConnectionID: "reviewer", Model: "vision-model"}, DocumentType: "resume", PageTarget: "one_page"}

	pages, approved, feedback, _, err := team.Review(context.Background(), "ws_test", run, []byte("%PDF-result"))
	if err != nil {
		t.Fatal(err)
	}
	if pages.PageCount != 1 || !approved || feedback != "Layout is balanced." || gateway.calls != 2 {
		t.Fatalf("pages=%d approved=%v feedback=%q calls=%d", pages.PageCount, approved, feedback, gateway.calls)
	}
}
