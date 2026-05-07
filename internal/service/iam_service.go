package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"saas/internal/db"
	"saas/internal/domain"
	"saas/internal/iam"
	"saas/internal/repository"

	"go.uber.org/zap"
)

var ErrEmailAlreadyExists = repository.ErrEmailAlreadyExists

type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Image    string
}

type LoginInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
}

type IamServiceIface interface {
	Register(ctx context.Context, input RegisterInput) (*domain.User, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Refresh(ctx context.Context, refreshToken string) (*LoginOutput, error)
	Logout(ctx context.Context, input LogoutInput) error
}

type IamService struct {
	userRepo           repository.UserRepository
	accountRepo        repository.AccountRepository
	txManager          db.TxManager
	jwt                *iam.JWT
	sessionStore       iam.SessionStore
	blocklist          iam.Blocklist
	log                *zap.Logger
	refreshExpireHours int
	makeUserRepo       func(*db.Queries) repository.UserRepository
	makeAccountRepo    func(*db.Queries) repository.AccountRepository
}

func NewIamService(
	userRepo repository.UserRepository,
	accountRepo repository.AccountRepository,
	tx db.TxManager,
	jwt *iam.JWT,
	ss iam.SessionStore,
	bl iam.Blocklist,
	log *zap.Logger,
	refreshExpireHours int,
) *IamService {
	return &IamService{
		userRepo:           userRepo,
		accountRepo:        accountRepo,
		txManager:          tx,
		jwt:                jwt,
		sessionStore:       ss,
		blocklist:          bl,
		log:                log,
		refreshExpireHours: refreshExpireHours,
		makeUserRepo:       repository.NewUserRepository,
		makeAccountRepo:    repository.NewAccountRepository,
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
		userRepo := s.makeUserRepo(q)
		accountRepo := s.makeAccountRepo(q)

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

func (s *IamService) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	account, err := s.accountRepo.GetByProvider(ctx, "credential", email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !iam.ComparePassword(input.Password, account.Password, account.Salt) {
		return nil, errors.New("invalid credentials")
	}

	user, err := s.userRepo.GetByID(ctx, account.UserID)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	refreshToken, err := iam.GenerateOpaqueToken()
	if err != nil {
		return nil, err
	}

	session := &domain.Session{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(time.Duration(s.refreshExpireHours) * time.Hour),
		IPAddress: input.IPAddress,
		UserAgent: input.UserAgent,
	}
	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, err
	}

	accessToken, err := s.jwt.GenerateToken(user.PublicID)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *IamService) Refresh(ctx context.Context, refreshToken string) (*LoginOutput, error) {
	session, err := s.sessionStore.GetByToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if err := s.sessionStore.Delete(ctx, refreshToken); err != nil {
		return nil, errors.New("failed to rotate session")
	}

	newRefreshToken, err := iam.GenerateOpaqueToken()
	if err != nil {
		return nil, err
	}

	newSession := &domain.Session{
		UserID:    session.UserID,
		Token:     newRefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(s.refreshExpireHours) * time.Hour),
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
	}
	if err := s.sessionStore.Create(ctx, newSession); err != nil {
		return nil, err
	}

	accessToken, err := s.jwt.GenerateToken(user.PublicID)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

type LogoutInput struct {
	PublicID       string
	RefreshToken   string
	JTI            string
	TokenExpiresAt time.Time
}

func (s *IamService) Logout(ctx context.Context, input LogoutInput) error {
	user, err := s.userRepo.GetByPublicID(ctx, input.PublicID)
	if err != nil {
		return errors.New("user not found")
	}

	session, err := s.sessionStore.GetByToken(ctx, input.RefreshToken)
	if err != nil {
		return errors.New("invalid session")
	}

	if session.UserID != user.ID {
		return errors.New("session does not belong to user")
	}

	if err := s.sessionStore.Delete(ctx, input.RefreshToken); err != nil {
		return err
	}

	if input.JTI != "" {
		ttl := time.Until(input.TokenExpiresAt)
		if err := s.blocklist.Add(ctx, input.JTI, ttl); err != nil {
			s.log.Warn("blocklist add failed", zap.String("jti", input.JTI), zap.Error(err))
		}
	}

	return nil
}
