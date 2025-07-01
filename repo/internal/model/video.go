package model

import (
	"fmt"
	"time"
)

type Video struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	FileName    string    `json:"file_name" db:"file_name"` // Original file name in MinIO
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// GetMasterPlaylistPath returns the path to the master HLS playlist
func (v *Video) GetMasterPlaylistPath() string {
	return fmt.Sprintf("%s_master.m3u8", v.ID)
}

// GetResolutionPath returns the path for a specific resolution file
func (v *Video) GetResolutionPath(resolution string) string {
	return fmt.Sprintf("%s_%sp.mp4", v.ID, resolution)
}

// GetHLSPlaylistPath returns the path for HLS playlist at specific resolution
func (v *Video) GetHLSPlaylistPath(resolution string) string {
	return fmt.Sprintf("%s/%s/index.m3u8", v.ID, resolution)
}
