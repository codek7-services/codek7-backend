package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/lumbrjx/codek7/repo/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioClient, error) {
	logger.Logger.Info("Initializing MinIO client",
		"endpoint", endpoint,
		"bucket", bucket,
		"use_ssl", useSSL,
	)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Logger.Error("Failed to create MinIO client",
			"endpoint", endpoint,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		logger.Logger.Error("Failed to check bucket existence",
			"bucket", bucket,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		logger.Logger.Info("Bucket does not exist, creating it",
			"bucket", bucket,
		)
		if err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			logger.Logger.Error("Failed to create bucket",
				"bucket", bucket,
				"error", err.Error(),
			)
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Logger.Info("Bucket created successfully",
			"bucket", bucket,
		)
	}

	logger.Logger.Info("MinIO client initialized successfully",
		"endpoint", endpoint,
		"bucket", bucket,
	)

	return &MinioClient{
		client: minioClient,
		bucket: bucket,
	}, nil
}

// Upload uploads a video file using objectKey
func (m *MinioClient) Upload(ctx context.Context, objectKey string, content []byte) error {
	start := time.Now()
	fileSize := int64(len(content))

	logger.Logger.Info("Uploading file to MinIO",
		"object_key", objectKey,
		"file_size_bytes", fileSize,
		"bucket", m.bucket,
	)

	_, err := m.client.PutObject(ctx, m.bucket, objectKey, bytes.NewReader(content), fileSize, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})

	logger.LogStorageOperation(ctx, "upload", objectKey, fileSize, time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to upload file to MinIO",
			"object_key", objectKey,
			"bucket", m.bucket,
			"error", err.Error(),
		)
		return fmt.Errorf("upload failed: %w", err)
	}

	logger.Logger.Info("File uploaded to MinIO successfully",
		"object_key", objectKey,
		"file_size_bytes", fileSize,
	)

	return nil
}

// Download retrieves the object using its objectKey
func (m *MinioClient) Download(ctx context.Context, objectKey string) ([]byte, error) {
	start := time.Now()

	logger.Logger.Info("Downloading file from MinIO",
		"object_key", objectKey,
		"bucket", m.bucket,
	)

	obj, err := m.client.GetObject(ctx, m.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		logger.Logger.Error("Failed to get object from MinIO",
			"object_key", objectKey,
			"bucket", m.bucket,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("get object failed: %w", err)
	}
	defer func() {
		if closeErr := obj.Close(); closeErr != nil {
			logger.Logger.Warn("Failed to close object",
				"object_key", objectKey,
				"error", closeErr.Error(),
			)
		}
	}()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, obj); err != nil {
		logger.Logger.Error("Failed to copy object content",
			"object_key", objectKey,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("copy object content failed: %w", err)
	}

	fileSize := int64(buf.Len())
	logger.LogStorageOperation(ctx, "download", objectKey, fileSize, time.Since(start), nil)
	logger.Logger.Info("File downloaded from MinIO successfully",
		"object_key", objectKey,
		"file_size_bytes", fileSize,
	)

	return buf.Bytes(), nil
}

func (m *MinioClient) Remove(ctx context.Context, objectKey string) error {
	start := time.Now()

	logger.Logger.Info("Removing file from MinIO",
		"object_key", objectKey,
		"bucket", m.bucket,
	)

	err := m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})

	logger.LogStorageOperation(ctx, "remove", objectKey, 0, time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to remove file from MinIO",
			"object_key", objectKey,
			"bucket", m.bucket,
			"error", err.Error(),
		)
		return fmt.Errorf("remove object failed: %w", err)
	}

	logger.Logger.Info("File removed from MinIO successfully",
		"object_key", objectKey,
	)

	return nil
}
