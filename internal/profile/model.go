package profile

import "time"

type Profile struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspaceId"`
	Name            string    `json:"name"`
	Domain          string    `json:"domain,omitempty"`
	DefaultLanguage string    `json:"defaultLanguage"`
	Description     string    `json:"description,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name            string `json:"name"`
	Domain          string `json:"domain"`
	DefaultLanguage string `json:"defaultLanguage"`
	Description     string `json:"description"`
}
