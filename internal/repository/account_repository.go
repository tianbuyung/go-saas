package repository

import (
	"context"

	"saas/internal/db"
	"saas/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateAccountInput struct {
	UserID       int64
	AccountID    string
	ProviderID   string
	PasswordHash string
	Salt         string
}

type UpdateAccountPasswordInput struct {
	UserID       int64
	PasswordHash string
	Salt         string
}

type AccountRepository interface {
	Create(ctx context.Context, input CreateAccountInput) (*domain.Account, error)
	GetByProvider(ctx context.Context, providerID, accountID string) (*domain.Account, error)
	GetCredentialByUserID(ctx context.Context, userID int64) (*domain.Account, error)
	UpdatePassword(ctx context.Context, input UpdateAccountPasswordInput) (*domain.Account, error)
}

type accountRepository struct {
	q *db.Queries
}

func NewAccountRepository(q *db.Queries) AccountRepository {
	return &accountRepository{q: q}
}

func (r *accountRepository) Create(ctx context.Context, input CreateAccountInput) (*domain.Account, error) {
	a, err := r.q.CreateAccount(ctx, db.CreateAccountParams{
		UserID:     input.UserID,
		AccountID:  input.AccountID,
		ProviderID: input.ProviderID,
		Password:   pgtype.Text{String: input.PasswordHash, Valid: input.PasswordHash != ""},
		Salt:       pgtype.Text{String: input.Salt, Valid: input.Salt != ""},
	})
	if err != nil {
		return nil, err
	}

	return toDomainAccount(a), nil
}

func (r *accountRepository) GetByProvider(ctx context.Context, providerID, accountID string) (*domain.Account, error) {
	a, err := r.q.GetAccountByProvider(ctx, db.GetAccountByProviderParams{
		ProviderID: providerID,
		AccountID:  accountID,
	})
	if err != nil {
		return nil, err
	}

	return toDomainAccount(a), nil
}

func (r *accountRepository) GetCredentialByUserID(ctx context.Context, userID int64) (*domain.Account, error) {
	a, err := r.q.GetCredentialAccountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toDomainAccount(a), nil
}

func (r *accountRepository) UpdatePassword(ctx context.Context, input UpdateAccountPasswordInput) (*domain.Account, error) {
	a, err := r.q.UpdateAccountPassword(ctx, db.UpdateAccountPasswordParams{
		UserID:   input.UserID,
		Password: pgtype.Text{String: input.PasswordHash, Valid: true},
		Salt:     pgtype.Text{String: input.Salt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return toDomainAccount(a), nil
}

func toDomainAccount(a db.Account) *domain.Account {
	return &domain.Account{
		ID:         a.ID,
		UserID:     a.UserID,
		AccountID:  a.AccountID,
		ProviderID: a.ProviderID,
		Password:   a.Password.String,
		Salt:       a.Salt.String,
	}
}
