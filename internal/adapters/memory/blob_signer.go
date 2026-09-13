package memory

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
)

type BlobSigner struct{}

func (BlobSigner) EnsureBucket(context.Context) error {
	return nil
}

func (BlobSigner) PresignUpload(_ context.Context, workspaceID, objectID, contentType string, expiry time.Duration) (blobstore.SignedURL, error) {
	key, err := blobstore.ObjectKey(workspaceID, objectID)
	if err != nil {
		return blobstore.SignedURL{}, err
	}
	return blobstore.SignedURL{
		ObjectID:  objectID,
		URL:       fmt.Sprintf("http://localhost:9000/development/%s?operation=upload", url.PathEscape(key)),
		ExpiresAt: time.Now().UTC().Add(expiry),
		Headers:   map[string]string{"Content-Type": contentType},
	}, nil
}

func (BlobSigner) PresignDownload(_ context.Context, workspaceID, objectID string, expiry time.Duration) (blobstore.SignedURL, error) {
	key, err := blobstore.ObjectKey(workspaceID, objectID)
	if err != nil {
		return blobstore.SignedURL{}, err
	}
	return blobstore.SignedURL{
		ObjectID:  objectID,
		URL:       fmt.Sprintf("http://localhost:9000/development/%s?operation=download", url.PathEscape(key)),
		ExpiresAt: time.Now().UTC().Add(expiry),
	}, nil
}
