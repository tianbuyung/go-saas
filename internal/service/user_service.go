package service

import (
	"context"
	"saas/internal/domain"
	"saas/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(r repository.UserRepository) *UserService {
	return &UserService{repo: r}
}

func (s *UserService) GetMe(ctx context.Context, userID int64) (*domain.User, error) {
	return s.repo.GetByID(ctx, userID)
}
