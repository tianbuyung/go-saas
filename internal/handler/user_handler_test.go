package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"saas/internal/domain"
	"saas/internal/router"
	"saas/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// mockUserService — implements service.UserServiceIface
// ---------------------------------------------------------------------------

type mockUserService struct {
	getMeFn func(ctx context.Context, publicID string) (*domain.User, error)
}

func (m *mockUserService) GetMe(ctx context.Context, publicID string) (*domain.User, error) {
	if m.getMeFn != nil {
		return m.getMeFn(ctx, publicID)
	}
	return nil, errors.New("not implemented")
}

// ---------------------------------------------------------------------------
// UserHandler.Me
// ---------------------------------------------------------------------------

func TestUserHandler_Me(t *testing.T) {
	validUser := &domain.User{
		ID:       99, // json:"-" — must NOT appear in response
		PublicID: "pub-me-123",
		Email:    "me@example.com",
		Name:     "Me User",
	}

	tests := []struct {
		name           string
		contextVals    map[string]any
		svcUser        *domain.User
		svcErr         error
		wantStatus     int
		wantBodyAbsent string // key that must NOT appear in response
	}{
		{
			name: "valid context — 200, no id in response",
			contextVals: map[string]any{
				router.ContextUserIDKey: "pub-me-123",
			},
			svcUser:        validUser,
			wantStatus:     http.StatusOK,
			wantBodyAbsent: "id",
		},
		{
			name:        "ContextUserIDKey missing — 401",
			contextVals: map[string]any{},
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name: "ContextUserIDKey wrong type (int) — 401",
			contextVals: map[string]any{
				router.ContextUserIDKey: 42, // int, not string
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "service error — 500",
			contextVals: map[string]any{
				router.ContextUserIDKey: "pub-me-123",
			},
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockUserService{
				getMeFn: func(_ context.Context, _ string) (*domain.User, error) {
					return tc.svcUser, tc.svcErr
				},
			}
			h := NewUserHandler(svc)
			c := newMockContext(nil)
			c.contextVals = tc.contextVals
			h.Me(c)

			assert.Equal(t, tc.wantStatus, c.statusCode)

			if tc.wantStatus == http.StatusOK {
				require.NotNil(t, c.respBody)
				if tc.wantBodyAbsent != "" {
					_, present := c.respBody[tc.wantBodyAbsent]
					assert.False(t, present, "field %q must not appear (json:\"-\" not working)", tc.wantBodyAbsent)
				}
			}

			// Compile-time guard: mockUserService satisfies the interface.
			var _ service.UserServiceIface = (*mockUserService)(nil)
		})
	}
}
