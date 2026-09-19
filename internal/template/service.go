package template

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

const maxTemplateBytes = 5 * 1024 * 1024

var templateFormats = map[string]string{".tex": "latex", ".zip": "latex", ".doc": "doc", ".docx": "docx"}

type Service struct {
	repository             Repository
	blobs                  blobstore.Signer
	uploads                bool
	reader                 blobstore.Reader
	defaultPreviewObjectID string
}

type Previewer interface {
	Preview(context.Context, string, io.Reader, int64) ([]byte, error)
	PreviewTemplate(context.Context, string, string, io.Reader, int64) ([]byte, error)
}

func NewService(repository Repository, blobs blobstore.Signer, uploads bool) *Service {
	return &Service{repository: repository, blobs: blobs, uploads: uploads}
}

func (s *Service) UploadsEnabled() bool { return s.uploads }

func (s *Service) ConfigurePreview(reader blobstore.Reader, defaultPreviewObjectID string) {
	s.reader, s.defaultPreviewObjectID = reader, defaultPreviewObjectID
}

func (s *Service) PreviewsEnabled() bool { return s.reader != nil && s.defaultPreviewObjectID != "" }

func (s *Service) List(ctx context.Context, workspaceID string) ([]Template, error) {
	items, err := s.repository.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return append([]Template{defaultResume(workspaceID, false)}, items...), nil
}

// Count reports how many templates are ready to use and how many are custom
// (non-built-in). Ready includes the always-present built-in Rezume template,
// which List prepends synthetically rather than storing in the repository.
func (s *Service) Count(ctx context.Context, workspaceID string) (Counts, error) {
	counts, err := s.repository.Count(ctx, workspaceID)
	if err != nil {
		return Counts{}, err
	}
	counts.Ready++
	return counts, nil
}

// builtInMatchesFilter reports whether the synthetic built-in Rezume template
// (never stored in the repository) would match filter, using the same
// search/kind semantics as the DB query.
func builtInMatchesFilter(filter Filter) bool {
	if filter.Kind != "" && filter.Kind != "resume" {
		return false
	}
	if filter.Search == "" {
		return true
	}
	query := strings.ToLower(filter.Search)
	builtin := defaultResume("", false)
	return strings.Contains(strings.ToLower(builtin.Name), query) ||
		strings.Contains(strings.ToLower(builtin.Description), query) ||
		strings.Contains(strings.ToLower(builtin.SourceName), query) ||
		strings.Contains(strings.ToLower(builtin.AuthorName), query) ||
		strings.Contains(strings.ToLower(builtin.Format), query)
}

// Search paginates and filters templates, splicing the synthetic built-in
// Rezume template (always logically first) into the DB-backed page.
func (s *Service) Search(ctx context.Context, workspaceID string, filter Filter) (Page, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	if len(filter.Search) > 200 {
		filter.Search = ""
	}
	if filter.Kind != "resume" && filter.Kind != "cover_letter" {
		filter.Kind = ""
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	includeBuiltIn := builtInMatchesFilter(filter)
	globalOffset := (filter.Page - 1) * filter.PageSize
	dbOffset, dbLimit := globalOffset, filter.PageSize
	prependBuiltIn := false
	if includeBuiltIn {
		if globalOffset == 0 {
			prependBuiltIn = true
			dbLimit--
		} else {
			dbOffset--
		}
	}

	items, dbTotal, err := s.repository.Search(ctx, workspaceID, filter.Search, filter.Kind, dbLimit, dbOffset)
	if err != nil {
		return Page{}, err
	}

	result := Page{Items: make([]Template, 0, len(items)+1), Page: filter.Page, PageSize: filter.PageSize, Total: dbTotal}
	if includeBuiltIn {
		result.Total++
	}
	if prependBuiltIn {
		result.Items = append(result.Items, defaultResume(workspaceID, false))
	}
	result.Items = append(result.Items, items...)
	return result, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, templateID string) (Template, error) {
	if templateID == DefaultResumeID {
		return defaultResume(workspaceID, true), nil
	}
	return s.repository.Get(ctx, workspaceID, templateID)
}

func (s *Service) Stage(ctx context.Context, workspaceID string, input StageInput) (StagedTemplate, error) {
	if !s.uploads {
		return StagedTemplate{}, ErrState
	}
	input.Name, input.Kind, input.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.Kind), strings.TrimSpace(input.Description)
	format, entryFile, validSource := validateTemplateSource(input.SourceName, input.ContentType, input.EntryFile)
	input.SourceName, input.ContentType, input.EntryFile = strings.TrimSpace(input.SourceName), strings.TrimSpace(input.ContentType), entryFile
	if !validSource || invalidText(input.Name, 120, false) || invalidText(input.Description, 1000, true) ||
		(input.Kind != "resume" && input.Kind != "cover_letter") || input.ContentType == "" || len(input.ContentType) > 200 {
		return StagedTemplate{}, ErrInvalid
	}
	now := time.Now().UTC()
	item := Template{ID: id.New("tpl"), WorkspaceID: workspaceID, Name: input.Name, Kind: input.Kind,
		Format: format, Description: input.Description, SourceName: input.SourceName, EntryFile: input.EntryFile, DeclaredMediaType: input.ContentType,
		ObjectID: blobstore.NewObjectID(), State: "staged", CreatedAt: now, UpdatedAt: now}
	target, err := s.blobs.PresignUpload(ctx, workspaceID, item.ObjectID, input.ContentType, 15*time.Minute)
	if err != nil {
		return StagedTemplate{}, err
	}
	if err := s.repository.Stage(ctx, item); err != nil {
		return StagedTemplate{}, err
	}
	return StagedTemplate{Template: item, Target: target}, nil
}

