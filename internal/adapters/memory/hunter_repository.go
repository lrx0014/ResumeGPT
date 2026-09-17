package memory

import (
	"context"
	"sync"

	"github.com/lrx0014/ResumeGPT/internal/hunter"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

type HunterRepository struct {
	mu      sync.RWMutex
	items   map[string]hunter.Hunter
	active  map[string]bool
	reviews map[string]hunter.ReviewItem
}

func NewHunterRepository() *HunterRepository {
	return &HunterRepository{items: make(map[string]hunter.Hunter), active: make(map[string]bool), reviews: make(map[string]hunter.ReviewItem)}
}

func (r *HunterRepository) List(_ context.Context, workspaceID string) ([]hunter.Hunter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]hunter.Hunter, 0)
	for _, item := range r.items {
		if item.WorkspaceID == workspaceID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *HunterRepository) Get(_ context.Context, workspaceID, hunterID string) (hunter.Hunter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.items[hunterID]
	if !ok || value.WorkspaceID != workspaceID {
		return hunter.Hunter{}, hunter.ErrNotFound
	}
	return value, nil
}

func (r *HunterRepository) Create(_ context.Context, value hunter.Hunter) (hunter.Hunter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[value.ID] = value
	return value, nil
}

func (r *HunterRepository) Update(_ context.Context, value hunter.Hunter) (hunter.Hunter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.items[value.ID]; !ok || current.WorkspaceID != value.WorkspaceID {
		return hunter.Hunter{}, hunter.ErrNotFound
	}
	r.items[value.ID] = value
	if !value.Enabled {
		delete(r.active, value.ID)
	}
	return value, nil
}

func (r *HunterRepository) Delete(_ context.Context, workspaceID, hunterID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.items[hunterID]
	if !ok || value.WorkspaceID != workspaceID {
		return hunter.ErrNotFound
	}
	delete(r.items, hunterID)
	delete(r.active, hunterID)
	for reviewID, review := range r.reviews {
		if review.WorkspaceID == workspaceID && review.HunterID == hunterID {
			delete(r.reviews, reviewID)
		}
	}
	return nil
}

func (r *HunterRepository) RunNow(_ context.Context, value hunter.Hunter, _ workqueue.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active[value.ID] {
		return hunter.ErrConflict
	}
	value.LastState = "queued"
	r.items[value.ID], r.active[value.ID] = value, true
	return nil
}

func (*HunterRepository) ScheduleDue(context.Context, int) (int, error)            { return 0, nil }
func (*HunterRepository) MarkRunning(context.Context, workqueue.Job, string) error { return nil }
func (*HunterRepository) Complete(context.Context, workqueue.Job, hunter.Hunter, []hunter.DiscoveredJob) (int, error) {
	return 0, nil
}
func (*HunterRepository) RecordFailure(context.Context, workqueue.Job, string) error { return nil }

func (r *HunterRepository) ListReviewItems(_ context.Context, workspaceID, hunterID string) ([]hunter.ReviewItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if value, ok := r.items[hunterID]; !ok || value.WorkspaceID != workspaceID {
		return nil, hunter.ErrNotFound
	}
	items := make([]hunter.ReviewItem, 0)
	for _, item := range r.reviews {
		if item.WorkspaceID == workspaceID && item.HunterID == hunterID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *HunterRepository) DismissReviewItem(_ context.Context, workspaceID, hunterID, reviewID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.reviews[reviewID]
	if !ok || item.WorkspaceID != workspaceID || item.HunterID != hunterID {
		return hunter.ErrNotFound
	}
	delete(r.reviews, reviewID)
	return nil
}
