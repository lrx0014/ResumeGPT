package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

type TemplateRepository struct {
	mu    sync.RWMutex
	items map[string]resumetemplate.Template
}

func NewTemplateRepository() *TemplateRepository {
	return &TemplateRepository{items: make(map[string]resumetemplate.Template)}
}
func (r *TemplateRepository) List(_ context.Context, workspaceID string) ([]resumetemplate.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]resumetemplate.Template, 0)
	for _, item := range r.items {
		if item.WorkspaceID == workspaceID {
			item.Content = ""
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
func (r *TemplateRepository) Get(_ context.Context, workspaceID, templateID string) (resumetemplate.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[templateID]
	if !ok || item.WorkspaceID != workspaceID {
		return resumetemplate.Template{}, resumetemplate.ErrNotFound
	}
	return item, nil
}
func (r *TemplateRepository) Stage(_ context.Context, item resumetemplate.Template) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return nil
}
func (r *TemplateRepository) Restage(_ context.Context, item resumetemplate.Template) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; !ok {
		return resumetemplate.ErrNotFound
	}
	r.items[item.ID] = item
	return nil
}
func (r *TemplateRepository) Queue(_ context.Context, workspaceID, templateID string, _ workqueue.Job) (resumetemplate.Template, error) {
	return resumetemplate.Template{}, resumetemplate.ErrState
}
func (r *TemplateRepository) Update(_ context.Context, item resumetemplate.Template) (resumetemplate.Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; !ok {
		return resumetemplate.Template{}, resumetemplate.ErrNotFound
	}
	r.items[item.ID] = item
	return item, nil
}
func (r *TemplateRepository) Delete(_ context.Context, workspaceID, templateID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[templateID]
	if !ok || item.WorkspaceID != workspaceID {
		return resumetemplate.ErrNotFound
	}
	delete(r.items, templateID)
	return nil
}
func (r *TemplateRepository) StoreExtraction(context.Context, workqueue.Job, string, string, string) (bool, error) {
	return false, resumetemplate.ErrState
}
func (r *TemplateRepository) RecordFailure(context.Context, workqueue.Job, string, string, string) error {
	return nil
}
