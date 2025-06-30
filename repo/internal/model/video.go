package model

import "time"

type Video struct {
	ID          string
	UserID      string
	Title       string
	Description string
	CreatedAt   time.Time
}
