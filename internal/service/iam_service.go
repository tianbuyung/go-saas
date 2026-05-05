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

type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Image    string
}

type LoginInput struct {
	Email    string
	Password string
}

type IamService struct {
	userRepo    repository.UserRepository
	accountRepo repository.AccountRepository
	txManager   db.TxManager
	jwt         *iam.JWT
}

func NewIamService(q *db.Queries, tx db.TxManager, jwt *iam.JWT) *IamService {
	return &IamService{
		userRepo:    repository.NewUserRepository(q),
		accountRepo: repository.NewAccountRepository(q),
		txManager:   tx,
		jwt:         jwt,
	}
}

func (s *IamService) Register(ctx context.Context, input RegisterInput) (*domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	hash, err := iam.GeneratePassword(input.Password)
	if err != nil {
		return nil, err
	}

	var result *domain.User
	err = s.txManager.WithTx(ctx, func(q *db.Queries) error {
		userRepo := repository.NewUserRepository(q)
		accountRepo := repository.NewAccountRepository(q)

		user, err := userRepo.Create(ctx, repository.CreateUserInput{
			Name:  input.Name,
			Email: email,
			Image: input.Image,
		})
		if err != nil {
			return err
		}

		_, err = accountRepo.Create(ctx, repository.CreateAccountInput{
			UserID:       user.ID,
			AccountID:    email,
			ProviderID:   "credential",
			PasswordHash: hash.Hash,
			Salt:         hash.Salt,
		})
		if err != nil {
			return err
		}

		result = user
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *IamService) Login(ctx context.Context, input LoginInput) (string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	account, err := s.accountRepo.GetByProvider(ctx, "credential", email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if !iam.ComparePassword(input.Password, account.Password, account.Salt) {
		return "", errors.New("invalid credentials")
	}

	user, err := s.userRepo.GetByID(ctx, account.UserID)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	token, err := s.jwt.GenerateToken(user.PublicID)
	if err != nil {
		return "", err
	}

	return token, nil
}
