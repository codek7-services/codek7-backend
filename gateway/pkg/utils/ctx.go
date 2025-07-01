package utils

import (
	"context"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
)

// WithUserID adds a user ID to a context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from a context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
