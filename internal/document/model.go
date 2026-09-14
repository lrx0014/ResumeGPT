package document

import (
	"context"
	"errors"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

const ExtractJobKind = "profile.document.extract.v1"

var (
	ErrInvalid  = errors.New("invalid document upload")
	ErrNotFound = errors.New("document upload not found")
	ErrState    = errors.New("document upload is not ready for this operation")
)

type Upload struct {
	ID                string    `json:"id"`
	WorkspaceID       string    `json:"workspaceId"`
	ProfileID         string    `json:"profileId"`
	ObjectID          string    `json:"objectId"`
	Name              string    `json:"name"`
	DeclaredMediaType string    `json:"declaredMediaType"`
	State             string    `json:"state"`
	JobID             string    `json:"jobId,omitempty"`
	ExtractedText     string    `json:"extractedText,omitempty"`
	ErrorCode         string    `json:"errorCode,omitempty"`
	ErrorMessage      string    `json:"errorMessage,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type StageInput struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
}

type StagedUpload struct {
	Upload Upload              `json:"upload"`
	Target blobstore.SignedURL `json:"target"`
}

type ExtractPayload struct {
	UploadID  string `json:"uploadId"`
	ProfileID string `json:"profileId"`
	ObjectID  string `json:"objectId"`
	Name      string `json:"name"`
}

type Extraction struct {
	MediaType     string              `json:"media_type"`
	SHA256        string              `json:"sha256"`
	ParserVersion string              `json:"parser_version"`
	MalwareStatus string              `json:"malware_status"`
	Segments      []ExtractionSegment `json:"segments"`
}

type ExtractionSegment struct {
	Text        string         `json:"text"`
	Page        *int           `json:"page"`
	Paragraph   int            `json:"paragraph"`
	Confidence  float64        `json:"confidence"`
	BoundingBox map[string]int `json:"bounding_box"`
}

type Repository interface {
	Stage(context.Context, Upload) error
	Queue(context.Context, string, string, string, workqueue.Job) (Upload, error)
	Get(context.Context, string, string, string) (Upload, error)
	DownloadAllowed(context.Context, string, string) (bool, error)
	StoreExtraction(context.Context, workqueue.Job, string, string) (bool, error)
	RecordFailure(context.Context, workqueue.Job, string, string, string) error
}
