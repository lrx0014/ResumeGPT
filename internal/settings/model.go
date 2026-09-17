package settings

import "time"

type Preferences struct {
	WorkspaceID       string    `json:"workspaceId"`
	InterfaceLanguage string    `json:"interfaceLanguage"`
	Theme             string    `json:"theme"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type PreferencesInput struct {
	InterfaceLanguage string `json:"interfaceLanguage"`
	Theme             string `json:"theme"`
}

type LLMConnection struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspaceId"`
	Name               string    `json:"name"`
	ExecutionMode      string    `json:"executionMode"`
	Provider           string    `json:"provider"`
	BaseURL            string    `json:"baseUrl"`
	APITokenConfigured bool      `json:"apiTokenConfigured"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type LLMConnectionInput struct {
	Name          string `json:"name"`
	ExecutionMode string `json:"executionMode"`
	Provider      string `json:"provider"`
	BaseURL       string `json:"baseUrl"`
	APIToken      string `json:"apiToken"`
	ClearAPIToken bool   `json:"clearApiToken"`
}

type StoredConnection struct {
	Connection      LLMConnection
	TokenCiphertext []byte
}

type ConnectionTest struct {
	Status string   `json:"status"`
	Models []string `json:"models"`
}

type RuntimeConnection struct {
	Connection LLMConnection
	APIToken   string
}

const (
	AgentWriter           = "writer"
	AgentTemplateApplier  = "template_applier"
	AgentDocumentDesigner = "document_designer"
	AgentVisualReviewer   = "visual_reviewer"
	AgentJobImport        = "job_import"
	AgentJobHunter        = "job_hunter"
)

type AgentDefault struct {
	Agent        string    `json:"agent"`
	ConnectionID string    `json:"connectionId"`
	Model        string    `json:"model"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type AgentDefaultInput struct {
	Agent        string `json:"agent"`
	ConnectionID string `json:"connectionId"`
	Model        string `json:"model"`
}

type AgentDefaultsInput struct {
	Items []AgentDefaultInput `json:"items"`
}
