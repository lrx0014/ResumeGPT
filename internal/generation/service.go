package generation

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

type Service struct {
	repository    Repository
	profiles      *profile.Service
	opportunities *job.Service
	templates     *resumetemplate.Service
	settings      *settings.Service
	blobs         blobstore.Reader
}

func NewService(repository Repository, profiles *profile.Service, opportunities *job.Service, templates *resumetemplate.Service, settingsService *settings.Service, blobs blobstore.Reader) *Service {
	return &Service{repository: repository, profiles: profiles, opportunities: opportunities, templates: templates, settings: settingsService, blobs: blobs}
}

func (s *Service) Create(ctx context.Context, workspaceID string, input CreateInput) (Run, error) {
	input, profileJSON, opportunityJSON, templateJSON, err := s.prepareInputs(ctx, workspaceID, input)
	if err != nil {
		return Run{}, err
	}
	now := time.Now().UTC()
	run := Run{ID: id.New("gen"), WorkspaceID: workspaceID, ProfileID: input.ProfileID, OpportunityID: input.OpportunityID, TemplateID: input.TemplateID,
		DocumentType: input.DocumentType, Language: input.Language, PageTarget: input.PageTarget, CustomInstructions: input.CustomInstructions,
		PipelineMode: input.PipelineMode, Writer: input.Writer, Renderer: input.Renderer, Reviewer: input.Reviewer, State: "queued", Stage: "queued",
		ProfileSnapshot: profileJSON, OpportunitySnapshot: opportunityJSON, TemplateSnapshot: templateJSON, CreatedAt: now, UpdatedAt: now}
	payload, _ := json.Marshal(Payload{RunID: run.ID})
	task := workqueue.Job{ID: id.New("task"), WorkspaceID: workspaceID, Kind: JobKind, IdempotencyKey: run.ID, Payload: payload, MaxAttempts: 3, AvailableAt: now}
	return s.repository.Create(ctx, run, task)
}

func (s *Service) prepareInputs(ctx context.Context, workspaceID string, input CreateInput) (CreateInput, json.RawMessage, json.RawMessage, json.RawMessage, error) {
	input.ProfileID, input.OpportunityID, input.TemplateID = strings.TrimSpace(input.ProfileID), strings.TrimSpace(input.OpportunityID), strings.TrimSpace(input.TemplateID)
	input.DocumentType, input.Language, input.PageTarget = strings.TrimSpace(input.DocumentType), strings.TrimSpace(input.Language), strings.TrimSpace(input.PageTarget)
	input.CustomInstructions, input.PipelineMode = strings.TrimSpace(input.CustomInstructions), strings.TrimSpace(input.PipelineMode)
	if input.DocumentType != "resume" && input.DocumentType != "cover_letter" || input.Language == "" || len(input.Language) > 40 ||
		input.PageTarget != "one_page" && input.PageTarget != "two_pages" && input.PageTarget != "flexible" || len(input.CustomInstructions) > 4000 ||
		input.PipelineMode != "single" && input.PipelineMode != "multi" || !validChoice(input.Writer) {
		return CreateInput{}, nil, nil, nil, ErrInvalid
	}
	if input.PipelineMode == "single" {
		input.Renderer, input.Reviewer = input.Writer, input.Writer
	}
	if !validChoice(input.Renderer) || !validChoice(input.Reviewer) {
		return CreateInput{}, nil, nil, nil, ErrInvalid
	}
	profileValue, err := s.profiles.Get(ctx, workspaceID, input.ProfileID)
	if err != nil || strings.TrimSpace(profileValue.Content) == "" {
		return CreateInput{}, nil, nil, nil, ErrState
	}
	opportunity, err := s.opportunities.Get(ctx, workspaceID, input.OpportunityID)
	if err != nil || strings.TrimSpace(opportunity.Description) == "" {
		return CreateInput{}, nil, nil, nil, ErrState
	}
	var templateValue resumetemplate.Template
	if input.TemplateID != "" {
		templateValue, err = s.templates.Get(ctx, workspaceID, input.TemplateID)
		if err != nil || templateValue.State != "ready" || templateValue.Kind != input.DocumentType || templateValue.Format != "latex" {
			return CreateInput{}, nil, nil, nil, ErrState
		}
	}
	for _, choice := range []ModelChoice{input.Writer, input.Renderer, input.Reviewer} {
		if _, err := s.settings.RuntimeConnection(ctx, workspaceID, choice.ConnectionID); err != nil {
			return CreateInput{}, nil, nil, nil, ErrState
		}
	}
	profileJSON, _ := json.Marshal(profileValue)
	opportunityJSON, _ := json.Marshal(opportunity)
	templateJSON := json.RawMessage(`{}`)
	if input.TemplateID != "" {
		templateJSON, _ = json.Marshal(templateValue)
	}
	return input, profileJSON, opportunityJSON, templateJSON, nil
}

