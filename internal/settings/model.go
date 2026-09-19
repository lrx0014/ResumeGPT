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

// MaxTokensCeiling bounds how large a MaxTokens override may be, keeping a
// misconfigured value from producing runaway request costs.
const MaxTokensCeiling = 32768

// DefaultAgentMaxTokens are the built-in max-output-token budgets used when
// an AgentDefault row has no MaxTokens override (0). Callers should resolve
// an effective budget via EffectiveMaxTokens rather than reading this map
// directly.
var DefaultAgentMaxTokens = map[string]int{
	AgentWriter:           4096,
	AgentTemplateApplier:  12288,
	AgentDocumentDesigner: 12288,
	AgentVisualReviewer:   1536,
	AgentJobImport:        6000,
	AgentJobHunter:        5000,
}

// EffectiveMaxTokens returns configured if it's a positive override, else
// the built-in default for agent (0 if agent is unrecognized).
func EffectiveMaxTokens(agent string, configured int) int {
	if configured > 0 {
		return configured
	}
	return DefaultAgentMaxTokens[agent]
}

type AgentDefault struct {
	Agent        string    `json:"agent"`
	ConnectionID string    `json:"connectionId"`
	Model        string    `json:"model"`
	MaxTokens    int       `json:"maxTokens"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type AgentDefaultInput struct {
	Agent        string `json:"agent"`
	ConnectionID string `json:"connectionId"`
	Model        string `json:"model"`
	MaxTokens    int    `json:"maxTokens"`
}

type AgentDefaultsInput struct {
	Items []AgentDefaultInput `json:"items"`
}