func (s *Service) ReplaceSource(ctx context.Context, workspaceID, templateID string, input SourceInput) (StagedTemplate, error) {
	if !s.uploads {
		return StagedTemplate{}, ErrState
	}
	if templateID == DefaultResumeID {
		return StagedTemplate{}, ErrBuiltIn
	}
	item, err := s.repository.Get(ctx, workspaceID, templateID)
	if err != nil {
		return StagedTemplate{}, err
	}
	format, entryFile, validSource := validateTemplateSource(input.SourceName, input.ContentType, input.EntryFile)
	input.SourceName, input.ContentType = strings.TrimSpace(input.SourceName), strings.TrimSpace(input.ContentType)
	if !validSource {
		return StagedTemplate{}, ErrInvalid
	}
	item.Format, item.SourceName, item.EntryFile, item.DeclaredMediaType = format, input.SourceName, entryFile, input.ContentType
	item.ObjectID, item.PreviewObjectID, item.Content = blobstore.NewObjectID(), "", ""
	item.State, item.JobID, item.ErrorCode, item.ErrorMessage = "staged", "", "", ""
	item.UpdatedAt = time.Now().UTC()
	target, err := s.blobs.PresignUpload(ctx, workspaceID, item.ObjectID, input.ContentType, 15*time.Minute)
	if err != nil {
		return StagedTemplate{}, err
	}
	if err := s.repository.Restage(ctx, item); err != nil {
		return StagedTemplate{}, err
	}
	return StagedTemplate{Template: item, Target: target}, nil
}

func (s *Service) Queue(ctx context.Context, workspaceID, templateID string) (Template, error) {
	if templateID == DefaultResumeID {
		return Template{}, ErrBuiltIn
	}
	job := workqueue.Job{ID: id.New("task"), WorkspaceID: workspaceID, Kind: ExtractJobKind,
		IdempotencyKey: templateID, MaxAttempts: 5, AvailableAt: time.Now().UTC()}
	return s.repository.Queue(ctx, workspaceID, templateID, job)
}

func (s *Service) Update(ctx context.Context, workspaceID, templateID string, input UpdateInput) (Template, error) {
	if templateID == DefaultResumeID {
		return Template{}, ErrBuiltIn
	}
	item, err := s.repository.Get(ctx, workspaceID, templateID)
	if err != nil {
		return Template{}, err
	}
	input.Name, input.Kind, input.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.Kind), strings.TrimSpace(input.Description)
	if invalidText(input.Name, 120, false) || invalidText(input.Description, 1000, true) || (input.Kind != "resume" && input.Kind != "cover_letter") {
		return Template{}, ErrInvalid
	}
	item.Name, item.Kind, item.Description, item.UpdatedAt = input.Name, input.Kind, input.Description, time.Now().UTC()
	return s.repository.Update(ctx, item)
}

func (s *Service) Delete(ctx context.Context, workspaceID, templateID string) error {
	if templateID == DefaultResumeID {
		return ErrBuiltIn
	}
	return s.repository.Delete(ctx, workspaceID, templateID)
}

func (s *Service) Download(ctx context.Context, workspaceID, templateID string) ([]byte, blobstore.SignedURL, error) {
	if templateID == DefaultResumeID {
		return append([]byte(nil), defaultResumeSource...), blobstore.SignedURL{}, nil
	}
	item, err := s.repository.Get(ctx, workspaceID, templateID)
	if err != nil {
		return nil, blobstore.SignedURL{}, err
	}
	if item.State != "ready" {
		return nil, blobstore.SignedURL{}, ErrState
	}
	signed, err := s.blobs.PresignDownload(ctx, workspaceID, item.ObjectID, 5*time.Minute)
	return nil, signed, err
}

