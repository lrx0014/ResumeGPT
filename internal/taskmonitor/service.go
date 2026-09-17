package taskmonitor

import (
	"context"
	"encoding/json"
	"strings"
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, workspaceID string, filter Filter) (Page, error) {
	filter.State, filter.Kind, filter.Search = strings.TrimSpace(filter.State), strings.TrimSpace(filter.Kind), strings.TrimSpace(filter.Search)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	if len(filter.State) > 40 || len(filter.Kind) > 200 || len(filter.Search) > 200 {
		filter.State, filter.Kind, filter.Search = "", "", ""
	}
	page, err := s.repository.List(ctx, workspaceID, filter)
	if err != nil {
		return Page{}, err
	}
	for index := range page.Items {
		page.Items[index].SafePayload = safePayload(page.Items[index].Payload)
	}
	return page, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, taskID string) (Task, error) {
	value, err := s.repository.Get(ctx, workspaceID, taskID)
	if err != nil {
		return Task{}, err
	}
	value.SafePayload = safePayload(value.Payload)
	return value, nil
}

func safePayload(source json.RawMessage) any {
	var value any
	if len(source) == 0 || json.Unmarshal(source, &value) != nil {
		return map[string]any{}
	}
	return redact(value)
}

func redact(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "apikey") || strings.Contains(lower, "api_key") || strings.Contains(lower, "authorization") || strings.Contains(lower, "credential") {
				result[key] = "[redacted]"
			} else {
				result[key] = redact(item)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = redact(item)
		}
		return result
	default:
		return value
	}
}
