package repository

import (
	"context"
	"saas/internal/db"
	"saas/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (*domain.UserAuth, error)
	Create(ctx context.Context, email, passwordHash, salt string) (*domain.User, error)
}

type userRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) UserRepository {
	return &userRepository{q: q}
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:    u.ID,
		Email: u.Email,
	}, nil
}

func (r *userRepository) GetUserAuthByEmail(ctx context.Context, email string) (*domain.UserAuth, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &domain.UserAuth{
		ID:       u.ID,
		Email:    u.Email,
		Password: u.Password.String,
		Salt:     u.Salt.String,
	}, nil
}

func (r *userRepository) Create(ctx context.Context, email, passwordHash, salt string) (*domain.User, error) {
	u, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Email: email,
		Password: pgtype.Text{
			String: passwordHash,
			Valid:  true,
		},
		Salt: pgtype.Text{
			String: salt,
			Valid:  true,
		},
		Provider:   pgtype.Text{Valid: false},
		ProviderID: pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:    u.ID,
		Email: u.Email,
	}, nil
}