func (s *Service) Preview(ctx context.Context, workspaceID, templateID string) ([]byte, error) {
	if !s.PreviewsEnabled() {
		return nil, ErrState
	}
	item, err := s.Get(ctx, workspaceID, templateID)
	if err != nil {
		return nil, err
	}
	if item.State != "ready" {
		return nil, ErrState
	}
	previewWorkspaceID, previewObjectID := workspaceID, item.PreviewObjectID
	if item.BuiltIn {
		previewWorkspaceID, previewObjectID = builtInWorkspaceID, s.defaultPreviewObjectID
	}
	if previewObjectID == "" {
		return nil, ErrState
	}
	object, err := s.reader.Open(ctx, previewWorkspaceID, previewObjectID)
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()
	if object.Size < 5 || object.Size > 20*1024*1024 {
		return nil, fmt.Errorf("stored template preview has an invalid size")
	}
	preview, err := io.ReadAll(io.LimitReader(object.Body, 20*1024*1024+1))
	if err != nil || len(preview) < 5 || len(preview) > 20*1024*1024 || string(preview[:5]) != "%PDF-" {
		return nil, fmt.Errorf("stored template preview is not a valid PDF")
	}
	return preview, nil
}

const builtInWorkspaceID = "ws_builtin"

func PrepareDefaultPreview(ctx context.Context, blobs TemplateBlobs, previewer Previewer) (string, error) {
	digest := sha256.Sum256(defaultResumeSource)
	objectID := "obj_builtin_rezume_preview_" + hex.EncodeToString(digest[:8])
	if existing, err := blobs.Open(ctx, builtInWorkspaceID, objectID); err == nil {
		header := make([]byte, 5)
		_, readErr := io.ReadFull(existing.Body, header)
		_ = existing.Body.Close()
		if readErr == nil && string(header) == "%PDF-" && existing.Size <= 20*1024*1024 {
			return objectID, nil
		}
	}
	preview, err := previewer.Preview(ctx, "rezume.tex", strings.NewReader(string(defaultResumeSource)), int64(len(defaultResumeSource)))
	if err != nil {
		return "", fmt.Errorf("render built-in template preview: %w", err)
	}
	if err := blobs.Put(ctx, builtInWorkspaceID, objectID, "application/pdf", bytes.NewReader(preview), int64(len(preview))); err != nil {
		return "", fmt.Errorf("store built-in template preview: %w", err)
	}
	return objectID, nil
}

func defaultResume(workspaceID string, includeContent bool) Template {
	content := ""
	if includeContent {
		content = string(defaultResumeSource)
	}
	return Template{ID: DefaultResumeID, WorkspaceID: workspaceID, Name: "Rezume", Kind: "resume", Format: "latex",
		Description: "A clean single-page LaTeX résumé template for developers.", SourceName: "rezume.tex",
		DeclaredMediaType: "application/x-tex", Content: content, State: "ready", BuiltIn: true,
		AuthorName: "Nanu Panchamurthy", SourceURL: "https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs", License: "MIT"}
}

func invalidText(value string, limit int, optional bool) bool {
	if optional && value == "" {
		return false
	}
	if value == "" || len(value) > limit || !utf8.ValidString(value) {
		return true
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func validateTemplateSource(sourceName, contentType, entryFile string) (string, string, bool) {
	sourceName, contentType, entryFile = strings.TrimSpace(sourceName), strings.TrimSpace(contentType), strings.TrimSpace(entryFile)
	ext := strings.ToLower(filepath.Ext(sourceName))
	format, supported := templateFormats[ext]
	if !supported || invalidText(sourceName, 200, false) || strings.ContainsAny(sourceName, "/\\") || contentType == "" || len(contentType) > 200 {
		return "", "", false
	}
	if ext != ".zip" {
		return format, "", entryFile == ""
	}
	if entryFile == "" {
		return format, "", true
	}
	if invalidText(entryFile, 300, false) || strings.Contains(entryFile, "\\") || strings.HasPrefix(entryFile, "/") || path.Clean(entryFile) != entryFile || strings.ToLower(path.Ext(entryFile)) != ".tex" {
		return "", "", false
	}
	for _, part := range strings.Split(entryFile, "/") {
		if part == "" || part == "." || part == ".." {
			return "", "", false
		}
	}
	return format, entryFile, true
}
