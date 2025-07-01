// internal/repository/video.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lumbrjx/codek7/repo/internal/model"
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
	query := `INSERT INTO videos (id, user_id, file_name, title, description, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query, v.ID, v.UserID, v.FileName, v.Title, v.Description, v.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert video failed: %w", err)
	}
	return v, nil
}

func (r *videoRepo) GetVideoByID(ctx context.Context, videoID string) (*model.Video, error) {
	query := `SELECT id, user_id, title, description, created_at, file_name FROM videos WHERE id=$1`
	row := r.db.QueryRow(ctx, query, videoID)

	var v model.Video
	err := row.Scan(&v.ID, &v.UserID, &v.Title, &v.Description, &v.CreatedAt, &v.FileName)
	if err != nil {
		return nil, fmt.Errorf("get video failed: %w", err)
	}
	return &v, nil
}
func (r *videoRepo) GetLast3VideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	query := `
SELECT id, user_id, title, description, created_at, file_name
FROM videos
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 3
`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query videos failed: %w", err)
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.Description, &v.CreatedAt, &v.FileName); err != nil {
			return nil, err
		}
		videos = append(videos, &v)
	}
	return videos, nil
}
func (r *videoRepo) GetVideosByUser(ctx context.Context, userID string) ([]*model.Video, error) {
	query := `SELECT id, user_id, title, description, created_at, file_name FROM videos WHERE user_id=$1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query videos failed: %w", err)
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.Description, &v.CreatedAt, &v.FileName); err != nil {
			return nil, err
		}
		videos = append(videos, &v)
	}
	return videos, nil
}

func (r *videoRepo) DeleteVideo(ctx context.Context, videoID string) error {
	query := `DELETE FROM videos WHERE id=$1`
	_, err := r.db.Exec(ctx, query, videoID)
	if err != nil {
		return fmt.Errorf("delete video failed: %w", err)
	}
	return nil
}
