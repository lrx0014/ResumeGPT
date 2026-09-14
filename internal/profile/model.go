package profile

import "time"

type Profile struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspaceId"`
	Name            string    `json:"name"`
	TargetRole      string    `json:"targetRole,omitempty"`
	DefaultLanguage string    `json:"defaultLanguage"`
	Content         string    `json:"content"`
	AvatarObjectID  string    `json:"avatarObjectId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type SaveInput struct {
	Name            string `json:"name"`
	TargetRole      string `json:"targetRole"`
	DefaultLanguage string `json:"defaultLanguage"`
	Content         string `json:"content"`
	AvatarObjectID  string `json:"avatarObjectId"`
}

type CreateInput = SaveInput
type UpdateInput = SaveInput
