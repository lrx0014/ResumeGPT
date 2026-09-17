package job

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

const ImportJobKind = "job.page.import.v1"

var (
	ErrInvalidInput    = errors.New("invalid job input")
	ErrInvalidURL      = errors.New("only public LinkedIn and Indeed HTTPS job URLs are supported")
	ErrInvalidAIConfig = errors.New("AI-assisted import requires an LLM connection and model")
	ErrTooManyURLs     = errors.New("a batch can contain at most 50 URLs")
)

var validStatuses = map[string]bool{
	"interested": true, "preparing": true, "applied": true, "screening": true,
	"interview": true, "offer": true, "accepted": true, "rejected": true, "withdrawn": true,
}

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
	value, err := prepare(input)
	if err != nil {
		return Job{}, err
	}
	now := s.now().UTC()
	value.ID, value.WorkspaceID, value.ImportState, value.Origin = id.New("job"), workspaceID, "manual", "manual"
	value.CreatedAt, value.UpdatedAt = now, now
	return s.repository.Create(ctx, value)
}

func (s *Service) Update(ctx context.Context, workspaceID, jobID string, input UpdateInput) (Job, error) {
	value, err := prepare(input)
	if err != nil {
		return Job{}, err
	}
	current, err := s.repository.Get(ctx, workspaceID, jobID)
	if err != nil {
		return Job{}, err
	}
	value.ID, value.WorkspaceID, value.CreatedAt, value.UpdatedAt = jobID, workspaceID, current.CreatedAt, s.now().UTC()
	value.Origin, value.HunterID = current.Origin, current.HunterID
	value.ImportState = "ready"
	if value.SourceURL == "" {
		value.ImportState = "manual"
	}
	return s.repository.Update(ctx, value)
}

func (s *Service) Delete(ctx context.Context, workspaceID, jobID string) error {
	return s.repository.Delete(ctx, workspaceID, jobID)
}

func prepare(input SaveInput) (Job, error) {
	input.Title, input.Company = strings.TrimSpace(input.Title), strings.TrimSpace(input.Company)
	input.Location, input.Country, input.City = strings.TrimSpace(input.Location), strings.TrimSpace(input.Country), strings.TrimSpace(input.City)
	input.WorkMode, input.EmploymentType = strings.TrimSpace(input.WorkMode), strings.TrimSpace(input.EmploymentType)
	input.SourceURL, input.Description, input.Status = strings.TrimSpace(input.SourceURL), strings.TrimSpace(input.Description), strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = "interested"
	}
	fields := []struct {
		value    string
		limit    int
		required bool
	}{
		{input.Title, 300, true}, {input.Company, 300, true}, {input.Location, 300, false},
		{input.Country, 100, false}, {input.City, 150, false}, {input.WorkMode, 100, false},
		{input.EmploymentType, 100, false}, {input.SourceURL, 2048, false},
	}
	for _, field := range fields {
		if !validText(field.value, field.limit, field.required) {
			return Job{}, ErrInvalidInput
		}
	}
	if len(input.Description) > 1024*1024 || !utf8.ValidString(input.Description) || !validStatuses[input.Status] {
		return Job{}, ErrInvalidInput
	}
	for _, character := range input.Description {
		if unicode.IsControl(character) && character != '\n' && character != '\r' && character != '\t' {
			return Job{}, ErrInvalidInput
		}
	}
	if input.SourceURL != "" {
		normalized, err := normalizeSourceURL(input.SourceURL)
		if err != nil {
			return Job{}, err
		}
		input.SourceURL = normalized
	}
	return Job{Title: input.Title, Company: input.Company, Location: input.Location, Country: input.Country,
		City: input.City, WorkMode: input.WorkMode, EmploymentType: input.EmploymentType,
		SourceURL: input.SourceURL, Description: input.Description, Status: input.Status}, nil
}

func validText(value string, limit int, required bool) bool {
	if (required && value == "") || len(value) > limit || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func NormalizeImportURL(raw string) (string, error) {
	normalized, err := normalizeSourceURL(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	parsed, _ := url.Parse(normalized)
	if !allowedJobHost(parsed.Hostname()) || (parsed.Port() != "" && parsed.Port() != "443") {
		return "", ErrInvalidURL
	}
	return normalized, nil
}

func NormalizeAIImportURL(raw string) (string, error) {
	normalized, err := normalizeSourceURL(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	parsed, _ := url.Parse(normalized)
	if parsed.Port() != "" && parsed.Port() != "443" {
		return "", ErrInvalidURL
	}
	return normalized, nil
}

func normalizeSourceURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" || len(raw) > 2048 {
		return "", ErrInvalidInput
	}
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "gclid" || lower == "fbclid" || lower == "mc_cid" || lower == "mc_eid" {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func allowedJobHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com") ||
		host == "indeed.com" || strings.HasSuffix(host, ".indeed.com")
}

type ImportService struct {
	repository ImportRepository
	now        func() time.Time
}

func NewImportService(repository ImportRepository) *ImportService {
	return &ImportService{repository: repository, now: time.Now}
}

func (s *ImportService) Create(ctx context.Context, workspaceID string, input ImportInput) ([]Job, error) {
	if len(input.URLs) == 0 || len(input.URLs) > 50 {
		return nil, ErrTooManyURLs
	}
	input.ConnectionID, input.Model = strings.TrimSpace(input.ConnectionID), strings.TrimSpace(input.Model)
	if input.AIAssisted && (!validText(input.ConnectionID, 200, true) || !validText(input.Model, 200, true)) {
		return nil, ErrInvalidAIConfig
	}
	normalizedURLs := make([]string, 0, len(input.URLs))
	seen := make(map[string]bool, len(input.URLs))
	for _, raw := range input.URLs {
		var sourceURL string
		var err error
		if input.AIAssisted {
			sourceURL, err = NormalizeAIImportURL(raw)
		} else {
			sourceURL, err = NormalizeImportURL(raw)
		}
		if err != nil {
			return nil, err
		}
		if seen[sourceURL] {
			continue
		}
		seen[sourceURL] = true
		normalizedURLs = append(normalizedURLs, sourceURL)
	}
	result := make([]Job, 0, len(normalizedURLs))
	for _, sourceURL := range normalizedURLs {
		now := s.now().UTC()
		value := Job{ID: id.New("job"), WorkspaceID: workspaceID, SourceURL: sourceURL, Status: "interested", Origin: "url_import",
			ImportState: "queued", CreatedAt: now, UpdatedAt: now}
		mode := "standard"
		if input.AIAssisted {
			mode = "agent"
		}
		payload, err := json.Marshal(ImportPayload{JobID: value.ID, SourceURL: sourceURL, Mode: mode,
			ConnectionID: input.ConnectionID, Model: input.Model})
		if err != nil {
			return nil, err
		}
		task := workqueue.Job{ID: id.New("task"), WorkspaceID: workspaceID, Kind: ImportJobKind,
			IdempotencyKey: value.ID, Payload: payload, MaxAttempts: 4, AvailableAt: now}
		created, err := s.repository.QueueImport(ctx, value, task)
		if err != nil {
			return nil, err
		}
		result = append(result, created)
	}
	return result, nil
}
