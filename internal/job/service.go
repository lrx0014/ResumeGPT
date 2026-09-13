package job

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

var ErrInvalidInput = errors.New("job title and company are required")

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) List(ctx context.Context, workspaceID string) ([]Job, error) {
	return s.repository.List(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, workspaceID, jobID string) (Job, error) {
	return s.repository.Get(ctx, workspaceID, jobID)
}

func (s *Service) Create(ctx context.Context, workspaceID string, input CreateInput) (Job, error) {
	title := strings.TrimSpace(input.Title)
	company := strings.TrimSpace(input.Company)
	if title == "" || company == "" {
		return Job{}, ErrInvalidInput
	}
	now := s.now().UTC()
	return s.repository.Create(ctx, Job{
		ID:          id.New("job"),
		WorkspaceID: workspaceID,
		Title:       title,
		Company:     company,
		Location:    strings.TrimSpace(input.Location),
		SourceURL:   strings.TrimSpace(input.SourceURL),
		Description: strings.TrimSpace(input.Description),
		Status:      "interested",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}
