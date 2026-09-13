package knowledge

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("invalid knowledge input")
	ErrNotFound = errors.New("knowledge resource not found")
	ErrConflict = errors.New("the fact changed; reload before reviewing")
)

const MaxSourceBytes = 64 * 1024

type Source struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Text          string    `json:"text"`
	Hash          string    `json:"hash"`
	MediaType     string    `json:"mediaType"`
	ParserVersion string    `json:"parserVersion"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Segment struct {
	ID          string         `json:"id"`
	SourceID    string         `json:"sourceId"`
	Page        *int           `json:"page,omitempty"`
	Paragraph   int            `json:"paragraph"`
	Text        string         `json:"text"`
	Hash        string         `json:"hash"`
	Confidence  float64        `json:"confidence"`
	BoundingBox map[string]int `json:"boundingBox,omitempty"`
}

type Version struct {
	ID        string    `json:"id"`
	FactID    string    `json:"factId"`
	Number    int       `json:"number"`
	Statement string    `json:"statement"`
	Status    string    `json:"status"`
	Sensitive bool      `json:"sensitive"`
	ActorID   string    `json:"actorId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Fact struct {
	ID                 string    `json:"id"`
	EvidenceSegmentIDs []string  `json:"evidenceSegmentIds"`
	CurrentVersionID   string    `json:"currentVersionId"`
	Versions           []Version `json:"versions"`
}

type Snapshot struct {
	SchemaVersion int       `json:"schemaVersion"`
	ProfileID     string    `json:"profileId"`
	Sources       []Source  `json:"sources"`
	Segments      []Segment `json:"segments"`
	Facts         []Fact    `json:"facts"`
}

type Import struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type Review struct {
	ExpectedVersionID string `json:"expectedVersionId"`
	Statement         string `json:"statement"`
	Status            string `json:"status"`
	Sensitive         bool   `json:"sensitive"`
}

type Repository interface {
	Import(context.Context, string, string, Source, []Segment, []Fact) (Source, error)
	Snapshot(context.Context, string, string) (Snapshot, error)
	Review(context.Context, string, string, string, Review) (Version, error)
	Search(context.Context, string, string, string) ([]Version, error)
	DeleteSource(context.Context, string, string, string) error
}
