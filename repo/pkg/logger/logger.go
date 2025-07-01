package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

var Logger *slog.Logger

func init() {
	// Configure structured logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	// Use JSON handler for production-ready structured logs
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(Logger)
}

// WithContext adds request context information to logs
func WithContext(ctx context.Context) *slog.Logger {
	// Extract request ID or other context values if available
	if reqID := ctx.Value("request_id"); reqID != nil {
		return Logger.With("request_id", reqID)
	}
	return Logger
}

// LogLevel represents different log levels
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Service-specific logging helpers
func LogUserOperation(ctx context.Context, operation, userID, username string, duration time.Duration, err error) {
	logger := WithContext(ctx).With(
		"service", "user",
		"operation", operation,
		"user_id", userID,
		"username", username,
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		logger.Error("User operation failed",
			"error", err.Error(),
		)
	} else {
		logger.Info("User operation completed successfully")
	}
}

func LogVideoOperation(ctx context.Context, operation, videoID, userID string, fileSize int64, duration time.Duration, err error) {
	logger := WithContext(ctx).With(
		"service", "video",
		"operation", operation,
		"video_id", videoID,
		"user_id", userID,
		"duration_ms", duration.Milliseconds(),
	)

	if fileSize > 0 {
		logger = logger.With("file_size_bytes", fileSize)
	}

	if err != nil {
		logger.Error("Video operation failed",
			"error", err.Error(),
		)
	} else {
		logger.Info("Video operation completed successfully")
	}
}

func LogStorageOperation(ctx context.Context, operation, fileName string, fileSize int64, duration time.Duration, err error) {
	logger := WithContext(ctx).With(
		"service", "storage",
		"operation", operation,
		"file_name", fileName,
		"file_size_bytes", fileSize,
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		logger.Error("Storage operation failed",
			"error", err.Error(),
		)
	} else {
		logger.Info("Storage operation completed successfully")
	}
}

func LogDatabaseOperation(ctx context.Context, operation, table string, duration time.Duration, err error) {
	logger := WithContext(ctx).With(
		"service", "database",
		"operation", operation,
		"table", table,
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		logger.Error("Database operation failed",
			"error", err.Error(),
		)
	} else {
		logger.Debug("Database operation completed successfully")
	}
}

func LogGRPCRequest(ctx context.Context, method string, duration time.Duration, err error) {
	logger := WithContext(ctx).With(
		"service", "grpc",
		"method", method,
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		logger.Error("gRPC request failed",
			"error", err.Error(),
		)
	} else {
		logger.Info("gRPC request completed successfully")
	}
}
