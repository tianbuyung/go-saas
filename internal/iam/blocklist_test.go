package iam

import (
	"context"
	"errors"
	"testing"
	"time"

	"saas/pkg/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NoopBlocklist
// ---------------------------------------------------------------------------

func TestNoopBlocklist_Add_AlwaysReturnsNil(t *testing.T) {
	tests := []struct {
		name string
		jti  string
		ttl  time.Duration
	}{
		{"positive ttl", "jti-1", time.Minute},
		{"zero ttl", "jti-2", 0},
		{"negative ttl", "jti-3", -time.Second},
	}

	bl := NewNoopBlocklist()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := bl.Add(context.Background(), tc.jti, tc.ttl)
			require.NoError(t, err)
		})
	}
}

func TestNoopBlocklist_IsRevoked_AlwaysReturnsFalseNil(t *testing.T) {
	bl := NewNoopBlocklist()
	revoked, err := bl.IsRevoked(context.Background(), "any-jti")
	require.NoError(t, err)
	assert.False(t, revoked)
}

// ---------------------------------------------------------------------------
// RedisBlocklist
// ---------------------------------------------------------------------------

type mockCacher struct {
	setFn     func(ctx context.Context, key, value string, ttl time.Duration) error
	getFn     func(ctx context.Context, key string) (string, error)
	delFn     func(ctx context.Context, keys ...string) error
	sAddFn    func(ctx context.Context, key string, members ...string) error
	sMembersFn func(ctx context.Context, key string) ([]string, error)
	sRemFn    func(ctx context.Context, key string, members ...string) error
}

func (m *mockCacher) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockCacher) Get(ctx context.Context, key string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return "", nil
}

func (m *mockCacher) Del(ctx context.Context, keys ...string) error {
	if m.delFn != nil {
		return m.delFn(ctx, keys...)
	}
	return nil
}

func (m *mockCacher) SAdd(ctx context.Context, key string, members ...string) error {
	if m.sAddFn != nil {
		return m.sAddFn(ctx, key, members...)
	}
	return nil
}

func (m *mockCacher) SMembers(ctx context.Context, key string) ([]string, error) {
	if m.sMembersFn != nil {
		return m.sMembersFn(ctx, key)
	}
	return nil, nil
}

func (m *mockCacher) SRem(ctx context.Context, key string, members ...string) error {
	if m.sRemFn != nil {
		return m.sRemFn(ctx, key, members...)
	}
	return nil
}

func TestRedisBlocklist_Add_ZeroOrNegativeTTL_NoOp(t *testing.T) {
	setCalled := false
	c := &mockCacher{
		setFn: func(_ context.Context, _, _ string, _ time.Duration) error {
			setCalled = true
			return nil
		},
	}
	bl := NewRedisBlocklist(c)
	require.NoError(t, bl.Add(context.Background(), "jti-1", 0))
	require.NoError(t, bl.Add(context.Background(), "jti-2", -time.Second))
	assert.False(t, setCalled, "Set must not be called for zero or negative TTL")
}

func TestRedisBlocklist_Add_PositiveTTL_CallsSet(t *testing.T) {
	var gotKey string
	c := &mockCacher{
		setFn: func(_ context.Context, key, _ string, _ time.Duration) error {
			gotKey = key
			return nil
		},
	}
	bl := NewRedisBlocklist(c)
	require.NoError(t, bl.Add(context.Background(), "abc123", time.Minute))
	assert.Equal(t, "blocklist:abc123", gotKey)
}

func TestRedisBlocklist_IsRevoked_KeyPresent_ReturnsTrue(t *testing.T) {
	c := &mockCacher{
		getFn: func(_ context.Context, _ string) (string, error) {
			return "1", nil
		},
	}
	bl := NewRedisBlocklist(c)
	revoked, err := bl.IsRevoked(context.Background(), "jti-x")
	require.NoError(t, err)
	assert.True(t, revoked)
}

func TestRedisBlocklist_IsRevoked_KeyMissing_ReturnsFalse(t *testing.T) {
	c := &mockCacher{
		getFn: func(_ context.Context, _ string) (string, error) {
			return "", cache.ErrNotFound
		},
	}
	bl := NewRedisBlocklist(c)
	revoked, err := bl.IsRevoked(context.Background(), "jti-y")
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestRedisBlocklist_IsRevoked_GetError_ReturnsError(t *testing.T) {
	c := &mockCacher{
		getFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("redis down")
		},
	}
	bl := NewRedisBlocklist(c)
	_, err := bl.IsRevoked(context.Background(), "jti-z")
	require.Error(t, err)
}
