package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lumbrjx/codek7/repo/internal/model"
	"github.com/lumbrjx/codek7/repo/internal/repository"
	"github.com/lumbrjx/codek7/repo/internal/storage"
	uuid "github.com/satori/go.uuid"
)

type VideoService interface {
	UploadVideo(ctx context.Context, userID, title, description, originalFileName string, content []byte) (*model.Video, error)
	GetVideoByID(ctx context.Context, videoID string) (*model.Video, error)
	GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error)
	DownloadVideo(ctx context.Context, videoID string) ([]byte, string, error)
}

type videoService struct {
	repo  repository.VideoRepository
	store *storage.MinioClient
}

func NewVideoService(repo repository.VideoRepository, store *storage.MinioClient) VideoService {
	return &videoService{
		repo:  repo,
		store: store,
	}
}

func (s *videoService) UploadVideo(ctx context.Context, userID, title, description, originalFileName string, content []byte) (*model.Video, error) {
	videoID := uuid.NewV4().String()
	objectKey := fmt.Sprintf("%s.mp4", videoID)

	// Upload to MinIO
	if err := s.store.Upload(ctx, objectKey, content); err != nil {
		return nil, fmt.Errorf("upload to MinIO failed: %w", err)
	}

	// Store metadata
	video := &model.Video{
		ID:          videoID,
		UserID:      userID,
		Title:       title,
		Description: description,
		CreatedAt:   time.Now(),
	}
	return s.repo.CreateVideo(ctx, video)
}

func (s *videoService) GetVideoByID(ctx context.Context, videoID string) (*model.Video, error) {
	return s.repo.GetVideoByID(ctx, videoID)
}

func (s *videoService) GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	return s.repo.GetVideosByUser(ctx, userID)
}

func (s *videoService) DownloadVideo(ctx context.Context, videoID string) ([]byte, string, error) {
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, "", fmt.Errorf("video not found: %w", err)
	}

	objectKey := fmt.Sprintf("%s.mp4", video.ID)
	fileContent, err := s.store.Download(ctx, objectKey)
	if err != nil {
		return nil, "", fmt.Errorf("download from MinIO failed: %w", err)
	}

	return fileContent, objectKey, nil
}
