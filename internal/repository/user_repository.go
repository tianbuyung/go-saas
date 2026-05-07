package repository

import (
	"context"
	"errors"

	"saas/internal/db"
	"saas/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrEmailAlreadyExists = errors.New("email already exists")

type CreateUserInput struct {
	Name  string
	Email string
	Image string
}

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByPublicID(ctx context.Context, publicID string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, input CreateUserInput) (*domain.User, error)
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

	return toDomainUser(u), nil
}

func (r *userRepository) GetByPublicID(ctx context.Context, publicID string) (*domain.User, error) {
	u, err := r.q.GetUserByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	return toDomainUser(u), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return toDomainUser(u), nil
}

func (r *userRepository) Create(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	u, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Name:  input.Name,
		Email: input.Email,
		Image: pgtype.Text{String: input.Image, Valid: input.Image != ""},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	return &domain.User{
		ID:            u.ID,
		PublicID:      u.PublicID,
		Name:          u.Name,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Image:         u.Image.String,
	}, nil
}

func toDomainUser(u db.ActiveUser) *domain.User {
	return &domain.User{
		ID:            u.ID,
		PublicID:      u.PublicID,
		Name:          u.Name,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Image:         u.Image.String,
	}
}
