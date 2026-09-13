package job

import "time"

type Job struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	Location    string    `json:"location,omitempty"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	SourceURL   string `json:"sourceUrl"`
	Description string `json:"description"`
}
