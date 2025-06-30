package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioClient, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinioClient{
		client: minioClient,
		bucket: bucket,
	}, nil
}

// Upload uploads a video file using objectKey
func (m *MinioClient) Upload(ctx context.Context, objectKey string, content []byte) error {
	_, err := m.client.PutObject(ctx, m.bucket, objectKey, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	return nil
}

// Download retrieves the object using its objectKey
func (m *MinioClient) Download(ctx context.Context, objectKey string) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object failed: %w", err)
	}
	defer func() {
		err = obj.Close()
		if err != nil {
			fmt.Printf("failed to close object: %v\n", err)
		}
	}()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, obj); err != nil {
		return nil, fmt.Errorf("copy object content failed: %w", err)
	}

	return buf.Bytes(), nil
}

func (m *MinioClient) Remove(ctx context.Context, objectKey string) error {
	err := m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object failed: %w", err)
	}
	return nil
}
