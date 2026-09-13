package memory

import (
	"context"
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
			result = append(result, item)
		}
	}
	return result, nil
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
