package knowledge

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func validText(value string, limit int) bool {
	if strings.TrimSpace(value) == "" || len(value) > limit || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

func digest(value string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(value))) }

func (s *Service) Import(ctx context.Context, workspaceID, profileID string, input Import) (Source, error) {
	if !validText(input.Name, 200) || !validText(input.Text, MaxSourceBytes) {
		return Source{}, ErrInvalid
	}
	created := time.Now().UTC()
	source := Source{ID: id.New("src"), Name: strings.TrimSpace(input.Name), Text: input.Text, Hash: digest(input.Text), MediaType: "text/plain", ParserVersion: "plain-lines-v1", State: "ready", CreatedAt: created}
	segments := []Segment{}
	facts := []Fact{}
	for index, line := range strings.Split(strings.ReplaceAll(input.Text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 4000 || len(segments) >= 200 {
			return Source{}, ErrInvalid
		}
		segment := Segment{ID: id.New("seg"), SourceID: source.ID, Paragraph: index + 1, Text: line, Hash: digest(line), Confidence: 1}
		factID, versionID := id.New("fact"), id.New("fv")
		// Verbatim statements are review candidates, never verified claims or model inferences.
		version := Version{ID: versionID, FactID: factID, Number: 1, Statement: line, Status: "extracted", Sensitive: true, CreatedAt: created}
		segments = append(segments, segment)
		facts = append(facts, Fact{ID: factID, EvidenceSegmentIDs: []string{segment.ID}, CurrentVersionID: versionID, Versions: []Version{version}})
	}
	return s.repository.Import(ctx, workspaceID, profileID, source, segments, facts)
}

func (s *Service) Snapshot(ctx context.Context, workspaceID, profileID string) (Snapshot, error) {
	return s.repository.Snapshot(ctx, workspaceID, profileID)
}

func (s *Service) Review(ctx context.Context, workspaceID, profileID, factID string, input Review) (Version, error) {
	if input.ExpectedVersionID == "" || !validText(input.Statement, 4000) {
		return Version{}, ErrInvalid
	}
	switch input.Status {
	case "user_asserted", "user_confirmed", "disputed", "rejected":
	default:
		return Version{}, ErrInvalid
	}
	input.Statement = strings.TrimSpace(input.Statement)
	return s.repository.Review(ctx, workspaceID, profileID, factID, input)
}

func (s *Service) DeleteSource(ctx context.Context, workspaceID, profileID, sourceID string) error {
	return s.repository.DeleteSource(ctx, workspaceID, profileID, sourceID)
}

func (s *Service) Search(ctx context.Context, workspaceID, profileID, query string) ([]Version, error) {
	if !validText(query, 200) {
		return nil, ErrInvalid
	}
	return s.repository.Search(ctx, workspaceID, profileID, strings.TrimSpace(query))
}
