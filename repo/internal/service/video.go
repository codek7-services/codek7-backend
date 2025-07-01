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
	// Original video upload - creates metadata in DB
	UploadOriginalVideo(ctx context.Context, userID, title, description, originalFileName string, content []byte) (*model.Video, error)

	// Generated files upload - only saves to MinIO, no DB metadata
	UploadGeneratedFile(ctx context.Context, fileName string, content []byte) error

	// Query operations
	GetVideoByID(ctx context.Context, videoID string) (*model.Video, error)
	GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error)
	GetLast3VideosByUser(ctx context.Context, userID string) ([]*model.Video, error)
	// Download operations
	DownloadFile(ctx context.Context, fileName string) ([]byte, string, error)

	// Remove operations
	RemoveVideo(ctx context.Context, videoID string) error
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

// UploadOriginalVideo handles the initial video upload with metadata
func (s *videoService) UploadOriginalVideo(ctx context.Context, userID, title, description, fileName string, content []byte) (*model.Video, error) {
	if userID == "" || title == "" || fileName == "" || len(content) == 0 {
		return nil, fmt.Errorf("invalid input: userID, title, and fileName cannot be empty")
	}

	// Generate unique video ID
	videoID := uuid.NewV4().String()

	// Create the original filename with video ID
	originalFileName := fmt.Sprintf("%s_original.mp4", videoID)

	// Upload original file to MinIO
	if err := s.store.Upload(ctx, originalFileName, content); err != nil {
		return nil, fmt.Errorf("upload to MinIO failed: %w", err)
	}

	// Create video metadata (only for original videos)
	video := &model.Video{
		ID:          videoID,
		UserID:      userID,
		Title:       title,
		Description: description,
		FinalName:   originalFileName,
		CreatedAt:   time.Now(),
	}

	// Store metadata in database
	v, err := s.repo.CreateVideo(ctx, video)
	if err != nil {
		// Cleanup MinIO on database failure
		if cleanupErr := s.store.Remove(ctx, originalFileName); cleanupErr != nil {
			return nil, fmt.Errorf("failed to create video metadata: %w, and failed to cleanup MinIO: %v", err, cleanupErr)
		}
		return nil, fmt.Errorf("failed to create video metadata: %w", err)
	}

	return v, nil
}

// UploadGeneratedFile handles generated files (HLS segments, different qualities) - no DB metadata
func (s *videoService) UploadGeneratedFile(ctx context.Context, fileName string, content []byte) error {
	if fileName == "" || len(content) == 0 {
		return fmt.Errorf("invalid input: fileName and content cannot be empty")
	}

	// Just upload to MinIO - no database metadata for generated files
	if err := s.store.Upload(ctx, fileName, content); err != nil {
		return fmt.Errorf("upload generated file to MinIO failed: %w", err)
	}

	return nil
}

func (s *videoService) GetVideoByID(ctx context.Context, videoID string) (*model.Video, error) {
	if videoID == "" {
		return nil, fmt.Errorf("videoID cannot be empty")
	}
	return s.repo.GetVideoByID(ctx, videoID)
}

func (s *videoService) GetLast3VideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}
	return s.repo.GetLast3VideosByUser(ctx, userID)
}
func (s *videoService) GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}
	return s.repo.GetVideosByUser(ctx, userID)
}

// DownloadFile downloads any file (original, segments, playlists) from MinIO
func (s *videoService) DownloadFile(ctx context.Context, fileName string) ([]byte, string, error) {
	if fileName == "" {
		return nil, "", fmt.Errorf("fileName cannot be empty")
	}

	fileContent, err := s.store.Download(ctx, fileName)
	if err != nil {
		return nil, "", fmt.Errorf("download from MinIO failed: %w", err)
	}

	return fileContent, fileName, nil
}

func (s *videoService) RemoveVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("videoID cannot be empty")
	}

	// Get video metadata
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("video not found: %w", err)
	}

	// Remove original file
	if err := s.store.Remove(ctx, video.FinalName); err != nil {
		return fmt.Errorf("remove original file from MinIO failed: %w", err)
	}

	// Remove all generated files for this video ID
	// This includes HLS segments, playlists, and different quality versions
	if err := s.removeGeneratedFiles(ctx, videoID); err != nil {
		return fmt.Errorf("failed to remove generated files: %w", err)
	}

	// Remove metadata from database
	if err := s.repo.DeleteVideo(ctx, videoID); err != nil {
		return fmt.Errorf("failed to delete video metadata: %w", err)
	}

	return nil
}

// removeGeneratedFiles removes all files related to a video ID
func (s *videoService) removeGeneratedFiles(ctx context.Context, videoID string) error {
	// List of patterns for generated files based on your Rust code
	patterns := []string{
		fmt.Sprintf("%s_master.m3u8", videoID), // Master playlist
		fmt.Sprintf("%s_144p.mp4", videoID),    // Resolution files
		fmt.Sprintf("%s_240p.mp4", videoID),
		fmt.Sprintf("%s_360p.mp4", videoID),
		fmt.Sprintf("%s_480p.mp4", videoID),
		fmt.Sprintf("%s_720p.mp4", videoID),
		fmt.Sprintf("%s_1080p.mp4", videoID),
	}

	// Remove individual files
	for _, pattern := range patterns {
		if err := s.store.Remove(ctx, pattern); err != nil {
			// Log error but don't fail the entire operation
			fmt.Printf("Warning: failed to remove file %s: %v\n", pattern, err)
		}
	}

	// Remove HLS segment directories (144, 240, 360, etc.)
	resolutions := []string{"144", "240", "360", "480", "720", "1080"}
	for _, res := range resolutions {
		// Remove playlist file
		playlistFile := fmt.Sprintf("%s/%s/index.m3u8", videoID, res)
		if err := s.store.Remove(ctx, playlistFile); err != nil {
			fmt.Printf("Warning: failed to remove playlist %s: %v\n", playlistFile, err)
		}

		// Note: MinIO doesn't have directory concept, but segment files follow pattern
		// videoID/resolution/seg_XXX.ts - you might need to list and remove these
		// This depends on your MinIO client implementation
	}

	return nil
}
