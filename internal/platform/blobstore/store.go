package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

var ErrInvalidObjectID = errors.New("invalid object identifier")

type SignedURL struct {
	ObjectID  string            `json:"objectId"`
	URL       string            `json:"url"`
	ExpiresAt time.Time         `json:"expiresAt"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type Signer interface {
	EnsureBucket(ctx context.Context) error
	PresignUpload(ctx context.Context, workspaceID, objectID, contentType string, expiry time.Duration) (SignedURL, error)
	PresignDownload(ctx context.Context, workspaceID, objectID string, expiry time.Duration) (SignedURL, error)
}

type Object struct {
	Body io.ReadCloser
	Size int64
}

type Reader interface {
	Open(ctx context.Context, workspaceID, objectID string) (Object, error)
}

type Writer interface {
	Put(ctx context.Context, workspaceID, objectID, contentType string, body io.Reader, size int64) error
}

func NewObjectID() string {
	return id.New("obj")
}

func ObjectKey(workspaceID, objectID string) (string, error) {
	if !strings.HasPrefix(workspaceID, "ws_") || !strings.HasPrefix(objectID, "obj_") ||
		strings.ContainsAny(workspaceID+objectID, "/\\") {
		return "", ErrInvalidObjectID
	}
	return fmt.Sprintf("%s/%s", workspaceID, objectID), nil
}
