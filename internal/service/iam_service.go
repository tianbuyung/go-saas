package service

import (
	"context"
	"errors"
	"strings"

	"saas/internal/db"
	"saas/internal/iam"

	"github.com/jackc/pgx/v5/pgtype"
)

type IamService struct {
	q   *db.Queries
	jwt *iam.JWT
}

func NewIamService(q *db.Queries, jwt *iam.JWT) *IamService {
	return &IamService{
		q:   q,
		jwt: jwt,
	}
}

func (s *IamService) Register(ctx context.Context, email, password string) (*db.User, error) {
	hash, err := iam.GeneratePassword(password)
	if err != nil {
		return nil, err
	}

	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email: email,
		Password: pgtype.Text{
			String: hash.Hash,
			Valid:  true,
		},
		Salt: pgtype.Text{
			String: hash.Salt,
			Valid:  true,
		},
		Provider:   pgtype.Text{Valid: false},
		ProviderID: pgtype.Text{Valid: false},
	})

	return &user, err
}

func (s *IamService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if !iam.ComparePassword(password, user.Password.String, user.Salt.String) {
		return "", errors.New("invalid credentials")
	}

	token, err := s.jwt.GenerateToken(user.ID)
	return token, err
}
