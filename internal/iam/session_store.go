package iam

import (
	"context"

	"saas/internal/domain"
)

type SessionStore interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByToken(ctx context.Context, token string) (*domain.Session, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID int64) error
}
