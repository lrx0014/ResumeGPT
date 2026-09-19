package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/lrx0014/ResumeGPT/internal/profile"
)

type ProfileRepository struct {
	mu    sync.RWMutex
	items map[string]profile.Profile
}

func NewProfileRepository() *ProfileRepository {
	return &ProfileRepository{items: make(map[string]profile.Profile)}
}

func (r *ProfileRepository) List(_ context.Context, workspaceID string) ([]profile.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]profile.Profile, 0)
	for _, item := range r.items {
		if item.WorkspaceID == workspaceID {
			item.HasContent = len(item.Content) > 0
			preview := []rune(item.Content)
			if len(preview) > 140 {
				preview = preview[:140]
			}
			item.ContentPreview = string(preview)
			item.Content = ""
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *ProfileRepository) Search(_ context.Context, workspaceID string, filter profile.Filter) (profile.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Search)
	matches := make([]profile.Profile, 0)
	for _, item := range r.items {
		if item.WorkspaceID != workspaceID {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) || strings.Contains(strings.ToLower(item.TargetRole), query) || strings.Contains(strings.ToLower(item.Content), query) {
			matches = append(matches, item)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].CreatedAt.After(matches[j].CreatedAt) })
	result := profile.Page{Items: make([]profile.Profile, 0), Total: len(matches), Page: filter.Page, PageSize: filter.PageSize}
	start := (filter.Page - 1) * filter.PageSize
	if start < len(matches) {
		end := start + filter.PageSize
		if end > len(matches) {
			end = len(matches)
		}
		result.Items = append(result.Items, matches[start:end]...)
	}
	return result, nil
}

func (r *ProfileRepository) Count(_ context.Context, workspaceID string) (profile.Counts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var counts profile.Counts
	for _, item := range r.items {
		if item.WorkspaceID != workspaceID {
			continue
		}
		counts.Total++
		if len(item.Content) > 0 {
			counts.WithContent++
		}
	}
	return counts, nil
}

func (r *ProfileRepository) Get(_ context.Context, workspaceID, profileID string) (profile.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[profileID]
	if !ok || item.WorkspaceID != workspaceID {
		return profile.Profile{}, profile.ErrNotFound
	}
	return item, nil
}

func (r *ProfileRepository) Create(_ context.Context, value profile.Profile) (profile.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[value.ID] = value
	return value, nil
}

func (r *ProfileRepository) Update(_ context.Context, value profile.Profile) (profile.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[value.ID]
	if !ok || current.WorkspaceID != value.WorkspaceID {
		return profile.Profile{}, profile.ErrNotFound
	}
	value.CreatedAt = current.CreatedAt
	r.items[value.ID] = value
	return value, nil
}

func (r *ProfileRepository) Delete(_ context.Context, workspaceID, profileID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[profileID]
	if !ok || current.WorkspaceID != workspaceID {
		return profile.ErrNotFound
	}
	delete(r.items, profileID)
	return nil
}
