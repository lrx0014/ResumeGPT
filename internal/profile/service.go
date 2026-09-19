package profile

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

var ErrInvalid = errors.New("invalid profile input")

const MaxContentBytes = 1024 * 1024

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

func (s *Service) Search(ctx context.Context, workspaceID string, filter Filter) (Page, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	if len(filter.Search) > 200 {
		filter.Search = ""
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	return s.repository.Search(ctx, workspaceID, filter)
}

func (s *Service) Count(ctx context.Context, workspaceID string) (Counts, error) {
	return s.repository.Count(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, workspaceID, profileID string) (Profile, error) {
	return s.repository.Get(ctx, workspaceID, profileID)
}

func (s *Service) Create(ctx context.Context, workspaceID string, input CreateInput) (Profile, error) {
	value, err := prepare(input)
	if err != nil {
		return Profile{}, err
	}
	now := s.now().UTC()
	value.ID, value.WorkspaceID, value.CreatedAt, value.UpdatedAt = id.New("prof"), workspaceID, now, now
	return s.repository.Create(ctx, value)
}

func (s *Service) Update(ctx context.Context, workspaceID, profileID string, input UpdateInput) (Profile, error) {
	value, err := prepare(input)
	if err != nil {
		return Profile{}, err
	}
	value.ID, value.WorkspaceID, value.UpdatedAt = profileID, workspaceID, s.now().UTC()
	return s.repository.Update(ctx, value)
}

func (s *Service) Delete(ctx context.Context, workspaceID, profileID string) error {
	return s.repository.Delete(ctx, workspaceID, profileID)
}

func prepare(input SaveInput) (Profile, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.TargetRole = strings.TrimSpace(input.TargetRole)
	input.DefaultLanguage = strings.TrimSpace(input.DefaultLanguage)
	input.AvatarObjectID = strings.TrimSpace(input.AvatarObjectID)
	if input.DefaultLanguage == "" {
		input.DefaultLanguage = "en-US"
	}
	if !validField(input.Name, 200, false) || !validField(input.TargetRole, 200, true) ||
		!validField(input.DefaultLanguage, 50, false) || len(input.Content) > MaxContentBytes || !utf8.ValidString(input.Content) ||
		(input.AvatarObjectID != "" && (!strings.HasPrefix(input.AvatarObjectID, "obj_") || strings.ContainsAny(input.AvatarObjectID, "/\\"))) {
		return Profile{}, ErrInvalid
	}
	for _, character := range input.Content {
		if unicode.IsControl(character) && character != '\n' && character != '\r' && character != '\t' {
			return Profile{}, ErrInvalid
		}
	}
	return Profile{Name: input.Name, TargetRole: input.TargetRole, DefaultLanguage: input.DefaultLanguage, Content: input.Content, AvatarObjectID: input.AvatarObjectID}, nil
}

func validField(value string, limit int, optional bool) bool {
	if (!optional && value == "") || len(value) > limit || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}
