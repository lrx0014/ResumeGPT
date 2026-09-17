package hunter

import "time"

const JobKind = "job.hunt.v1"

type Hunter struct {
	ID               string     `json:"id"`
	WorkspaceID      string     `json:"workspaceId"`
	Name             string     `json:"name"`
	RoleQuery        string     `json:"roleQuery"`
	Location         string     `json:"location,omitempty"`
	WorkMode         string     `json:"workMode,omitempty"`
	EmploymentType   string     `json:"employmentType,omitempty"`
	ExperienceYears  *int       `json:"experienceYears,omitempty"`
	Keywords         string     `json:"keywords,omitempty"`
	AdditionalPrompt string     `json:"additionalPrompt,omitempty"`
	ProfileID        string     `json:"profileId,omitempty"`
	ConnectionID     string     `json:"connectionId"`
	Model            string     `json:"model"`
	MaxResults       int        `json:"maxResults"`
	IntervalMinutes  int        `json:"intervalMinutes"`
	Enabled          bool       `json:"enabled"`
	NextRunAt        time.Time  `json:"nextRunAt"`
	LastRunAt        *time.Time `json:"lastRunAt,omitempty"`
	LastState        string     `json:"lastState"`
	LastError        string     `json:"lastError,omitempty"`
	LastFoundCount   int        `json:"lastFoundCount"`
	ReviewCount      int        `json:"reviewCount"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type ReviewItem struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspaceId"`
	HunterID       string    `json:"hunterId"`
	SourceURL      string    `json:"sourceUrl"`
	FailureCode    string    `json:"failureCode"`
	FailureMessage string    `json:"failureMessage"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type SaveInput struct {
	Name             string `json:"name"`
	RoleQuery        string `json:"roleQuery"`
	Location         string `json:"location"`
	WorkMode         string `json:"workMode"`
	EmploymentType   string `json:"employmentType"`
	ExperienceYears  *int   `json:"experienceYears"`
	Keywords         string `json:"keywords"`
	AdditionalPrompt string `json:"additionalPrompt"`
	ProfileID        string `json:"profileId"`
	ConnectionID     string `json:"connectionId"`
	Model            string `json:"model"`
	MaxResults       int    `json:"maxResults"`
	IntervalMinutes  int    `json:"intervalMinutes"`
	Enabled          bool   `json:"enabled"`
}

type Payload struct {
	HunterID string `json:"hunterId"`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type DiscoveredJob struct {
	JobID     string
	TaskID    string
	SourceURL string
	Payload   []byte
}
