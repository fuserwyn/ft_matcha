package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	client   *minio.Client
	endpoint string
	bucket   string
	useSSL   bool
}

func NewMinIO(endpoint, accessKey, secretKey, bucket string) (*MinIO, error) {
	useSSL := strings.HasPrefix(endpoint, "https://")
	cleanEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	client, err := minio.New(cleanEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinIO{
		client:   client,
		endpoint: cleanEndpoint,
		bucket:   bucket,
		useSSL:   useSSL,
	}, nil
}

func (m *MinIO) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
}

func (m *MinIO) PutObject(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := m.client.PutObject(ctx, m.bucket, objectKey, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return m.ObjectURL(objectKey), nil
}

func (m *MinIO) RemoveObject(ctx context.Context, objectKey string) error {
	return m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (m *MinIO) ObjectURL(objectKey string) string {
	scheme := "http"
	if m.useSSL {
		scheme = "https"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   m.endpoint,
		Path:   path.Join("/", m.bucket, objectKey),
	}
	return u.String()
}

func BuildPhotoObjectKey(userID, photoID string, fileName string) string {
	ext := ""
	if idx := strings.LastIndex(fileName, "."); idx >= 0 && idx < len(fileName)-1 {
		ext = fileName[idx:]
	}
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("users/%s/%s%s", userID, photoID, strings.ToLower(ext))
}
