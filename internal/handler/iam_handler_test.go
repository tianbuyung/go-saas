package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"saas/internal/domain"
	"saas/internal/router"
	"saas/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// mockContext — implements router.Context for handler unit tests
// ---------------------------------------------------------------------------

type mockContext struct {
	body        []byte
	contextVals map[string]any
	headers     map[string]string
	// captured response
	statusCode int
	respBody   map[string]any
}

func newMockContext(body []byte) *mockContext {
	return &mockContext{
		body:        body,
		contextVals: make(map[string]any),
		headers:     make(map[string]string),
	}
}

func (m *mockContext) JSON(code int, data any) {
	m.statusCode = code
	b, _ := json.Marshal(data)
	_ = json.Unmarshal(b, &m.respBody)
}

func (m *mockContext) BindJSON(v any) error {
	return json.Unmarshal(m.body, v)
}

func (m *mockContext) Get(key string) (any, bool) {
	v, ok := m.contextVals[key]
	return v, ok
}

func (m *mockContext) Set(key string, val any) {
	m.contextVals[key] = val
}

func (m *mockContext) GetHeader(key string) string {
	return m.headers[key]
}

func (m *mockContext) Context() context.Context {
	return context.Background()
}

func (m *mockContext) Param(_ string) string { return "" }
func (m *mockContext) Query(_ string) string { return "" }

// ---------------------------------------------------------------------------
// mockIamService — implements service.IamServiceIface
// ---------------------------------------------------------------------------

type mockIamService struct {
	registerFn func(ctx context.Context, input service.RegisterInput) (*domain.User, error)
	loginFn    func(ctx context.Context, input service.LoginInput) (*service.LoginOutput, error)
	refreshFn  func(ctx context.Context, refreshToken string) (*service.LoginOutput, error)
	logoutFn   func(ctx context.Context, input service.LogoutInput) error
}

