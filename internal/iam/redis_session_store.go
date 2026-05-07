package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"saas/internal/domain"
	"saas/pkg/cache"

	"go.uber.org/zap"
)

type RedisSessionStore struct {
	r   cache.Cacher
	log *zap.Logger
}

func NewRedisSessionStore(r cache.Cacher, log *zap.Logger) SessionStore {
	return &RedisSessionStore{r: r, log: log}
}

func (s *RedisSessionStore) sessionKey(token string) string {
	return "session:" + token
}

func (s *RedisSessionStore) userKey(userID int64) string {
	return fmt.Sprintf("user_sessions:%d", userID)
}

func (s *RedisSessionStore) Create(ctx context.Context, session *domain.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return errors.New("session already expired")
	}
	if err := s.r.Set(ctx, s.sessionKey(session.Token), string(data), ttl); err != nil {
		return err
	}
	return s.r.SAdd(ctx, s.userKey(session.UserID), session.Token)
}

func (s *RedisSessionStore) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	data, err := s.r.Get(ctx, s.sessionKey(token))
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	var session domain.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	if session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("session expired")
	}
	return &session, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, token string) error {
	session, err := s.GetByToken(ctx, token)
	if err == nil {
		if err := s.r.SRem(ctx, s.userKey(session.UserID), token); err != nil {
			s.log.Warn("redis SRem failed on session delete", zap.Error(err))
		}
	}
	return s.r.Del(ctx, s.sessionKey(token))
}

func (s *RedisSessionStore) DeleteByUserID(ctx context.Context, userID int64) error {
	userKey := s.userKey(userID)
	tokens, err := s.r.SMembers(ctx, userKey)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(tokens)+1)
	for _, t := range tokens {
		keys = append(keys, s.sessionKey(t))
	}
	keys = append(keys, userKey)
	return s.r.Del(ctx, keys...)
}
