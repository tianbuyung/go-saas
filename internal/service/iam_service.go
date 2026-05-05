package service

import (
	"context"
	"errors"
	"strings"

	"saas/internal/db"
	"saas/internal/domain"
	"saas/internal/iam"
	"saas/internal/repository"
)

type IamService struct {
	userRepo repository.UserRepository
	jwt      *iam.JWT
}

func NewIamService(q *db.Queries, jwt *iam.JWT) *IamService {
	userRepo := repository.NewUserRepository(q)

	return &IamService{
		userRepo: userRepo,
		jwt:      jwt,
	}
}

func (s *IamService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	hash, err := iam.GeneratePassword(password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, email, hash.Hash, hash.Salt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *IamService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	userAuth, err := s.userRepo.GetUserAuthByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if !iam.ComparePassword(password, userAuth.Password, userAuth.Salt) {
		return "", errors.New("invalid credentials")
	}

	token, err := s.jwt.GenerateToken(userAuth.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
