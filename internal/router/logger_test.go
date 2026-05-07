package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newObservedLogger returns a zap.Logger that writes to an in-memory observer
// and the observer core so tests can inspect emitted log entries.
func newObservedLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return zap.New(core), logs
}

// newTestGinContext builds a *gin.Context suitable for unit tests.
// It uses gin.CreateTestContext so Gin's internal state is properly initialised,
// then attaches a minimal synthetic HTTP request.
func newTestGinContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, nil)
	gc.Request = req
	return gc, w
}

// invokeLogger runs ZapLogger end-to-end:
//  1. Creates the middleware with the provided logger.
//  2. Wraps a no-op next handler so the middleware can call next(c) and then
//     proceed to the logging block.
//  3. Passes the ginContext built around gc.
func invokeLogger(log *zap.Logger, gc *gin.Context) {
	ctx := &ginContext{ctx: gc}
	next := HandlerFunc(func(_ Context) {}) // no-op handler
	ZapLogger(log)(next)(ctx)
}

// fieldNames extracts the set of field names from a log entry for easy assertion.
func fieldNames(entry observer.LoggedEntry) map[string]struct{} {
	names := make(map[string]struct{}, len(entry.Context))
	for _, f := range entry.Context {
		names[f.Key] = struct{}{}
	}
	return names
}

// fieldStringValue returns the string value of a named field, or ("", false).
func fieldStringValue(entry observer.LoggedEntry, key string) (string, bool) {
	for _, f := range entry.Context {
		if f.Key == key {
			return f.String, true
		}
	}
	return "", false
}

// TestZapLogger_PublicRoute_NoPublicIDField verifies that when ContextUserIDKey
// is absent from the gin context (a public/unauthenticated route) the emitted
// log entry does not contain a "public_id" field.
func TestZapLogger_PublicRoute_NoPublicIDField(t *testing.T) {
	log, logs := newObservedLogger()
	gc, _ := newTestGinContext(http.MethodGet, "/health")

	// No key set — simulates an unauthenticated route.
	invokeLogger(log, gc)

	require.Equal(t, 1, logs.Len(), "expected exactly one log entry")
	entry := logs.All()[0]
	assert.Equal(t, "request", entry.Message)

	names := fieldNames(entry)
	assert.Contains(t, names, "method", "standard fields must be present")
	assert.Contains(t, names, "path")
	assert.Contains(t, names, "status")
	assert.Contains(t, names, "latency")
	assert.Contains(t, names, "ip")

	_, hasPubID := names["public_id"]
	assert.False(t, hasPubID, "public_id must be absent on a public route")
}

// TestZapLogger_ProtectedRoute_PublicIDFieldPresent verifies that when
// ContextUserIDKey is set to a non-empty string (injected by JWT middleware on
// a protected route) the emitted log entry contains a "public_id" field with
// the correct value.
func TestZapLogger_ProtectedRoute_PublicIDFieldPresent(t *testing.T) {
	const wantPublicID = "usr_abc123"

	log, logs := newObservedLogger()
	gc, _ := newTestGinContext(http.MethodGet, "/api/me")
	gc.Set(ContextUserIDKey, wantPublicID)

	invokeLogger(log, gc)

	require.Equal(t, 1, logs.Len(), "expected exactly one log entry")
	entry := logs.All()[0]

	gotPublicID, found := fieldStringValue(entry, "public_id")
	require.True(t, found, "public_id field must be present for an authenticated route")
	assert.Equal(t, wantPublicID, gotPublicID)
}

// TestZapLogger_EmptyPublicID_FieldOmitted verifies the edge case where the
// ContextUserIDKey key exists in the gin context but its value is an empty
// string. The log entry must not include a "public_id" field (the empty-string
// guard in the implementation must fire).
func TestZapLogger_EmptyPublicID_FieldOmitted(t *testing.T) {
	log, logs := newObservedLogger()
	gc, _ := newTestGinContext(http.MethodPost, "/api/resource")
	gc.Set(ContextUserIDKey, "") // key present, value empty

	invokeLogger(log, gc)

	require.Equal(t, 1, logs.Len(), "expected exactly one log entry")
	entry := logs.All()[0]

	names := fieldNames(entry)
	_, hasPubID := names["public_id"]
	assert.False(t, hasPubID, "public_id must be omitted when the value is an empty string")
}

