package job

import "time"

type Job struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspaceId"`
	Title          string    `json:"title"`
	Company        string    `json:"company"`
	Location       string    `json:"location,omitempty"`
	Country        string    `json:"country,omitempty"`
	City           string    `json:"city,omitempty"`
	WorkMode       string    `json:"workMode,omitempty"`
	EmploymentType string    `json:"employmentType,omitempty"`
	SourceURL      string    `json:"sourceUrl,omitempty"`
	Description    string    `json:"description,omitempty"`
	HasDescription bool      `json:"hasDescription"`
	Status         string    `json:"status"`
	ImportState    string    `json:"importState"`
	ImportError    string    `json:"importError,omitempty"`
	Origin         string    `json:"origin"`
	HunterID       string    `json:"hunterId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type SaveInput struct {
	Title          string `json:"title"`
	Company        string `json:"company"`
	Location       string `json:"location"`
	Country        string `json:"country"`
	City           string `json:"city"`
	WorkMode       string `json:"workMode"`
	EmploymentType string `json:"employmentType"`
	SourceURL      string `json:"sourceUrl"`
	Description    string `json:"description"`
	Status         string `json:"status"`
}

type CreateInput = SaveInput
type UpdateInput = SaveInput

type StatusInput struct {
	Status string `json:"status"`
}

type ImportInput struct {
	URLs         []string `json:"urls"`
	AIAssisted   bool     `json:"aiAssisted"`
	ConnectionID string   `json:"connectionId"`
	Model        string   `json:"model"`
}

type ImportPayload struct {
	JobID        string `json:"jobId"`
	SourceURL    string `json:"sourceUrl"`
	Mode         string `json:"mode,omitempty"`
	ConnectionID string `json:"connectionId,omitempty"`
	Model        string `json:"model,omitempty"`
}

type ParsedJob struct {
	Title          string
	Company        string
	Location       string
	Country        string
	City           string
	WorkMode       string
	EmploymentType string
	Description    string
}
