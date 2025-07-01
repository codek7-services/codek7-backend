// internal/repository/video.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lumbrjx/codek7/repo/internal/model"
	"github.com/lumbrjx/codek7/repo/pkg/logger"
)

type VideoRepository interface {
	CreateVideo(ctx context.Context, v *model.Video) (*model.Video, error)
	GetVideoByID(ctx context.Context, videoID string) (*model.Video, error)
	GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error)
	GetLast3VideosByUser(ctx context.Context, userID string) ([]*model.Video, error)
	DeleteVideo(ctx context.Context, videoID string) error
}

type videoRepo struct {
	db *pgxpool.Pool
}

func NewVideoRepository(pool *pgxpool.Pool) VideoRepository {
	return &videoRepo{db: pool}
}

func (r *videoRepo) CreateVideo(ctx context.Context, v *model.Video) (*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Creating video in database",
		"video_id", v.ID,
		"user_id", v.UserID,
		"title", v.Title,
		"filename", v.FileName,
	)

	query := `INSERT INTO videos (id, user_id, file_name, title, description, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query, v.ID, v.UserID, v.FileName, v.Title, v.Description, v.CreatedAt)

	logger.LogDatabaseOperation(ctx, "insert", "videos", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to insert video",
			"video_id", v.ID,
			"user_id", v.UserID,
			"title", v.Title,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("insert video failed: %w", err)
	}

	logger.Logger.Info("Video created in database successfully",
		"video_id", v.ID,
		"title", v.Title,
	)

	return v, nil
}

func (r *videoRepo) GetVideoByID(ctx context.Context, videoID string) (*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Fetching video from database",
		"video_id", videoID,
	)

	query := `SELECT id, user_id, title, description, created_at, file_name FROM videos WHERE id=$1`
	row := r.db.QueryRow(ctx, query, videoID)

	var v model.Video
	err := row.Scan(&v.ID, &v.UserID, &v.Title, &v.Description, &v.CreatedAt, &v.FileName)

	logger.LogDatabaseOperation(ctx, "select", "videos", time.Since(start), err)

	if err != nil {
		logger.Logger.Warn("Video not found in database",
			"video_id", videoID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("get video failed: %w", err)
	}

	logger.Logger.Info("Video fetched from database successfully",
		"video_id", v.ID,
		"title", v.Title,
		"user_id", v.UserID,
	)

	return &v, nil
}
func (r *videoRepo) GetLast3VideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Fetching last 3 videos for user from database",
		"user_id", userID,
	)

	query := `
SELECT id, user_id, title, description, created_at, file_name
FROM videos
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 3
`
	rows, err := r.db.Query(ctx, query, userID)

	logger.LogDatabaseOperation(ctx, "select", "videos", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to query last 3 videos",
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("query videos failed: %w", err)
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.Description, &v.CreatedAt, &v.FileName); err != nil {
			logger.Logger.Error("Failed to scan video row",
				"user_id", userID,
				"error", err.Error(),
			)
			return nil, err
		}
		videos = append(videos, &v)
	}

	logger.Logger.Info("Last 3 videos fetched from database successfully",
		"user_id", userID,
		"video_count", len(videos),
	)

	return videos, nil
}
func (r *videoRepo) GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	start := time.Now()

	logger.Logger.Info("Fetching all videos for user from database",
		"user_id", userID,
	)

	query := `SELECT id, user_id, title, description, created_at, file_name FROM videos WHERE user_id=$1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)

	logger.LogDatabaseOperation(ctx, "select", "videos", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to query user videos",
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("query videos failed: %w", err)
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.Description, &v.CreatedAt, &v.FileName); err != nil {
			logger.Logger.Error("Failed to scan video row",
				"user_id", userID,
				"error", err.Error(),
			)
			return nil, err
		}
		videos = append(videos, &v)
	}

	logger.Logger.Info("User videos fetched from database successfully",
		"user_id", userID,
		"video_count", len(videos),
	)

	return videos, nil
}

func (r *videoRepo) DeleteVideo(ctx context.Context, videoID string) error {
	start := time.Now()

	logger.Logger.Info("Deleting video from database",
		"video_id", videoID,
	)

	query := `DELETE FROM videos WHERE id=$1`
	_, err := r.db.Exec(ctx, query, videoID)

	logger.LogDatabaseOperation(ctx, "delete", "videos", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to delete video",
			"video_id", videoID,
			"error", err.Error(),
		)
		return fmt.Errorf("delete video failed: %w", err)
	}

	logger.Logger.Info("Video deleted from database successfully",
		"video_id", videoID,
	)

	return nil
}