func validChoice(value ModelChoice) bool {
	return strings.TrimSpace(value.ConnectionID) != "" && strings.TrimSpace(value.Model) != "" && len(value.Model) <= 200
}
func (s *Service) List(ctx context.Context, workspaceID string) ([]Run, error) {
	return s.repository.List(ctx, workspaceID)
}
func (s *Service) Get(ctx context.Context, workspaceID, id string) (Run, error) {
	return s.repository.Get(ctx, workspaceID, id)
}
func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	return s.repository.Delete(ctx, workspaceID, strings.TrimSpace(id))
}
func (s *Service) Reconfigure(ctx context.Context, workspaceID, generationID string, input CreateInput) (Run, error) {
	current, err := s.repository.Get(ctx, workspaceID, strings.TrimSpace(generationID))
	if err != nil {
		return Run{}, err
	}
	if current.State == "queued" || current.State == "running" {
		return Run{}, ErrState
	}
	input.OpportunityID = current.OpportunityID
	input, profileJSON, opportunityJSON, templateJSON, err := s.prepareInputs(ctx, workspaceID, input)
	if err != nil {
		return Run{}, err
	}
	current.ProfileID, current.TemplateID = input.ProfileID, input.TemplateID
	current.DocumentType, current.Language, current.PageTarget = input.DocumentType, input.Language, input.PageTarget
	current.CustomInstructions, current.PipelineMode = input.CustomInstructions, input.PipelineMode
	current.Writer, current.Renderer, current.Reviewer = input.Writer, input.Renderer, input.Reviewer
	current.ProfileSnapshot, current.OpportunitySnapshot, current.TemplateSnapshot = profileJSON, opportunityJSON, templateJSON
	current.UpdatedAt = time.Now().UTC()
	return s.repository.Reconfigure(ctx, current)
}
func (s *Service) Retry(ctx context.Context, workspaceID, id string) (Run, error) {
	return s.repository.Retry(ctx, workspaceID, strings.TrimSpace(id))
}
func (s *Service) Revise(ctx context.Context, workspaceID, id, prompt string) (Run, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || len(prompt) > 4000 {
		return Run{}, ErrInvalid
	}
	return s.repository.Revise(ctx, workspaceID, strings.TrimSpace(id), prompt)
}
func (s *Service) Steps(ctx context.Context, workspaceID, id string) ([]Step, error) {
	if _, err := s.repository.Get(ctx, workspaceID, id); err != nil {
		return nil, err
	}
	return s.repository.ListSteps(ctx, workspaceID, id)
}
func (s *Service) StepArtifact(ctx context.Context, workspaceID, generationID, stepID string) (blobstore.Object, error) {
	step, err := s.repository.GetStep(ctx, workspaceID, generationID, stepID)
	if err != nil {
		return blobstore.Object{}, err
	}
	if step.Kind != "rendered_pdf" || step.ArtifactObjectID == "" {
		return blobstore.Object{}, ErrState
	}
	return s.blobs.Open(ctx, workspaceID, step.ArtifactObjectID)
}
func (s *Service) Artifact(ctx context.Context, workspaceID, id string) (blobstore.Object, error) {
	run, err := s.repository.Get(ctx, workspaceID, id)
	if err != nil {
		return blobstore.Object{}, err
	}
	if run.State != "ready" || run.ArtifactObjectID == "" {
		return blobstore.Object{}, ErrState
	}
	return s.blobs.Open(ctx, workspaceID, run.ArtifactObjectID)
}
