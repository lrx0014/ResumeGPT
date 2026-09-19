package template

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

const (
	ExtractJobKind  = "template.extract.v1"
	DefaultResumeID = "tpl_builtin_rezume"
)

var (
	ErrInvalid  = errors.New("invalid template")
	ErrNotFound = errors.New("template not found")
	ErrState    = errors.New("template is not ready for this operation")
	ErrBuiltIn  = errors.New("built-in template cannot be changed")
)

//go:embed assets/default_cv.tex
var defaultResumeSource []byte

type Template struct {
	ID                string    `json:"id"`
	WorkspaceID       string    `json:"workspaceId"`
	Name              string    `json:"name"`
	Kind              string    `json:"kind"`
	Format            string    `json:"format"`
	Description       string    `json:"description,omitempty"`
	SourceName        string    `json:"sourceName"`
	EntryFile         string    `json:"entryFile,omitempty"`
	DeclaredMediaType string    `json:"declaredMediaType"`
	ObjectID          string    `json:"objectId,omitempty"`
	PreviewObjectID   string    `json:"previewObjectId,omitempty"`
	Content           string    `json:"content,omitempty"`
	State             string    `json:"state"`
	JobID             string    `json:"jobId,omitempty"`
	ErrorCode         string    `json:"errorCode,omitempty"`
	ErrorMessage      string    `json:"errorMessage,omitempty"`
	BuiltIn           bool      `json:"builtIn"`
	AuthorName        string    `json:"authorName,omitempty"`
	SourceURL         string    `json:"sourceUrl,omitempty"`
	License           string    `json:"license,omitempty"`
	CreatedAt         time.Time `json:"createdAt,omitzero"`
	UpdatedAt         time.Time `json:"updatedAt,omitzero"`
}

type StageInput struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	SourceName  string `json:"sourceName"`
	ContentType string `json:"contentType"`
	EntryFile   string `json:"entryFile"`
}

type UpdateInput struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type SourceInput struct {
	SourceName  string `json:"sourceName"`
	ContentType string `json:"contentType"`
	EntryFile   string `json:"entryFile"`
}

type StagedTemplate struct {
	Template Template            `json:"template"`
	Target   blobstore.SignedURL `json:"target"`
}

type ExtractPayload struct {
	TemplateID string `json:"templateId"`
	ObjectID   string `json:"objectId"`
	SourceName string `json:"sourceName"`
	EntryFile  string `json:"entryFile,omitempty"`
}

type Counts struct {
	Ready  int `json:"ready"`
	Custom int `json:"custom"`
}

type Repository interface {
	List(context.Context, string) ([]Template, error)
	Get(context.Context, string, string) (Template, error)
	Stage(context.Context, Template) error
	Restage(context.Context, Template) error
	Queue(context.Context, string, string, workqueue.Job) (Template, error)
	Update(context.Context, Template) (Template, error)
	Delete(context.Context, string, string) error
	StoreExtraction(context.Context, workqueue.Job, string, string, string) (bool, error)
	RecordFailure(context.Context, workqueue.Job, string, string, string) error
	Count(context.Context, string) (Counts, error)
}
