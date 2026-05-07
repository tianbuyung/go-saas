//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"saas/internal/db"
	"saas/internal/domain"
	"saas/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedUserForSession creates a minimal user row and returns its internal ID.
func seedUserForSession(t *testing.T, q *db.Queries, ctx context.Context) int64 {
	t.Helper()
	email := fmt.Sprintf("sess-user-%d@example.com", time.Now().UnixNano())
	userRepo := NewUserRepository(q)
	user, err := userRepo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)
	t.Cleanup(func() {
		// sessions are cleaned up individually per test; clean user last
		q.DeleteUserSessions(ctx, user.ID) //nolint:errcheck
		// pool not available here; rely on per-test cleanup for user rows
	})
	return user.ID
}

func TestSessionRepository_Create_Success(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID := seedUserForSession(t, q, ctx)
	token := fmt.Sprintf("tok-create-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM sessions WHERE token = $1", token)
		pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	})

	store := NewDBSessionStore(q)
	sess := &domain.Session{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "Go-Test/1.0",
	}
	err := store.Create(ctx, sess)
	require.NoError(t, err)
}

func TestSessionRepository_GetByToken_Found(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID := seedUserForSession(t, q, ctx)
	token := fmt.Sprintf("tok-get-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM sessions WHERE token = $1", token)
		pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	})

	store := NewDBSessionStore(q)
	sess := &domain.Session{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, store.Create(ctx, sess))

	got, err := store.GetByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, token, got.Token)
	assert.Equal(t, userID, got.UserID)
}

func TestSessionRepository_GetByToken_NotFound_ReturnsError(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	store := NewDBSessionStore(q)

	_, err := store.GetByToken(context.Background(), "nonexistent-token-xyz")
	require.Error(t, err)
}

func TestSessionRepository_Delete_RemovesSession(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID := seedUserForSession(t, q, ctx)
	token := fmt.Sprintf("tok-del-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		// already deleted by the test; this is a safety net
		pool.Exec(ctx, "DELETE FROM sessions WHERE token = $1", token)
		pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	})

	store := NewDBSessionStore(q)
	sess := &domain.Session{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, store.Create(ctx, sess))

	require.NoError(t, store.Delete(ctx, token))

	_, err := store.GetByToken(ctx, token)
	require.Error(t, err, "session must not be retrievable after deletion")
}

func TestSessionRepository_DeleteByUserID_RemovesAllUserSessions(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID := seedUserForSession(t, q, ctx)
	tokens := []string{
		fmt.Sprintf("tok-uid-a-%d", time.Now().UnixNano()),
		fmt.Sprintf("tok-uid-b-%d", time.Now().UnixNano()),
	}

	t.Cleanup(func() {
		for _, tok := range tokens {
			pool.Exec(ctx, "DELETE FROM sessions WHERE token = $1", tok)
		}
		pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	})

	store := NewDBSessionStore(q)
	for _, tok := range tokens {
		sess := &domain.Session{
			UserID:    userID,
			Token:     tok,
			ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, store.Create(ctx, sess))
	}

	require.NoError(t, store.DeleteByUserID(ctx, userID))

	for _, tok := range tokens {
		_, err := store.GetByToken(ctx, tok)
		require.Error(t, err, "session %q must be gone after DeleteByUserID", tok)
	}
}
