package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lumbrjx/codek7/repo/internal/model"
	"github.com/lumbrjx/codek7/repo/pkg/logger"
	uuid "github.com/satori/go.uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, password, email, username string) (*model.User, error)
	GetUser(ctx context.Context, userID string) (*model.User, error)
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{db: pool}
}

func (r *userRepo) CreateUser(ctx context.Context, password, email, username string) (*model.User, error) {
	start := time.Now()

	logger.Logger.Info("Creating user in database",
		"username", username,
		"email", email,
	)

	user := &model.User{
		ID:        uuid.NewV4().String(),
		Username:  username,
		Password:  password,
		Email:     email,
		CreatedAt: time.Now(),
	}

	query := `INSERT INTO users (id, username, email, password, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(ctx, query, user.ID, user.Username, user.Email, user.Password, user.CreatedAt)

	logger.LogDatabaseOperation(ctx, "insert", "users", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to insert user",
			"username", username,
			"email", email,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("insert user failed: %w", err)
	}

	logger.Logger.Info("User created in database successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	return user, nil
}

func (r *userRepo) GetUser(ctx context.Context, userID string) (*model.User, error) {
	start := time.Now()

	logger.Logger.Info("Fetching user from database",
		"username", userID,
	)

	query := `SELECT id, username, email, password, created_at FROM users WHERE username = $1`
	row := r.db.QueryRow(ctx, query, userID)

	var user model.User

	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)

	logger.LogDatabaseOperation(ctx, "select", "users", time.Since(start), err)

	if err != nil {
		logger.Logger.Warn("User not found in database",
			"username", userID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	logger.Logger.Info("User fetched from database successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	return &user, nil
}
