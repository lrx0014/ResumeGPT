package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint       string
	PublicEndpoint string
	Bucket         string
	AccessKey      string
	SecretKey      string
	Region         string
}

func (s *BlobSigner) Put(ctx context.Context, workspaceID, objectID, contentType string, body io.Reader, size int64) error {
	key, err := blobstore.ObjectKey(workspaceID, objectID)
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("store object: %w", err)
	}
	return nil
}

type BlobSigner struct {
	client       *minio.Client
	publicClient *minio.Client
	bucket       string
	region       string
}

func NewBlobSigner(cfg Config) (*BlobSigner, error) {
	client, err := newClient(cfg.Endpoint, cfg)
	if err != nil {
		return nil, fmt.Errorf("create internal object storage client: %w", err)
	}
	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}
	publicClient, err := newClient(publicEndpoint, cfg)
	if err != nil {
		return nil, fmt.Errorf("create public object storage client: %w", err)
	}
	return &BlobSigner{client: client, publicClient: publicClient, bucket: cfg.Bucket, region: cfg.Region}, nil
}

func newClient(endpoint string, cfg Config) (*minio.Client, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil || endpointURL.Host == "" || (endpointURL.Scheme != "http" && endpointURL.Scheme != "https") {
		return nil, fmt.Errorf("parse endpoint")
	}
	client, err := minio.New(endpointURL.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: endpointURL.Scheme == "https",
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
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
	signed, err := s.publicClient.PresignedPutObject(ctx, s.bucket, key, expiry)
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
	signed, err := s.publicClient.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return blobstore.SignedURL{}, fmt.Errorf("presign download: %w", err)
	}
	return blobstore.SignedURL{ObjectID: objectID, URL: signed.String(), ExpiresAt: time.Now().UTC().Add(expiry)}, nil
}

func (s *BlobSigner) Open(ctx context.Context, workspaceID, objectID string) (blobstore.Object, error) {
	key, err := blobstore.ObjectKey(workspaceID, objectID)
	if err != nil {
		return blobstore.Object{}, err
	}
	stat, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return blobstore.Object{}, fmt.Errorf("stat staged object: %w", err)
	}
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return blobstore.Object{}, fmt.Errorf("open staged object: %w", err)
	}
	return blobstore.Object{Body: object, Size: stat.Size}, nil
}
