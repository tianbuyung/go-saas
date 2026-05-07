package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"saas/internal/db"
	"saas/internal/domain"
	"saas/internal/iam"
	"saas/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// mock: repository.UserRepository
// ---------------------------------------------------------------------------

type mockUserRepository struct {
	getByIDFn       func(ctx context.Context, id int64) (*domain.User, error)
	getByPublicIDFn func(ctx context.Context, publicID string) (*domain.User, error)
	getByEmailFn    func(ctx context.Context, email string) (*domain.User, error)
	createFn        func(ctx context.Context, input repository.CreateUserInput) (*domain.User, error)
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) GetByPublicID(ctx context.Context, publicID string) (*domain.User, error) {
	if m.getByPublicIDFn != nil {
		return m.getByPublicIDFn(ctx, publicID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) Create(ctx context.Context, input repository.CreateUserInput) (*domain.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}

// ---------------------------------------------------------------------------
// mock: repository.AccountRepository
// ---------------------------------------------------------------------------

type mockAccountRepository struct {
	createFn                func(ctx context.Context, input repository.CreateAccountInput) (*domain.Account, error)
	getByProviderFn         func(ctx context.Context, providerID, accountID string) (*domain.Account, error)
	getCredentialByUserIDFn func(ctx context.Context, userID int64) (*domain.Account, error)
	updatePasswordFn        func(ctx context.Context, input repository.UpdateAccountPasswordInput) (*domain.Account, error)
}

func (m *mockAccountRepository) Create(ctx context.Context, input repository.CreateAccountInput) (*domain.Account, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAccountRepository) GetByProvider(ctx context.Context, providerID, accountID string) (*domain.Account, error) {
	if m.getByProviderFn != nil {
		return m.getByProviderFn(ctx, providerID, accountID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAccountRepository) GetCredentialByUserID(ctx context.Context, userID int64) (*domain.Account, error) {
	if m.getCredentialByUserIDFn != nil {
		return m.getCredentialByUserIDFn(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAccountRepository) UpdatePassword(ctx context.Context, input repository.UpdateAccountPasswordInput) (*domain.Account, error) {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}

// ---------------------------------------------------------------------------
// mock: db.TxManager
// ---------------------------------------------------------------------------

type mockTxManager struct {
	withTxFn func(ctx context.Context, fn func(*db.Queries) error) error
}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(*db.Queries) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(nil)
}

// ---------------------------------------------------------------------------
// mock: iam.SessionStore
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
// mock: iam.Blocklist
// ---------------------------------------------------------------------------

type mockBlocklist struct {
	addFn       func(ctx context.Context, jti string, ttl time.Duration) error
	isRevokedFn func(ctx context.Context, jti string) (bool, error)
}

func (m *mockBlocklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	if m.addFn != nil {
		return m.addFn(ctx, jti, ttl)
	}
	return nil
}

func (m *mockBlocklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if m.isRevokedFn != nil {
		return m.isRevokedFn(ctx, jti)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testJWT() *iam.JWT {
	return iam.NewJWT(iam.JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
	})
}

func testUser(id int64, publicID string) *domain.User {
	return &domain.User{
		ID:       id,
		PublicID: publicID,
		Email:    "user@example.com",
		Name:     "Test User",
	}
}

func newService(
	userRepo repository.UserRepository,
	accountRepo repository.AccountRepository,
	tx db.TxManager,
	ss iam.SessionStore,
	bl iam.Blocklist,
) *IamService {
	return NewIamService(userRepo, accountRepo, tx, testJWT(), ss, bl, zap.NewNop(), 24)
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------


func TestIamService_Register_TxControlled(t *testing.T) {
	tests := []struct {
		name      string
		input     RegisterInput
		txErr     error
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "success — email normalised to lowercase, user returned",
			input: RegisterInput{
				Email:    "  USER@EXAMPLE.COM  ",
				Password: "strongpassword",
				Name:     "Test User",
			},
			txErr:   nil,
			wantErr: false,
		},
		{
			name: "duplicate email returns ErrEmailAlreadyExists",
			input: RegisterInput{
				Email:    "dup@example.com",
				Password: "pass",
			},
			txErr:     repository.ErrEmailAlreadyExists,
			wantErr:   true,
			wantErrIs: repository.ErrEmailAlreadyExists,
		},
		{
			name: "other tx error propagated",
			input: RegisterInput{
				Email:    "a@b.com",
				Password: "pass",
			},
			txErr:   errors.New("db down"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expectedUser := testUser(1, "pub-reg-001")

			mockUser := &mockUserRepository{
				createFn: func(_ context.Context, input repository.CreateUserInput) (*domain.User, error) {
					assert.Equal(t, "user@example.com", input.Email)
					return expectedUser, nil
				},
			}
			mockAccount := &mockAccountRepository{
				createFn: func(_ context.Context, _ repository.CreateAccountInput) (*domain.Account, error) {
					return &domain.Account{}, nil
				},
			}

			tx := &mockTxManager{
				withTxFn: func(ctx context.Context, fn func(*db.Queries) error) error {
					if tc.txErr != nil {
						return tc.txErr
					}
					return fn(nil)
				},
			}

			svc := &IamService{
				userRepo:           &mockUserRepository{},
				accountRepo:        &mockAccountRepository{},
				txManager:          tx,
				jwt:                testJWT(),
				sessionStore:       &mockSessionStore{},
				blocklist:          &mockBlocklist{},
				log:                zap.NewNop(),
				refreshExpireHours: 24,
				makeUserRepo:       func(_ *db.Queries) repository.UserRepository { return mockUser },
				makeAccountRepo:    func(_ *db.Queries) repository.AccountRepository { return mockAccount },
			}

			user, err := svc.Register(ctx(t), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrIs != nil {
					assert.ErrorIs(t, err, tc.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, user)
			assert.Equal(t, expectedUser.PublicID, user.PublicID)
		})
	}
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestIamService_Login(t *testing.T) {
	validHash, err := iam.GeneratePassword("correct-password")
	require.NoError(t, err)

	validAccount := &domain.Account{
		ID:         10,
		UserID:     1,
		AccountID:  "user@example.com",
		ProviderID: "credential",
		Password:   validHash.Hash,
		Salt:       validHash.Salt,
	}
	validUser := testUser(1, "pub-123")

	tests := []struct {
		name          string
		input         LoginInput
		accountResult *domain.Account
		accountErr    error
		userResult    *domain.User
		userErr       error
		sessionErr    error
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name:          "success — both tokens non-empty",
			input:         LoginInput{Email: "user@example.com", Password: "correct-password"},
			accountResult: validAccount,
			userResult:    validUser,
			wantErr:       false,
		},
		{
			name:       "account not found — invalid credentials",
			input:      LoginInput{Email: "ghost@example.com", Password: "pass"},
			accountErr: errors.New("no rows"),
			wantErr:    true,
			wantErrMsg: "invalid credentials",
		},
		{
			name:          "wrong password — invalid credentials",
			input:         LoginInput{Email: "user@example.com", Password: "wrong-password"},
			accountResult: validAccount,
			wantErr:       true,
			wantErrMsg:    "invalid credentials",
		},
		{
			name:          "session create fails — returns error",
			input:         LoginInput{Email: "user@example.com", Password: "correct-password"},
			accountResult: validAccount,
			userResult:    validUser,
			sessionErr:    errors.New("session store down"),
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			accountRepo := &mockAccountRepository{
				getByProviderFn: func(_ context.Context, _, _ string) (*domain.Account, error) {
					return tc.accountResult, tc.accountErr
				},
			}
			userRepo := &mockUserRepository{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return tc.userResult, tc.userErr
				},
			}
			ss := &mockSessionStore{
				createFn: func(_ context.Context, _ *domain.Session) error {
					return tc.sessionErr
				},
			}

			svc := newService(userRepo, accountRepo, &mockTxManager{}, ss, &mockBlocklist{})
			out, err := svc.Login(ctx(t), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.NotEmpty(t, out.AccessToken)
			assert.NotEmpty(t, out.RefreshToken)
		})
	}
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestIamService_Refresh(t *testing.T) {
	validUser := testUser(1, "pub-xyz")
	validSession := &domain.Session{
		ID:        5,
		UserID:    1,
		Token:     "old-refresh-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name           string
		refreshToken   string
		sessionResult  *domain.Session
		sessionErr     error
		userResult     *domain.User
		userErr        error
		deleteErr      error
		newSessionErr  error
		wantErr        bool
		wantErrMsg     string
	}{
		{
			name:          "success — old session deleted, new tokens returned",
			refreshToken:  "old-refresh-token",
			sessionResult: validSession,
			userResult:    validUser,
			wantErr:       false,
		},
		{
			name:         "session not found — invalid refresh token",
			refreshToken: "nonexistent-token",
			sessionErr:   errors.New("not found"),
			wantErr:      true,
			wantErrMsg:   "invalid refresh token",
		},
		{
			name:          "session delete fails — failed to rotate session",
			refreshToken:  "old-refresh-token",
			sessionResult: validSession,
			userResult:    validUser,
			deleteErr:     errors.New("delete failed"),
			wantErr:       true,
			wantErrMsg:    "failed to rotate session",
		},
		{
			name:          "new session create fails — returns error",
			refreshToken:  "old-refresh-token",
			sessionResult: validSession,
			userResult:    validUser,
			newSessionErr: errors.New("create failed"),
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getCallCount := 0
			ss := &mockSessionStore{
				getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
					getCallCount++
					return tc.sessionResult, tc.sessionErr
				},
				deleteFn: func(_ context.Context, _ string) error {
					return tc.deleteErr
				},
				createFn: func(_ context.Context, _ *domain.Session) error {
					return tc.newSessionErr
				},
			}
			userRepo := &mockUserRepository{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return tc.userResult, tc.userErr
				},
			}

			svc := newService(userRepo, &mockAccountRepository{}, &mockTxManager{}, ss, &mockBlocklist{})
			out, err := svc.Refresh(ctx(t), tc.refreshToken)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.NotEmpty(t, out.AccessToken)
			assert.NotEmpty(t, out.RefreshToken)
			assert.NotEqual(t, tc.refreshToken, out.RefreshToken,
				"refresh token must be rotated to a new value")
		})
	}
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestIamService_Logout(t *testing.T) {
	validUser := testUser(1, "pub-logout")
	validSession := &domain.Session{
		ID:     7,
		UserID: 1,
		Token:  "valid-refresh-token",
	}

	futureExpiry := time.Now().Add(5 * time.Minute)

	tests := []struct {
		name           string
		input          LogoutInput
		userResult     *domain.User
		userErr        error
		sessionResult  *domain.Session
		sessionErr     error
		deleteErr      error
		blocklistErr   error
		wantErr        bool
	}{
		{
			name: "success — session deleted and JTI blocklisted",
			input: LogoutInput{
				PublicID:       "pub-logout",
				RefreshToken:   "valid-refresh-token",
				JTI:            "some-jti",
				TokenExpiresAt: futureExpiry,
			},
			userResult:    validUser,
			sessionResult: validSession,
			wantErr:       false,
		},
		{
			name: "user not found — returns error",
			input: LogoutInput{
				PublicID:     "nonexistent",
				RefreshToken: "token",
			},
			userErr: errors.New("not found"),
			wantErr: true,
		},
		{
			name: "session belongs to different user — returns error",
			input: LogoutInput{
				PublicID:     "pub-logout",
				RefreshToken: "valid-refresh-token",
			},
			userResult: testUser(999, "pub-logout"), // user.ID=999 != session.UserID=1
			sessionResult: &domain.Session{
				ID:     7,
				UserID: 1, // mismatch
				Token:  "valid-refresh-token",
			},
			wantErr: true,
		},
		{
			name: "blocklist Add fails — warning logged, logout still succeeds",
			input: LogoutInput{
				PublicID:       "pub-logout",
				RefreshToken:   "valid-refresh-token",
				JTI:            "some-jti",
				TokenExpiresAt: futureExpiry,
			},
			userResult:    validUser,
			sessionResult: validSession,
			blocklistErr:  errors.New("redis down"),
			wantErr:       false, // blocklist failure is non-fatal
		},
		{
			name: "session not found — returns error",
			input: LogoutInput{
				PublicID:     "pub-logout",
				RefreshToken: "missing-token",
			},
			userResult: validUser,
			sessionErr: errors.New("not found"),
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &mockUserRepository{
				getByPublicIDFn: func(_ context.Context, _ string) (*domain.User, error) {
					return tc.userResult, tc.userErr
				},
			}
			ss := &mockSessionStore{
				getByTokenFn: func(_ context.Context, _ string) (*domain.Session, error) {
					return tc.sessionResult, tc.sessionErr
				},
				deleteFn: func(_ context.Context, _ string) error {
					return tc.deleteErr
				},
			}
			bl := &mockBlocklist{
				addFn: func(_ context.Context, _ string, _ time.Duration) error {
					return tc.blocklistErr
				},
			}

			svc := newService(userRepo, &mockAccountRepository{}, &mockTxManager{}, ss, bl)
			err := svc.Logout(ctx(t), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
