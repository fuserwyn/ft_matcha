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
	client        *minio.Client
	endpoint      string
	bucket        string
	useSSL        bool
	publicBaseURL string
}

func NewMinIO(endpoint, accessKey, secretKey, bucket, publicBaseURL string) (*MinIO, error) {
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
		client:        client,
		endpoint:      cleanEndpoint,
		bucket:        bucket,
		useSSL:        useSSL,
		publicBaseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
	}, nil
}

func (m *MinIO) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if exists {
		return m.ensurePublicReadPolicy(ctx)
	}
	if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
		return err
	}
	return m.ensurePublicReadPolicy(ctx)
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

func (m *MinIO) GetObject(ctx context.Context, objectKey string) (*minio.Object, error) {
	return m.client.GetObject(ctx, m.bucket, objectKey, minio.GetObjectOptions{})
}

func (m *MinIO) ObjectURL(objectKey string) string {
	if m.publicBaseURL != "" {
		base, err := url.Parse(m.publicBaseURL)
		if err == nil {
			base.Path = path.Join(base.Path, m.bucket, objectKey)
			return base.String()
		}
	}
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

func (m *MinIO) ensurePublicReadPolicy(ctx context.Context) error {
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]},{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:ListBucket"],"Resource":["arn:aws:s3:::%s"]}]}`, m.bucket, m.bucket)
	return m.client.SetBucketPolicy(ctx, m.bucket, policy)
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
