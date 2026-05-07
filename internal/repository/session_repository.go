package repository

import (
	"context"

	"saas/internal/db"
	"saas/internal/domain"
	"saas/internal/iam"

	"github.com/jackc/pgx/v5/pgtype"
)

type DBSessionStore struct {
	q *db.Queries
}

func NewDBSessionStore(q *db.Queries) iam.SessionStore {
	return &DBSessionStore{q: q}
}

func (s *DBSessionStore) Create(ctx context.Context, session *domain.Session) error {
	_, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		UserID:    session.UserID,
		Token:     session.Token,
		ExpiresAt: pgtype.Timestamptz{Time: session.ExpiresAt, Valid: true},
		IpAddress: pgtype.Text{String: session.IPAddress, Valid: session.IPAddress != ""},
		UserAgent: pgtype.Text{String: session.UserAgent, Valid: session.UserAgent != ""},
	})
	return err
}

func (s *DBSessionStore) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	row, err := s.q.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return &domain.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		IPAddress: row.IpAddress.String,
		UserAgent: row.UserAgent.String,
	}, nil
}

func (s *DBSessionStore) Delete(ctx context.Context, token string) error {
	return s.q.DeleteSession(ctx, token)
}

func (s *DBSessionStore) DeleteByUserID(ctx context.Context, userID int64) error {
	return s.q.DeleteUserSessions(ctx, userID)
}
