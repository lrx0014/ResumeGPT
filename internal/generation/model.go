package generation

import (
	"encoding/json"
	"errors"
	"time"
)

const JobKind = "generation.run.v1"

var (
	ErrInvalid  = errors.New("invalid generation")
	ErrNotFound = errors.New("generation not found")
	ErrState    = errors.New("generation input is not ready")
)

type ModelChoice struct {
	ConnectionID string `json:"connectionId"`
	Model        string `json:"model"`
}

type CreateInput struct {
	ProfileID          string      `json:"profileId"`
	OpportunityID      string      `json:"opportunityId"`
	TemplateID         string      `json:"templateId"`
	DocumentType       string      `json:"documentType"`
	Language           string      `json:"language"`
	PageTarget         string      `json:"pageTarget"`
	CustomInstructions string      `json:"customInstructions"`
	PipelineMode       string      `json:"pipelineMode"`
	Writer             ModelChoice `json:"writer"`
	Renderer           ModelChoice `json:"renderer"`
	Reviewer           ModelChoice `json:"reviewer"`
}

type Run struct {
	ID                  string          `json:"id"`
	WorkspaceID         string          `json:"workspaceId"`
	ProfileID           string          `json:"profileId"`
	OpportunityID       string          `json:"opportunityId"`
	TemplateID          string          `json:"templateId"`
	DocumentType        string          `json:"documentType"`
	Language            string          `json:"language"`
	PageTarget          string          `json:"pageTarget"`
	CustomInstructions  string          `json:"customInstructions,omitempty"`
	PipelineMode        string          `json:"pipelineMode"`
	Writer              ModelChoice     `json:"writer"`
	Renderer            ModelChoice     `json:"renderer"`
	Reviewer            ModelChoice     `json:"reviewer"`
	State               string          `json:"state"`
	Stage               string          `json:"stage"`
	Draft               string          `json:"draft,omitempty"`
	RenderedSource      string          `json:"renderedSource,omitempty"`
	Review              string          `json:"review,omitempty"`
	RepairCount         int             `json:"repairCount"`
	ArtifactObjectID    string          `json:"artifactObjectId,omitempty"`
	ErrorCode           string          `json:"errorCode,omitempty"`
	ErrorMessage        string          `json:"errorMessage,omitempty"`
	ProfileSnapshot     json.RawMessage `json:"-"`
	OpportunitySnapshot json.RawMessage `json:"-"`
	TemplateSnapshot    json.RawMessage `json:"-"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
}

type Payload struct {
	RunID          string `json:"runId"`
	UseFallback    bool   `json:"useFallback,omitempty"`
	RevisionPrompt string `json:"revisionPrompt,omitempty"`
}

type Step struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspaceId"`
	GenerationID     string    `json:"generationId"`
	Kind             string    `json:"kind"`
	Sequence         int       `json:"sequence"`
	Content          string    `json:"content,omitempty"`
	Feedback         string    `json:"feedback,omitempty"`
	ArtifactObjectID string    `json:"artifactObjectId,omitempty"`
	RepairCount      int       `json:"repairCount"`
	CreatedAt        time.Time `json:"createdAt"`
}
