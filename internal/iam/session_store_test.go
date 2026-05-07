package iam

import (
	"context"
	"errors"
	"testing"
	"time"

	"saas/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// mockSessionStore — hand-written mock for the SessionStore interface
// ---------------------------------------------------------------------------

type mockSessionStore struct {
	createFn         func(ctx context.Context, s *domain.Session) error
	getByTokenFn     func(ctx context.Context, token string) (*domain.Session, error)
	deleteFn         func(ctx context.Context, token string) error
	deleteByUserIDFn func(ctx context.Context, userID int64) error
}

func (m *mockSessionStore) Create(ctx context.Context, s *domain.Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, s)
	}
	return nil
}

func (m *mockSessionStore) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	if m.getByTokenFn != nil {
		return m.getByTokenFn(ctx, token)
	}
	return nil, errors.New("not found")
}

func (m *mockSessionStore) Delete(ctx context.Context, token string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, token)
	}
	return nil
}

func (m *mockSessionStore) DeleteByUserID(ctx context.Context, userID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testSession() *domain.Session {
	return &domain.Session{
		ID:        1,
		UserID:    42,
		Token:     "test-token-abc",
		ExpiresAt: time.Now().Add(time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test",
	}
}

func noopLogger() *zap.Logger {
	return zap.NewNop()
}

// ---------------------------------------------------------------------------
// HybridSessionStore — Create
// ---------------------------------------------------------------------------

func TestHybridSessionStore_Create_DBSucceedsRedisSucceeds_NoError(t *testing.T) {
	db := &mockSessionStore{
		createFn: func(_ context.Context, _ *domain.Session) error { return nil },
	}
	red := &mockSessionStore{
		createFn: func(_ context.Context, _ *domain.Session) error { return nil },
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	err := store.Create(context.Background(), testSession())
	require.NoError(t, err)
}

func TestHybridSessionStore_Create_DBSucceedsRedisFails_NoError(t *testing.T) {
	db := &mockSessionStore{
		createFn: func(_ context.Context, _ *domain.Session) error { return nil },
	}
	red := &mockSessionStore{
		createFn: func(_ context.Context, _ *domain.Session) error {
			return errors.New("redis unavailable")
		},
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	// Redis failure must be swallowed (logged as warning), not propagated.
	err := store.Create(context.Background(), testSession())
	require.NoError(t, err)
}

func TestHybridSessionStore_Create_DBFails_ReturnsError_RedisNotCalled(t *testing.T) {
	redisCalled := false
	db := &mockSessionStore{
		createFn: func(_ context.Context, _ *domain.Session) error {
			return errors.New("db down")
		},
	}
	red := &mockSessionStore{
		createFn: func(_ context.Context, _ *domain.Session) error {
			redisCalled = true
			return nil
		},
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	err := store.Create(context.Background(), testSession())
	require.Error(t, err)
	assert.False(t, redisCalled, "Redis must not be called when DB write fails")
}

// ---------------------------------------------------------------------------
// HybridSessionStore — GetByToken
// ---------------------------------------------------------------------------

func TestHybridSessionStore_GetByToken_RedisHit_ReturnsSession_DBNotCalled(t *testing.T) {
	sess := testSession()
	dbCalled := false

	red := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return sess, nil
		},
	}
	db := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			dbCalled = true
			return nil, errors.New("should not be called")
		},
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	got, err := store.GetByToken(context.Background(), "test-token-abc")
	require.NoError(t, err)
	assert.Equal(t, sess, got)
	assert.False(t, dbCalled)
}

func TestHybridSessionStore_GetByToken_RedisMiss_DBHit_RepopulatesRedis(t *testing.T) {
	sess := testSession()
	redisRepopulated := false

	red := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return nil, errors.New("cache miss")
		},
		createFn: func(_ context.Context, _ *domain.Session) error {
			redisRepopulated = true
			return nil
		},
	}
	db := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return sess, nil
		},
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	got, err := store.GetByToken(context.Background(), "test-token-abc")
	require.NoError(t, err)
	assert.Equal(t, sess, got)
	assert.True(t, redisRepopulated, "session must be written back to Redis on DB hit")
}

func TestHybridSessionStore_GetByToken_RedisMiss_DBMiss_ReturnsError(t *testing.T) {
	red := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return nil, errors.New("cache miss")
		},
	}
	db := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return nil, errors.New("not found in db")
		},
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	_, err := store.GetByToken(context.Background(), "missing-token")
	require.Error(t, err)
}

func TestHybridSessionStore_GetByToken_RedisMiss_DBHit_RepopulateFails_SessionStillReturned(t *testing.T) {
	sess := testSession()

	red := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return nil, errors.New("cache miss")
		},
		createFn: func(_ context.Context, _ *domain.Session) error {
			return errors.New("redis write failed")
		},
	}
	db := &mockSessionStore{
		getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
			return sess, nil
		},
	}
	store := NewHybridSessionStore(db, red, noopLogger())
	got, err := store.GetByToken(context.Background(), "test-token-abc")
	// Redis repopulate failure is a warning, not a fatal error.
	require.NoError(t, err)
	assert.Equal(t, sess, got)
}

// ---------------------------------------------------------------------------
// HybridSessionStore — Delete
// ---------------------------------------------------------------------------

func TestHybridSessionStore_Delete_RedisFails_DBIsReturnValue(t *testing.T) {
	tests := []struct {
		name      string
		redisErr  error
		dbErr     error
		wantError bool
	}{
		{
			name:      "redis fails, db succeeds — no error returned",
			redisErr:  errors.New("redis down"),
			dbErr:     nil,
			wantError: false,
		},
		{
			name:      "redis succeeds, db fails — error returned",
			redisErr:  nil,
			dbErr:     errors.New("db down"),
			wantError: true,
		},
		{
			name:      "both succeed — no error",
			redisErr:  nil,
			dbErr:     nil,
			wantError: false,
		},
		{
			name:      "both fail — db error returned",
			redisErr:  errors.New("redis down"),
			dbErr:     errors.New("db down"),
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			red := &mockSessionStore{
				deleteFn: func(_ context.Context, _ string) error { return tc.redisErr },
			}
			db := &mockSessionStore{
				deleteFn: func(_ context.Context, _ string) error { return tc.dbErr },
			}
			store := NewHybridSessionStore(db, red, noopLogger())
			err := store.Delete(context.Background(), "some-token")
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HybridSessionStore — DeleteByUserID
// ---------------------------------------------------------------------------

func TestHybridSessionStore_DeleteByUserID_RedisFails_DBIsReturnValue(t *testing.T) {
	tests := []struct {
		name      string
		redisErr  error
		dbErr     error
		wantError bool
	}{
		{
			name:      "redis fails, db succeeds — no error returned",
			redisErr:  errors.New("redis down"),
			dbErr:     nil,
			wantError: false,
		},
		{
			name:      "redis succeeds, db fails — error returned",
			redisErr:  nil,
			dbErr:     errors.New("db down"),
			wantError: true,
		},
		{
			name:      "both succeed — no error",
			redisErr:  nil,
			dbErr:     nil,
			wantError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			red := &mockSessionStore{
				deleteByUserIDFn: func(_ context.Context, _ int64) error { return tc.redisErr },
			}
			db := &mockSessionStore{
				deleteByUserIDFn: func(_ context.Context, _ int64) error { return tc.dbErr },
			}
			store := NewHybridSessionStore(db, red, noopLogger())
			err := store.DeleteByUserID(context.Background(), 42)
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

