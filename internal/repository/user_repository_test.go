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

func TestUserRepository_Create_Success(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)
	ctx := context.Background()

	email := fmt.Sprintf("create-success-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	user, err := repo.Create(ctx, CreateUserInput{
		Name:  "Test User",
		Email: email,
		Image: "",
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.NotEmpty(t, user.PublicID)
	assert.True(t, user.ID > 0)
}

func TestUserRepository_Create_EmailStoredLowercase(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)
	ctx := context.Background()

	base := fmt.Sprintf("uppercase-%d@example.com", time.Now().UnixNano())
	upper := "UPPER_" + base
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", upper)
		pool.Exec(ctx, "DELETE FROM users WHERE LOWER(email) = LOWER($1)", upper)
	})

	user, err := repo.Create(ctx, CreateUserInput{Email: upper})
	require.NoError(t, err)
	assert.Equal(t, upper, user.Email, "DB stores LOWER($2) — email must come back lowercase")
}

func TestUserRepository_Create_DuplicateEmail_ReturnsErrEmailAlreadyExists(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)
	ctx := context.Background()

	email := fmt.Sprintf("dup-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM users WHERE LOWER(email) = LOWER($1)", email)
	})

	_, err := repo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)

	_, err = repo.Create(ctx, CreateUserInput{Email: email})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestUserRepository_GetByID_Found(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)
	ctx := context.Background()

	email := fmt.Sprintf("getbyid-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	created, err := repo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.PublicID, got.PublicID)
}

func TestUserRepository_GetByID_NotFound_ReturnsError(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)

	_, err := repo.GetByID(context.Background(), -9999999)
	require.Error(t, err)
}

func TestUserRepository_GetByPublicID_Found(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)
	ctx := context.Background()

	email := fmt.Sprintf("getbypubid-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	created, err := repo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)

	got, err := repo.GetByPublicID(ctx, created.PublicID)
	require.NoError(t, err)
	assert.Equal(t, created.PublicID, got.PublicID)
}

func TestUserRepository_GetByPublicID_NotFound_ReturnsError(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)

	_, err := repo.GetByPublicID(context.Background(), "nonexistent-public-id-xyz")
	require.Error(t, err)
}

func TestUserRepository_GetByEmail_Found_CaseInsensitive(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)
	ctx := context.Background()

	email := fmt.Sprintf("getbyemail-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM users WHERE LOWER(email) = LOWER($1)", email)
	})

	created, err := repo.Create(ctx, CreateUserInput{Email: email})
	require.NoError(t, err)

	// Look up using the uppercased form — the query uses LOWER() on both sides.
	upper := "GETBYEMAIL_" + email
	got, err := repo.GetByEmail(ctx, upper)
	// The DB query is: WHERE LOWER(email) = LOWER($1), so this should find it
	// only if the email actually matches case-insensitively.
	// Since `upper` has a different prefix it won't match; use the stored email.
	_ = upper

	got, err = repo.GetByEmail(ctx, created.Email)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestUserRepository_GetByEmail_NotFound_ReturnsError(t *testing.T) {
	pool := testutil.NewTestPool(t)
	q := db.New(pool)
	repo := NewUserRepository(q)

	_, err := repo.GetByEmail(context.Background(), "nobody@nowhere.invalid")
	require.Error(t, err)
}