func (m *mockIamService) Register(ctx context.Context, input service.RegisterInput) (*domain.User, error) {
	if m.registerFn != nil {
		return m.registerFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockIamService) Login(ctx context.Context, input service.LoginInput) (*service.LoginOutput, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockIamService) Refresh(ctx context.Context, refreshToken string) (*service.LoginOutput, error) {
	if m.refreshFn != nil {
		return m.refreshFn(ctx, refreshToken)
	}
	return nil, errors.New("not implemented")
}

func (m *mockIamService) Logout(ctx context.Context, input service.LogoutInput) error {
	if m.logoutFn != nil {
		return m.logoutFn(ctx, input)
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func errorBody(msg string) []byte {
	return mustJSON(map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// IamHandler.Register
// ---------------------------------------------------------------------------

func TestIamHandler_Register(t *testing.T) {
	successUser := &domain.User{
		ID:       1, // json:"-" — must NOT appear in response
		PublicID: "pub-123",
		Email:    "user@example.com",
		Name:     "Test User",
	}

	tests := []struct {
		name           string
		body           []byte
		svcErr         error
		svcUser        *domain.User
		wantStatus     int
		wantBodyKey    string // key that must be present in response
		wantBodyAbsent string // key that must NOT appear (e.g. "id")
		wantErrMsg     string
	}{
		{
			name:           "valid body — 201, no id field in response",
			body:           mustJSON(map[string]string{"email": "user@example.com", "password": "pass123"}),
			svcUser:        successUser,
			wantStatus:     http.StatusCreated,
			wantBodyKey:    "PublicID",
			wantBodyAbsent: "id",
		},
		{
			name:       "empty email — 400",
			body:       mustJSON(map[string]string{"email": "", "password": "pass123"}),
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "email and password are required",
		},
		{
			name:       "empty password — 400",
			body:       mustJSON(map[string]string{"email": "a@b.com", "password": ""}),
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "email and password are required",
		},
		{
			name:       "service ErrEmailAlreadyExists — 409",
			body:       mustJSON(map[string]string{"email": "dup@example.com", "password": "pass"}),
			svcErr:     service.ErrEmailAlreadyExists,
			wantStatus: http.StatusConflict,
			wantErrMsg: "email already in use",
		},
		{
			name:       "service other error — 500 with generic message",
			body:       mustJSON(map[string]string{"email": "a@b.com", "password": "pass"}),
			svcErr:     errors.New("db exploded"),
			wantStatus: http.StatusInternalServerError,
			wantErrMsg: "registration failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockIamService{
				registerFn: func(_ context.Context, _ service.RegisterInput) (*domain.User, error) {
					return tc.svcUser, tc.svcErr
				},
			}
			h := NewIamHandler(svc)
			c := newMockContext(tc.body)
			h.Register(c)

			assert.Equal(t, tc.wantStatus, c.statusCode)

			if tc.wantErrMsg != "" {
				assert.Equal(t, tc.wantErrMsg, c.respBody["error"])
			}
			if tc.wantBodyAbsent != "" {
				_, present := c.respBody[tc.wantBodyAbsent]
				assert.False(t, present, "field %q must not appear in response (json:\"-\" not working)", tc.wantBodyAbsent)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IamHandler.Login
// ---------------------------------------------------------------------------

func TestIamHandler_Login(t *testing.T) {
	validOutput := &service.LoginOutput{
		AccessToken:  "access-tok",
		RefreshToken: "refresh-tok",
	}

	tests := []struct {
		name       string
		body       []byte
		svcErr     error
		svcOut     *service.LoginOutput
		wantStatus int
		wantErrMsg string
	}{
		{
			name:       "valid — 200 with both tokens",
			body:       mustJSON(map[string]string{"email": "a@b.com", "password": "pass"}),
			svcOut:     validOutput,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty email — 400",
			body:       mustJSON(map[string]string{"email": "", "password": "pass"}),
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "email and password are required",
		},
		{
			name:       "empty password — 400",
			body:       mustJSON(map[string]string{"email": "a@b.com", "password": ""}),
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "email and password are required",
		},
		{
			name:       "service error — 401 with invalid credentials",
			body:       mustJSON(map[string]string{"email": "a@b.com", "password": "wrong"}),
			svcErr:     errors.New("invalid credentials"),
			wantStatus: http.StatusUnauthorized,
			wantErrMsg: "invalid credentials",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockIamService{
				loginFn: func(_ context.Context, _ service.LoginInput) (*service.LoginOutput, error) {
					return tc.svcOut, tc.svcErr
				},
			}
			h := NewIamHandler(svc)
			c := newMockContext(tc.body)
			h.Login(c)

			assert.Equal(t, tc.wantStatus, c.statusCode)

			if tc.wantErrMsg != "" {
				assert.Equal(t, tc.wantErrMsg, c.respBody["error"])
			}
			if tc.wantStatus == http.StatusOK {
				require.NotNil(t, c.respBody)
				assert.NotEmpty(t, c.respBody["access_token"])
				assert.NotEmpty(t, c.respBody["refresh_token"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IamHandler.Refresh
// ---------------------------------------------------------------------------

func TestIamHandler_Refresh(t *testing.T) {
	validOutput := &service.LoginOutput{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
	}

	tests := []struct {
		name       string
		body       []byte
		svcErr     error
		svcOut     *service.LoginOutput
		wantStatus int
		wantErrMsg string
	}{
		{
			name:       "valid — 200 with both tokens",
			body:       mustJSON(map[string]string{"refresh_token": "old-token"}),
			svcOut:     validOutput,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty refresh_token — 400",
			body:       mustJSON(map[string]string{"refresh_token": ""}),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error — 401 with exact message",
			body:       mustJSON(map[string]string{"refresh_token": "bad-token"}),
			svcErr:     errors.New("invalid refresh token"),
			wantStatus: http.StatusUnauthorized,
			wantErrMsg: "invalid or expired refresh token",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockIamService{
				refreshFn: func(_ context.Context, _ string) (*service.LoginOutput, error) {
					return tc.svcOut, tc.svcErr
				},
			}
			h := NewIamHandler(svc)
			c := newMockContext(tc.body)
			h.Refresh(c)

			assert.Equal(t, tc.wantStatus, c.statusCode)

			if tc.wantErrMsg != "" {
				assert.Equal(t, tc.wantErrMsg, c.respBody["error"])
			}
			if tc.wantStatus == http.StatusOK {
				require.NotNil(t, c.respBody)
				assert.NotEmpty(t, c.respBody["access_token"])
				assert.NotEmpty(t, c.respBody["refresh_token"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IamHandler.Logout
// ---------------------------------------------------------------------------

func TestIamHandler_Logout(t *testing.T) {
	validTime := time.Now().Add(5 * time.Minute)

	// baseVals returns a valid context value map for a successful logout.
	baseVals := func() map[string]any {
		return map[string]any{
			router.ContextUserIDKey:         "pub-abc",
			router.ContextJTIKey:            "jti-xyz",
			router.ContextTokenExpiresAtKey: validTime,
		}
	}
	validBody := mustJSON(map[string]string{"refresh_token": "tok"})

	tests := []struct {
		name        string
		body        []byte
		contextVals map[string]any
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "success — 200",
			body:        validBody,
			contextVals: baseVals(),
			wantStatus:  http.StatusOK,
		},
		{
			name: "ContextUserIDKey missing — 401",
			body: validBody,
			contextVals: map[string]any{
				router.ContextJTIKey:            "jti-xyz",
				router.ContextTokenExpiresAtKey: validTime,
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "ContextUserIDKey wrong type (int) — 401",
			body: validBody,
			contextVals: map[string]any{
				router.ContextUserIDKey:         42, // int, not string
				router.ContextJTIKey:            "jti-xyz",
				router.ContextTokenExpiresAtKey: validTime,
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "ContextJTIKey missing — 401",
			body: validBody,
			contextVals: map[string]any{
				router.ContextUserIDKey:         "pub-abc",
				router.ContextTokenExpiresAtKey: validTime,
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "ContextJTIKey wrong type — 401",
			body: validBody,
			contextVals: map[string]any{
				router.ContextUserIDKey:         "pub-abc",
				router.ContextJTIKey:            12345, // int, not string
				router.ContextTokenExpiresAtKey: validTime,
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "ContextTokenExpiresAtKey missing — 401",
			body: validBody,
			contextVals: map[string]any{
				router.ContextUserIDKey: "pub-abc",
				router.ContextJTIKey:    "jti-xyz",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "ContextTokenExpiresAtKey wrong type (string) — 401",
			body: validBody,
			contextVals: map[string]any{
				router.ContextUserIDKey:         "pub-abc",
				router.ContextJTIKey:            "jti-xyz",
				router.ContextTokenExpiresAtKey: "not-a-time", // string, not time.Time
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "empty refresh_token — 400",
			body:        mustJSON(map[string]string{"refresh_token": ""}),
			contextVals: baseVals(),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "service error — 401",
			body:        validBody,
			contextVals: baseVals(),
			svcErr:      errors.New("invalid session"),
			wantStatus:  http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockIamService{
				logoutFn: func(_ context.Context, _ service.LogoutInput) error {
					return tc.svcErr
				},
			}
			h := NewIamHandler(svc)
			c := newMockContext(tc.body)
			c.contextVals = tc.contextVals
			h.Logout(c)

			assert.Equal(t, tc.wantStatus, c.statusCode)
		})
	}
}
