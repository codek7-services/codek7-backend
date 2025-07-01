package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lumbrjx/codek7/repo/internal/model"
	"github.com/lumbrjx/codek7/repo/internal/repository"
	"github.com/lumbrjx/codek7/repo/internal/storage"
	"github.com/lumbrjx/codek7/repo/pkg/logger"
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
	start := time.Now()
	fileSize := int64(len(content))

	logger.Logger.Info("Starting original video upload",
		"user_id", userID,
		"title", title,
		"filename", fileName,
		"file_size_bytes", fileSize,
	)

	if userID == "" || title == "" || fileName == "" || len(content) == 0 {
		err := fmt.Errorf("invalid input: userID, title, and fileName cannot be empty")
		logger.Logger.Error("Invalid video upload parameters",
			"user_id", userID,
			"title", title,
			"filename", fileName,
			"content_size", len(content),
		)
		return nil, err
	}

	// Generate unique video ID
	videoID := uuid.NewV4().String()

	base := strings.TrimSuffix(fileName, ".mp4")
	// Create the original filename with video ID
	originalFileName := fmt.Sprintf("%s_original.mp4", base)

	logger.Logger.Info("Uploading original video to storage",
		"video_id", videoID,
		"original_filename", originalFileName,
		"user_id", userID,
	)

	// Upload original file to MinIO
	if err := s.store.Upload(ctx, originalFileName, content); err != nil {
		logger.Logger.Error("Failed to upload video to MinIO",
			"video_id", videoID,
			"filename", originalFileName,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("upload to MinIO failed: %w", err)
	}

	logger.Logger.Info("Video uploaded to storage successfully",
		"video_id", videoID,
		"filename", originalFileName,
	)

	// Create video metadata (only for original videos)
	video := &model.Video{
		ID:          videoID,
		UserID:      userID,
		Title:       title,
		Description: description,
		FileName:    originalFileName,
		CreatedAt:   time.Now(),
	}

	logger.Logger.Info("Creating video metadata in database",
		"video_id", videoID,
	)

	// Store metadata in database
	v, err := s.repo.CreateVideo(ctx, video)
	if err != nil {
		logger.Logger.Error("Failed to create video metadata, cleaning up storage",
			"video_id", videoID,
			"filename", originalFileName,
			"error", err.Error(),
		)
		// Cleanup MinIO on database failure
		if cleanupErr := s.store.Remove(ctx, originalFileName); cleanupErr != nil {
			logger.Logger.Error("Failed to cleanup storage after database error",
				"video_id", videoID,
				"filename", originalFileName,
				"cleanup_error", cleanupErr.Error(),
			)
			return nil, fmt.Errorf("failed to create video metadata: %w, and failed to cleanup MinIO: %v", err, cleanupErr)
		}
		return nil, fmt.Errorf("failed to create video metadata: %w", err)
	}

	logger.LogVideoOperation(ctx, "upload_original", videoID, userID, fileSize, time.Since(start), nil)
	logger.Logger.Info("Original video uploaded successfully",
		"video_id", videoID,
		"title", title,
		"filename", originalFileName,
		"user_id", userID,
	)

	return v, nil
}

// UploadGeneratedFile handles generated files (HLS segments, different qualities) - no DB metadata
func (s *videoService) UploadGeneratedFile(ctx context.Context, fileName string, content []byte) error {
	start := time.Now()
	fileSize := int64(len(content))

	logger.Logger.Info("Uploading generated file",
		"filename", fileName,
		"file_size_bytes", fileSize,
	)

	if fileName == "" || len(content) == 0 {
		err := fmt.Errorf("invalid input: fileName and content cannot be empty")
		logger.Logger.Error("Invalid generated file upload parameters",
			"filename", fileName,
			"content_size", len(content),
		)
		return err
	}

	// Just upload to MinIO - no database metadata for generated files
	if err := s.store.Upload(ctx, fileName, content); err != nil {
		logger.Logger.Error("Failed to upload generated file",
			"filename", fileName,
			"error", err.Error(),
		)
		return fmt.Errorf("upload generated file to MinIO failed: %w", err)
	}

	logger.LogStorageOperation(ctx, "upload_generated", fileName, fileSize, time.Since(start), nil)
	logger.Logger.Info("Generated file uploaded successfully",
		"filename", fileName,
		"file_size_bytes", fileSize,
	)

	return nil
}

