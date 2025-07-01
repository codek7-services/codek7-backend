// internal/service/user.go
package service

import (
	"context"
	"time"

	"github.com/lumbrjx/codek7/repo/internal/model"
	"github.com/lumbrjx/codek7/repo/internal/repository"
	"github.com/lumbrjx/codek7/repo/pkg/logger"
)

type UserService interface {
	CreateUser(ctx context.Context, password, email, username string) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, password, email, username string) (*model.User, error) {
	start := time.Now()

	logger.Logger.Info("Creating user",
		"username", username,
		"email", email,
	)

	user, err := s.repo.CreateUser(ctx, password, email, username)

	logger.LogUserOperation(ctx, "create", "", username, time.Since(start), err)

	if err != nil {
		logger.Logger.Error("User creation failed",
			"username", username,
			"email", email,
			"error", err.Error(),
		)
		return nil, err
	}

	logger.Logger.Info("User created successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	return user, nil
}

func (s *userService) GetUser(ctx context.Context, username string) (*model.User, error) {
	start := time.Now()

	logger.Logger.Info("Fetching user",
		"username", username,
	)

	user, err := s.repo.GetUser(ctx, username)

	logger.LogUserOperation(ctx, "get", "", username, time.Since(start), err)

	if err != nil {
		logger.Logger.Warn("User not found",
			"username", username,
			"error", err.Error(),
		)
		return nil, err
	}

	logger.Logger.Info("User fetched successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	return user, nil
}
