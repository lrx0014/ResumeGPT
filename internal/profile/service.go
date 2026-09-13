package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

var ErrInvalidName = errors.New("profile name is required")

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) List(ctx context.Context, workspaceID string) ([]Profile, error) {
	return s.repository.List(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, workspaceID, profileID string) (Profile, error) {
	return s.repository.Get(ctx, workspaceID, profileID)
}

func (s *Service) Create(ctx context.Context, workspaceID string, input CreateInput) (Profile, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Profile{}, ErrInvalidName
	}
	language := strings.TrimSpace(input.DefaultLanguage)
	if language == "" {
		language = "en-US"
	}
	now := s.now().UTC()
	return s.repository.Create(ctx, Profile{
		ID:              id.New("prof"),
		WorkspaceID:     workspaceID,
		Name:            name,
		Domain:          strings.TrimSpace(input.Domain),
		DefaultLanguage: language,
		Description:     strings.TrimSpace(input.Description),
		CreatedAt:       now,
		UpdatedAt:       now,
	})
}
