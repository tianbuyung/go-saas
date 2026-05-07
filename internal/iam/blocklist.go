package iam

import (
	"context"
	"errors"
	"time"

	"saas/pkg/cache"
)

type Blocklist interface {
	Add(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// NoopBlocklist is a fail-open no-op used when Redis is not configured.
type NoopBlocklist struct{}

func NewNoopBlocklist() Blocklist          { return &NoopBlocklist{} }
func (*NoopBlocklist) Add(_ context.Context, _ string, _ time.Duration) error { return nil }
func (*NoopBlocklist) IsRevoked(_ context.Context, _ string) (bool, error)    { return false, nil }

// RedisBlocklist stores revoked JTIs in Redis until their natural expiry.
type RedisBlocklist struct {
	r cache.Cacher
}

func NewRedisBlocklist(r cache.Cacher) Blocklist {
	return &RedisBlocklist{r: r}
}

func (b *RedisBlocklist) key(jti string) string {
	return "blocklist:" + jti
}

func (b *RedisBlocklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return b.r.Set(ctx, b.key(jti), "1", ttl)
}

func (b *RedisBlocklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	_, err := b.r.Get(ctx, b.key(jti))
	if errors.Is(err, cache.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