func (s *videoService) GetVideoByID(ctx context.Context, videoID string) (*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Fetching video by ID",
		"video_id", videoID,
	)

	if videoID == "" {
		err := fmt.Errorf("videoID cannot be empty")
		logger.Logger.Error("Invalid video ID", "video_id", videoID)
		return nil, err
	}

	video, err := s.repo.GetVideoByID(ctx, videoID)

	logger.LogVideoOperation(ctx, "get_by_id", videoID, "", 0, time.Since(start), err)

	if err != nil {
		logger.Logger.Warn("Video not found",
			"video_id", videoID,
			"error", err.Error(),
		)
		return nil, err
	}

	logger.Logger.Info("Video fetched successfully",
		"video_id", video.ID,
		"title", video.Title,
		"user_id", video.UserID,
	)

	return video, nil
}

func (s *videoService) GetLast3VideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Fetching last 3 videos for user",
		"user_id", userID,
	)

	if userID == "" {
		err := fmt.Errorf("userID cannot be empty")
		logger.Logger.Error("Invalid user ID for last 3 videos", "user_id", userID)
		return nil, err
	}

	videos, err := s.repo.GetLast3VideosByUser(ctx, userID)

	logger.LogVideoOperation(ctx, "get_last_3", "", userID, 0, time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to fetch last 3 videos",
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, err
	}

	logger.Logger.Info("Last 3 videos fetched successfully",
		"user_id", userID,
		"video_count", len(videos),
	)

	return videos, nil
}
func (s *videoService) GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Fetching all videos for user",
		"user_id", userID,
	)

	if userID == "" {
		err := fmt.Errorf("userID cannot be empty")
		logger.Logger.Error("Invalid user ID for videos", "user_id", userID)
		return nil, err
	}

	videos, err := s.repo.GetVideosByUser(ctx, userID)

	logger.LogVideoOperation(ctx, "get_by_user", "", userID, 0, time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to fetch user videos",
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, err
	}

	logger.Logger.Info("User videos fetched successfully",
		"user_id", userID,
		"video_count", len(videos),
	)

	return videos, nil
}

// DownloadFile downloads any file (original, segments, playlists) from MinIO
func (s *videoService) DownloadFile(ctx context.Context, fileName string) ([]byte, string, error) {
	start := time.Now()

	logger.Logger.Info("Downloading file",
		"filename", fileName,
	)

	if fileName == "" {
		err := fmt.Errorf("fileName cannot be empty")
		logger.Logger.Error("Invalid filename for download", "filename", fileName)
		return nil, "", err
	}

	fileContent, err := s.store.Download(ctx, fileName)

	fileSize := int64(0)
	if fileContent != nil {
		fileSize = int64(len(fileContent))
	}

	logger.LogStorageOperation(ctx, "download", fileName, fileSize, time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to download file",
			"filename", fileName,
			"error", err.Error(),
		)
		return nil, "", fmt.Errorf("download from MinIO failed: %w", err)
	}

	logger.Logger.Info("File downloaded successfully",
		"filename", fileName,
		"file_size_bytes", fileSize,
	)

	return fileContent, fileName, nil
}

func (s *videoService) RemoveVideo(ctx context.Context, videoID string) error {
	start := time.Now()

	logger.Logger.Info("Removing video",
		"video_id", videoID,
	)

	if videoID == "" {
		err := fmt.Errorf("videoID cannot be empty")
		logger.Logger.Error("Invalid video ID for removal", "video_id", videoID)
		return err
	}

	// Get video metadata
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		logger.Logger.Error("Video not found for removal",
			"video_id", videoID,
			"error", err.Error(),
		)
		return fmt.Errorf("video not found: %w", err)
	}

	logger.Logger.Info("Removing original file from storage",
		"video_id", videoID,
		"filename", video.FileName,
	)

	// Remove original file
	if err := s.store.Remove(ctx, video.FileName); err != nil {
		logger.Logger.Error("Failed to remove original file",
			"video_id", videoID,
			"filename", video.FileName,
			"error", err.Error(),
		)
		return fmt.Errorf("remove original file from MinIO failed: %w", err)
	}

	// Remove all generated files for this video ID
	// This includes HLS segments, playlists, and different quality versions
	logger.Logger.Info("Removing generated files",
		"video_id", videoID,
	)

	if err := s.removeGeneratedFiles(ctx, videoID); err != nil {
		logger.Logger.Error("Failed to remove generated files",
			"video_id", videoID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to remove generated files: %w", err)
	}

	// Remove metadata from database
	logger.Logger.Info("Removing video metadata from database",
		"video_id", videoID,
	)

	if err := s.repo.DeleteVideo(ctx, videoID); err != nil {
		logger.Logger.Error("Failed to delete video metadata",
			"video_id", videoID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to delete video metadata: %w", err)
	}

	logger.LogVideoOperation(ctx, "remove", videoID, video.UserID, 0, time.Since(start), nil)
	logger.Logger.Info("Video removed successfully",
		"video_id", videoID,
		"title", video.Title,
		"user_id", video.UserID,
	)

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
