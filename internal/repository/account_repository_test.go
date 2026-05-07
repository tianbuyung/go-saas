//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"saas/internal/db"
	"saas/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestUser is a helper that inserts a user row and registers cleanup.
func createTestUser(t *testing.T, pool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (interface{ RowsAffected() int64 }, error)
}, repo UserRepository, ctx context.Context, email string) int64 {
	t.Helper()
	user, err := repo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)
	return user.ID
}

// seedUser creates a user and registers a cleanup that removes user + accounts.
func seedUser(t *testing.T, q *db.Queries, ctx context.Context) (userID int64, email string) {
	t.Helper()
	email = fmt.Sprintf("acct-test-%d@example.com", time.Now().UnixNano())
	userRepo := NewUserRepository(q)
	user, err := userRepo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)
	return user.ID, email
}

func TestAccountRepository_Create_Success(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID, email := seedUser(t, q, ctx)
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM accounts WHERE user_id = $1", userID)
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	repo := NewAccountRepository(q)
	acc, err := repo.Create(ctx, CreateAccountInput{
		UserID:       userID,
		AccountID:    email,
		ProviderID:   "credential",
		PasswordHash: "hashvalue",
		Salt:         "saltvalue",
	})
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.Equal(t, userID, acc.UserID)
	assert.Equal(t, "credential", acc.ProviderID)
	assert.Equal(t, email, acc.AccountID)
}

func TestAccountRepository_GetByProvider_Found(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID, email := seedUser(t, q, ctx)
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM accounts WHERE user_id = $1", userID)
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	repo := NewAccountRepository(q)
	_, err := repo.Create(ctx, CreateAccountInput{
		UserID:       userID,
		AccountID:    email,
		ProviderID:   "credential",
		PasswordHash: "hash",
		Salt:         "salt",
	})
	require.NoError(t, err)

	got, err := repo.GetByProvider(ctx, "credential", email)
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, email, got.AccountID)
}

func TestAccountRepository_GetByProvider_NotFound_ReturnsError(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewAccountRepository(q)

	_, err := repo.GetByProvider(context.Background(), "credential", "nobody@nowhere.invalid")
	require.Error(t, err)
}

func TestAccountRepository_GetCredentialByUserID_Found(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID, email := seedUser(t, q, ctx)
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM accounts WHERE user_id = $1", userID)
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	repo := NewAccountRepository(q)
	_, err := repo.Create(ctx, CreateAccountInput{
		UserID:       userID,
		AccountID:    email,
		ProviderID:   "credential",
		PasswordHash: "hash",
		Salt:         "salt",
	})
	require.NoError(t, err)

	got, err := repo.GetCredentialByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, "credential", got.ProviderID)
}

func TestAccountRepository_GetCredentialByUserID_NotFound_ReturnsError(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewAccountRepository(q)

	_, err := repo.GetCredentialByUserID(context.Background(), -9999999)
	require.Error(t, err)
}

func TestAccountRepository_UpdatePassword_Success(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	ctx := context.Background()

	userID, email := seedUser(t, q, ctx)
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM accounts WHERE user_id = $1", userID)
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	repo := NewAccountRepository(q)
	_, err := repo.Create(ctx, CreateAccountInput{
		UserID:       userID,
		AccountID:    email,
		ProviderID:   "credential",
		PasswordHash: "old-hash",
		Salt:         "old-salt",
	})
	require.NoError(t, err)

	updated, err := repo.UpdatePassword(ctx, UpdateAccountPasswordInput{
		UserID:       userID,
		PasswordHash: "new-hash",
		Salt:         "new-salt",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-hash", updated.Password)
	assert.Equal(t, "new-salt", updated.Salt)
}
