package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/lrx0014/ResumeGPT/internal/settings"
)

type SettingsRepository struct {
	mu          sync.RWMutex
	preferences map[string]settings.Preferences
	connections map[string]settings.StoredConnection
}

func NewSettingsRepository() *SettingsRepository {
	return &SettingsRepository{preferences: make(map[string]settings.Preferences), connections: make(map[string]settings.StoredConnection)}
}

func (r *SettingsRepository) GetPreferences(_ context.Context, workspaceID string) (settings.Preferences, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if value, ok := r.preferences[workspaceID]; ok {
		return value, nil
	}
	return settings.Preferences{WorkspaceID: workspaceID, InterfaceLanguage: "en", Theme: "system"}, nil
}

func (r *SettingsRepository) SavePreferences(_ context.Context, value settings.Preferences) (settings.Preferences, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.preferences[value.WorkspaceID] = value
	return value, nil
}

func (r *SettingsRepository) ListConnections(_ context.Context, workspaceID string) ([]settings.StoredConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]settings.StoredConnection, 0)
	for _, item := range r.connections {
		if item.Connection.WorkspaceID == workspaceID {
			result = append(result, cloneStoredConnection(item))
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Connection.Name < result[right].Connection.Name })
	return result, nil
}

func (r *SettingsRepository) GetConnection(_ context.Context, workspaceID, connectionID string) (settings.StoredConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.connections[connectionID]
	if !ok || item.Connection.WorkspaceID != workspaceID {
		return settings.StoredConnection{}, settings.ErrNotFound
	}
	return cloneStoredConnection(item), nil
}

func (r *SettingsRepository) CreateConnection(_ context.Context, value settings.StoredConnection) (settings.StoredConnection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[value.Connection.ID] = cloneStoredConnection(value)
	return cloneStoredConnection(value), nil
}

func (r *SettingsRepository) UpdateConnection(_ context.Context, value settings.StoredConnection) (settings.StoredConnection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.connections[value.Connection.ID]
	if !ok || current.Connection.WorkspaceID != value.Connection.WorkspaceID {
		return settings.StoredConnection{}, settings.ErrNotFound
	}
	r.connections[value.Connection.ID] = cloneStoredConnection(value)
	return cloneStoredConnection(value), nil
}

func (r *SettingsRepository) DeleteConnection(_ context.Context, workspaceID, connectionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.connections[connectionID]
	if !ok || item.Connection.WorkspaceID != workspaceID {
		return settings.ErrNotFound
	}
	delete(r.connections, connectionID)
	return nil
}

func cloneStoredConnection(value settings.StoredConnection) settings.StoredConnection {
	value.TokenCiphertext = append([]byte(nil), value.TokenCiphertext...)
	return value
}
