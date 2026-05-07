package iam

import (
	"context"

	"saas/internal/domain"

	"go.uber.org/zap"
)

type HybridSessionStore struct {
	db    SessionStore
	redis SessionStore
	log   *zap.Logger
}

func NewHybridSessionStore(db SessionStore, redis SessionStore, log *zap.Logger) SessionStore {
	return &HybridSessionStore{db: db, redis: redis, log: log}
}

func (s *HybridSessionStore) Create(ctx context.Context, session *domain.Session) error {
	if err := s.db.Create(ctx, session); err != nil {
		return err
	}
	if err := s.redis.Create(ctx, session); err != nil {
		s.log.Warn("hybrid: redis cache write failed on create", zap.Error(err))
	}
	return nil
}

func (s *HybridSessionStore) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	session, err := s.redis.GetByToken(ctx, token)
	if err == nil {
		return session, nil
	}
	session, err = s.db.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.redis.Create(ctx, session); err != nil {
		s.log.Warn("hybrid: redis cache write failed on create", zap.Error(err))
	}
	return session, nil
}

func (s *HybridSessionStore) Delete(ctx context.Context, token string) error {
	if err := s.redis.Delete(ctx, token); err != nil {
		s.log.Warn("hybrid: redis cache delete failed", zap.Error(err))
	}
	return s.db.Delete(ctx, token)
}

func (s *HybridSessionStore) DeleteByUserID(ctx context.Context, userID int64) error {
	if err := s.redis.DeleteByUserID(ctx, userID); err != nil {
		s.log.Warn("hybrid: redis cache delete-by-user failed", zap.Error(err))
	}
	return s.db.DeleteByUserID(ctx, userID)
}
