package hunter

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

var ErrInvalid = errors.New("invalid job hunter")

var validIntervals = map[int]bool{360: true, 720: true, 1440: true, 10080: true}

type Service struct {
	repository Repository
	profiles   ProfileReader
	now        func() time.Time
}

type ProfileReader interface {
	Get(context.Context, string, string) (profile.Profile, error)
}

func NewService(repository Repository, profiles ...ProfileReader) *Service {
	service := &Service{repository: repository, now: time.Now}
	if len(profiles) > 0 {
		service.profiles = profiles[0]
	}
	return service
}

func (s *Service) List(ctx context.Context, workspaceID string) ([]Hunter, error) {
	return s.repository.List(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, workspaceID, hunterID string) (Hunter, error) {
	return s.repository.Get(ctx, workspaceID, hunterID)
}

func (s *Service) ListReviewItems(ctx context.Context, workspaceID, hunterID string) ([]ReviewItem, error) {
	if _, err := s.repository.Get(ctx, workspaceID, hunterID); err != nil {
		return nil, err
	}
	return s.repository.ListReviewItems(ctx, workspaceID, hunterID)
}

func (s *Service) DismissReviewItem(ctx context.Context, workspaceID, hunterID, reviewID string) error {
	return s.repository.DismissReviewItem(ctx, workspaceID, hunterID, reviewID)
}

func (s *Service) Create(ctx context.Context, workspaceID string, input SaveInput) (Hunter, error) {
	value, err := prepare(input)
	if err != nil {
		return Hunter{}, err
	}
	if err := s.validateProfile(ctx, workspaceID, value.ProfileID); err != nil {
		return Hunter{}, err
	}
	now := s.now().UTC()
	value.ID, value.WorkspaceID = id.New("hunt"), workspaceID
	value.NextRunAt, value.LastState = now, "never"
	value.CreatedAt, value.UpdatedAt = now, now
	return s.repository.Create(ctx, value)
}

func (s *Service) Update(ctx context.Context, workspaceID, hunterID string, input SaveInput) (Hunter, error) {
	value, err := prepare(input)
	if err != nil {
		return Hunter{}, err
	}
	if err := s.validateProfile(ctx, workspaceID, value.ProfileID); err != nil {
		return Hunter{}, err
	}
	current, err := s.repository.Get(ctx, workspaceID, hunterID)
	if err != nil {
		return Hunter{}, err
	}
	value.ID, value.WorkspaceID, value.CreatedAt = current.ID, current.WorkspaceID, current.CreatedAt
	value.LastRunAt, value.LastState, value.LastError, value.LastFoundCount, value.ReviewCount = current.LastRunAt, current.LastState, current.LastError, current.LastFoundCount, current.ReviewCount
	value.NextRunAt, value.UpdatedAt = current.NextRunAt, s.now().UTC()
	if value.Enabled && (!current.Enabled || value.IntervalMinutes != current.IntervalMinutes) {
		value.NextRunAt = value.UpdatedAt
	}
	return s.repository.Update(ctx, value)
}

func (s *Service) Delete(ctx context.Context, workspaceID, hunterID string) error {
	return s.repository.Delete(ctx, workspaceID, hunterID)
}

func (s *Service) RunNow(ctx context.Context, workspaceID, hunterID string) error {
	value, err := s.repository.Get(ctx, workspaceID, hunterID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(Payload{HunterID: hunterID})
	now := s.now().UTC()
	task := workqueue.Job{ID: id.New("task"), WorkspaceID: workspaceID, Kind: JobKind,
		IdempotencyKey: hunterID + ":manual:" + id.New("run"), Payload: payload, MaxAttempts: 3, AvailableAt: now}
	return s.repository.RunNow(ctx, value, task)
}

func prepare(input SaveInput) (Hunter, error) {
	input.Name, input.RoleQuery, input.Location = strings.TrimSpace(input.Name), strings.TrimSpace(input.RoleQuery), strings.TrimSpace(input.Location)
	input.WorkMode, input.EmploymentType = strings.TrimSpace(input.WorkMode), strings.TrimSpace(input.EmploymentType)
	input.Keywords, input.AdditionalPrompt = strings.TrimSpace(input.Keywords), strings.TrimSpace(input.AdditionalPrompt)
	input.ProfileID = strings.TrimSpace(input.ProfileID)
	input.ConnectionID, input.Model = strings.TrimSpace(input.ConnectionID), strings.TrimSpace(input.Model)
	if input.MaxResults == 0 {
		input.MaxResults = 10
	}
	fields := []struct {
		value    string
		maximum  int
		required bool
	}{
		{input.Name, 120, true}, {input.RoleQuery, 300, true}, {input.Location, 300, false},
		{input.WorkMode, 100, false}, {input.EmploymentType, 100, false}, {input.Keywords, 1000, false},
		{input.AdditionalPrompt, 4000, false}, {input.ConnectionID, 200, true}, {input.Model, 200, true},
		{input.ProfileID, 200, false},
	}
	for _, field := range fields {
		if !validField(field.value, field.maximum, field.required) {
			return Hunter{}, ErrInvalid
		}
	}
	if !validIntervals[input.IntervalMinutes] || input.MaxResults < 1 || input.MaxResults > 10 ||
		input.MaxTokens < 0 || input.MaxTokens > settings.MaxTokensCeiling ||
		(input.ExperienceYears != nil && (*input.ExperienceYears < 0 || *input.ExperienceYears > 60)) {
		return Hunter{}, ErrInvalid
	}
	return Hunter{Name: input.Name, RoleQuery: input.RoleQuery, Location: input.Location, WorkMode: input.WorkMode,
		EmploymentType: input.EmploymentType, ExperienceYears: input.ExperienceYears, Keywords: input.Keywords,
		AdditionalPrompt: input.AdditionalPrompt, ProfileID: input.ProfileID, ConnectionID: input.ConnectionID, Model: input.Model,
		MaxTokens:       input.MaxTokens,
		MaxResults:      input.MaxResults,
		IntervalMinutes: input.IntervalMinutes, Enabled: input.Enabled}, nil
}

func (s *Service) validateProfile(ctx context.Context, workspaceID, profileID string) error {
	if profileID == "" {
		return nil
	}
	if s.profiles == nil {
		return ErrInvalid
	}
	if _, err := s.profiles.Get(ctx, workspaceID, profileID); err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			return ErrInvalid
		}
		return err
	}
	return nil
}

func validField(value string, maximum int, required bool) bool {
	if (required && value == "") || len(value) > maximum || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) && character != '\n' && character != '\r' && character != '\t' {
			return false
		}
	}
	return true
}
