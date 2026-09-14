package document

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

var allowedExtensions = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".tex": true,
	".png": true, ".jpg": true, ".jpeg": true, ".txt": true,
	".md": true,
}

type Service struct {
	repository Repository
	blobs      blobstore.Signer
}

func NewService(repository Repository, blobs blobstore.Signer) *Service {
	return &Service{repository: repository, blobs: blobs}
}

func (s *Service) Stage(ctx context.Context, workspaceID, profileID string, input StageInput) (StagedUpload, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ContentType = strings.TrimSpace(input.ContentType)
	invalidName := input.Name == "" || len(input.Name) > 200 || !utf8.ValidString(input.Name) || strings.ContainsAny(input.Name, "/\\")
	for _, character := range input.Name {
		invalidName = invalidName || unicode.IsControl(character)
	}
	if invalidName ||
		input.ContentType == "" || len(input.ContentType) > 200 || !allowedExtensions[strings.ToLower(filepath.Ext(input.Name))] {
		return StagedUpload{}, ErrInvalid
	}
	now := time.Now().UTC()
	upload := Upload{
		ID: id.New("upl"), WorkspaceID: workspaceID, ProfileID: profileID,
		ObjectID: blobstore.NewObjectID(), Name: input.Name, DeclaredMediaType: input.ContentType,
		State: "staged", CreatedAt: now, UpdatedAt: now,
	}
	target, err := s.blobs.PresignUpload(ctx, workspaceID, upload.ObjectID, input.ContentType, 15*time.Minute)
	if err != nil {
		return StagedUpload{}, err
	}
	if err := s.repository.Stage(ctx, upload); err != nil {
		return StagedUpload{}, err
	}
	return StagedUpload{Upload: upload, Target: target}, nil
}

func (s *Service) Queue(ctx context.Context, workspaceID, profileID, uploadID string) (Upload, error) {
	payload, err := json.Marshal(ExtractPayload{UploadID: uploadID, ProfileID: profileID})
	if err != nil {
		return Upload{}, err
	}
	job := workqueue.Job{
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: ExtractJobKind,
		IdempotencyKey: uploadID, Payload: payload, MaxAttempts: 5,
		AvailableAt: time.Now().UTC(),
	}
	return s.repository.Queue(ctx, workspaceID, profileID, uploadID, job)
}

func (s *Service) Get(ctx context.Context, workspaceID, profileID, uploadID string) (Upload, error) {
	return s.repository.Get(ctx, workspaceID, profileID, uploadID)
}

func (s *Service) DownloadAllowed(ctx context.Context, workspaceID, objectID string) (bool, error) {
	return s.repository.DownloadAllowed(ctx, workspaceID, objectID)
}
