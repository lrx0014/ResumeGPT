package s3

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

type BlobSigner struct {
	client *minio.Client
	bucket string
	region string
}

func NewBlobSigner(cfg Config) (*BlobSigner, error) {
	endpointURL, err := url.Parse(cfg.Endpoint)
	if err != nil || endpointURL.Host == "" {
		return nil, fmt.Errorf("parse object storage endpoint")
	}
	client, err := minio.New(endpointURL.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: endpointURL.Scheme == "https",
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create object storage client: %w", err)
	}
	return &BlobSigner{client: client, bucket: cfg.Bucket, region: cfg.Region}, nil
}

func (s *BlobSigner) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check object storage bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region}); err != nil {
		return fmt.Errorf("create object storage bucket: %w", err)
	}
	return nil
}

func (s *BlobSigner) PresignUpload(ctx context.Context, workspaceID, objectID, contentType string, expiry time.Duration) (blobstore.SignedURL, error) {
	key, err := blobstore.ObjectKey(workspaceID, objectID)
	if err != nil {
		return blobstore.SignedURL{}, err
	}
	signed, err := s.client.PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return blobstore.SignedURL{}, fmt.Errorf("presign upload: %w", err)
	}
	headers := map[string]string{}
	if strings.TrimSpace(contentType) != "" {
		headers["Content-Type"] = contentType
	}
	return blobstore.SignedURL{ObjectID: objectID, URL: signed.String(), ExpiresAt: time.Now().UTC().Add(expiry), Headers: headers}, nil
}

func (s *BlobSigner) PresignDownload(ctx context.Context, workspaceID, objectID string, expiry time.Duration) (blobstore.SignedURL, error) {
	key, err := blobstore.ObjectKey(workspaceID, objectID)
	if err != nil {
		return blobstore.SignedURL{}, err
	}
	signed, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return blobstore.SignedURL{}, fmt.Errorf("presign download: %w", err)
	}
	return blobstore.SignedURL{ObjectID: objectID, URL: signed.String(), ExpiresAt: time.Now().UTC().Add(expiry)}, nil
}