// TestZapLogger_WrongTypePublicID_NoPanic verifies the edge case where
// ContextUserIDKey is set to a non-string value. The logger must not panic and
// must emit the log entry without a "public_id" field.
func TestZapLogger_WrongTypePublicID_NoPanic(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "integer value", value: 42},
		{name: "bool value", value: true},
		{name: "nil value", value: nil},
		{name: "struct value", value: struct{ ID int }{ID: 1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log, logs := newObservedLogger()
			gc, _ := newTestGinContext(http.MethodGet, "/api/resource")
			gc.Set(ContextUserIDKey, tc.value)

			// The call must not panic.
			require.NotPanics(t, func() { invokeLogger(log, gc) })

			require.Equal(t, 1, logs.Len(), "expected exactly one log entry")
			entry := logs.All()[0]

			names := fieldNames(entry)
			_, hasPubID := names["public_id"]
			assert.False(t, hasPubID, "public_id must be absent when value is wrong type (%T)", tc.value)
		})
	}
}

// stubContext is a minimal Context implementation that is NOT a *ginContext.
// It is used to exercise the type-assertion guard in ZapLogger.
type stubContext struct{}

func (s *stubContext) JSON(_ int, _ any)         {}
func (s *stubContext) Param(_ string) string     { return "" }
func (s *stubContext) Query(_ string) string     { return "" }
func (s *stubContext) BindJSON(_ any) error      { return nil }
func (s *stubContext) Set(_ string, _ any)       {}
func (s *stubContext) Get(_ string) (any, bool)  { return nil, false }
func (s *stubContext) GetHeader(_ string) string { return "" }
func (s *stubContext) Context() context.Context  { return context.Background() }

// TestZapLogger_NonGinContext_EmitsWarn verifies that passing a context that is
// not a *ginContext causes the middleware to emit a Warn-level log (not a panic
// and not a request log).
func TestZapLogger_NonGinContext_EmitsWarn(t *testing.T) {
	// Capture only Warn-and-above to assert on the warning itself.
	warnCore, warnLogs := observer.New(zapcore.WarnLevel)
	// Capture only Info-and-above, then filter down to exactly InfoLevel entries
	// to confirm no "request" info log was emitted.
	allCore, allLogs := observer.New(zapcore.InfoLevel)
	log := zap.New(zapcore.NewTee(warnCore, allCore))

	next := HandlerFunc(func(_ Context) {})
	// Must not panic.
	require.NotPanics(t, func() { ZapLogger(log)(next)(&stubContext{}) })

	// Exactly one Warn entry must be emitted.
	require.Equal(t, 1, warnLogs.Len(), "expected one Warn entry for unexpected context type")
	assert.Equal(t, zapcore.WarnLevel, warnLogs.All()[0].Level)

	// No Info-level "request" entry must be emitted. allLogs will contain the
	// Warn entry too (Warn >= Info), so filter down to entries whose level is
	// exactly Info.
	var infoEntries []observer.LoggedEntry
	for _, e := range allLogs.All() {
		if e.Level == zapcore.InfoLevel {
			infoEntries = append(infoEntries, e)
		}
	}
	assert.Empty(t, infoEntries, "no Info/request log should be emitted for non-gin context")
}

// TestZapLogger_StandardFields_AlwaysPresent verifies that the five standard
// fields (method, path, status, latency, ip) are always present regardless of
// authentication state.
func TestZapLogger_StandardFields_AlwaysPresent(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		setKey bool
		pubID  string
	}{
		{name: "GET public route", method: http.MethodGet, path: "/ping"},
		{name: "POST authenticated", method: http.MethodPost, path: "/api/data", setKey: true, pubID: "u-99"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log, logs := newObservedLogger()
			gc, _ := newTestGinContext(tc.method, tc.path)
			if tc.setKey {
				gc.Set(ContextUserIDKey, tc.pubID)
			}

			invokeLogger(log, gc)

			require.Equal(t, 1, logs.Len())
			entry := logs.All()[0]
			names := fieldNames(entry)

			for _, required := range []string{"method", "path", "status", "latency", "ip"} {
				assert.Contains(t, names, required, "standard field %q must always be present", required)
			}

			methodVal, _ := fieldStringValue(entry, "method")
			assert.Equal(t, tc.method, methodVal, "method field must match request method")

			pathVal, _ := fieldStringValue(entry, "path")
			assert.Equal(t, tc.path, pathVal, "path field must match request path")
		})
	}
}
