//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseURL() string {
	if u := os.Getenv("E2E_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:3002"
}

// post is a thin helper for JSON POST requests.
func post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(baseURL()+path, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	return resp
}

// authedGet performs a GET with a Bearer token.
func authedGet(t *testing.T, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL()+path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// authedPost performs an authenticated POST (Bearer token + JSON body).
func authedPost(t *testing.T, path, token, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL()+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// decodeBody decodes the response body into a map and closes the body.
func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

// uniqueEmail returns a unique email address for each test run.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@e2e-test.example.com", prefix, time.Now().UnixNano())
}

// ---------------------------------------------------------------------------
// Full happy path: register → login → me → refresh → replay → logout → me
// ---------------------------------------------------------------------------

func TestAPI_Auth_FullHappyPath(t *testing.T) {
	email := uniqueEmail("happy")
	password := "SecurePass123!"
	creds := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)

	// 1. Register
	resp := post(t, "/auth/register", creds)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "register must return 201")
	resp.Body.Close()

	// 2. Login
	resp = post(t, "/auth/login", creds)
	require.Equal(t, http.StatusOK, resp.StatusCode, "login must return 200")
	loginBody := decodeBody(t, resp)
	accessToken, ok := loginBody["access_token"].(string)
	require.True(t, ok && accessToken != "", "access_token must be non-empty")
	refreshToken, ok := loginBody["refresh_token"].(string)
	require.True(t, ok && refreshToken != "", "refresh_token must be non-empty")

	// 3. GET /api/me with access token
	resp = authedGet(t, "/api/me", accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "/api/me must return 200 with valid token")
	meBody := decodeBody(t, resp)
	_, hasID := meBody["id"]
	assert.False(t, hasID, "response must not expose internal id (json:\"-\")")

	// 4. Refresh — new tokens must differ from old ones
	refreshBody := fmt.Sprintf(`{"refresh_token":%q}`, refreshToken)
	resp = post(t, "/auth/refresh", refreshBody)
	require.Equal(t, http.StatusOK, resp.StatusCode, "refresh must return 200")
	refreshRespBody := decodeBody(t, resp)
	newAccessToken, ok := refreshRespBody["access_token"].(string)
	require.True(t, ok && newAccessToken != "")
	newRefreshToken, ok := refreshRespBody["refresh_token"].(string)
	require.True(t, ok && newRefreshToken != "")
	assert.NotEqual(t, accessToken, newAccessToken, "new access token must differ")
	assert.NotEqual(t, refreshToken, newRefreshToken, "new refresh token must differ (rolling rotation)")

	// 5. Replay the old (rotated-out) refresh token — must be rejected
	resp = post(t, "/auth/refresh", refreshBody)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"replaying a rotated-out refresh token must return 401")
	resp.Body.Close()

	// 6. Logout with the new tokens
	logoutBody := fmt.Sprintf(`{"refresh_token":%q}`, newRefreshToken)
	resp = authedPost(t, "/auth/logout", newAccessToken, logoutBody)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "logout must return 200")
	resp.Body.Close()

	// 7. After logout, /api/me with same access token:
	// If Redis blocklist is configured the token is immediately revoked (401).
	// If only NoopBlocklist is configured the JWT is still technically valid
	// until expiry (200). Both outcomes are acceptable here; we just verify
	// the server does not panic (no 5xx).
	resp = authedGet(t, "/api/me", newAccessToken)
	assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
		"server must not 500 after logout")
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Register edge cases
// ---------------------------------------------------------------------------

func TestAPI_Register_DuplicateEmail_Returns409(t *testing.T) {
	email := uniqueEmail("dup")
	creds := fmt.Sprintf(`{"email":%q,"password":"pass123"}`, email)

	resp := post(t, "/auth/register", creds)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = post(t, "/auth/register", creds)
	assert.Equal(t, http.StatusConflict, resp.StatusCode,
		"duplicate registration must return 409")
	resp.Body.Close()
}

func TestAPI_Register_EmptyEmail_Returns400(t *testing.T) {
	resp := post(t, "/auth/register", `{"email":"","password":"pass123"}`)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestAPI_Register_EmptyPassword_Returns400(t *testing.T) {
	resp := post(t, "/auth/register", fmt.Sprintf(`{"email":%q,"password":""}`, uniqueEmail("nopw")))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Login edge cases
// ---------------------------------------------------------------------------

func TestAPI_Login_WrongPassword_Returns401(t *testing.T) {
	email := uniqueEmail("wrongpw")
	resp := post(t, "/auth/register", fmt.Sprintf(`{"email":%q,"password":"correct"}`, email))
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = post(t, "/auth/login", fmt.Sprintf(`{"email":%q,"password":"wrong"}`, email))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Protected endpoint auth boundary
// ---------------------------------------------------------------------------

func TestAPI_Me_NoToken_Returns401(t *testing.T) {
	resp, err := http.Get(baseURL() + "/api/me")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestAPI_Me_MalformedAuthHeader_Returns401(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, baseURL()+"/api/me", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Token abc") // wrong scheme
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
