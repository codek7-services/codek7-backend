// internal/service/user.go
package service

import (
	"context"

	"github.com/lumbrjx/codek7/repo/internal/model"
	"github.com/lumbrjx/codek7/repo/internal/repository"
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
	user, err := s.repo.CreateUser(ctx, password, email, username)
	return user, err
}

func (s *userService) GetUser(ctx context.Context, username string) (*model.User, error) {
	println("Fetching user with username:", username)
	return s.repo.GetUser(ctx, username)
}
