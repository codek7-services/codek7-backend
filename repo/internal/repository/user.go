package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lumbrjx/codek7/repo/internal/model"
	uuid "github.com/satori/go.uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, username string) (*model.User, error)
	GetUser(ctx context.Context, userID string) (*model.User, error)
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{db: pool}
}

func (r *userRepo) CreateUser(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{
		ID:        uuid.NewV4().String(),
		Username:  username,
		CreatedAt: time.Now(),
	}

	query := `INSERT INTO users (id, username, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, query, user.ID, user.Username, user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user failed: %w", err)
	}

	return user, nil
}

func (r *userRepo) GetUser(ctx context.Context, userID string) (*model.User, error) {
	query := `SELECT id, username, created_at FROM users WHERE id = $1`
	row := r.db.QueryRow(ctx, query, userID)

	var user model.User
	if err := row.Scan(&user.ID, &user.Username, &user.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	return &user, nil
}
