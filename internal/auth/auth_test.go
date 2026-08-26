package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/auth"
)

// Behavior 73: with DISCORD_MCP_AUTH_TOKEN set, /mcp without a bearer
// returns 401 with a WWW-Authenticate header.
func TestMiddleware_NoBearer_401WithWWWAuthenticate(t *testing.T) {
	t.Parallel()
	h := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler reached without a bearer token")
	}), "configured-token")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("WWW-Authenticate header is empty, want a challenge")
	}
}

// Behavior 74: with the token set, /mcp with the correct bearer reaches
// the MCP handler.
func TestMiddleware_CorrectBearer_ReachesHandler(t *testing.T) {
	t.Parallel()
	reached := false
	h := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}), "configured-token")

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer configured-token")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !reached {
		t.Error("handler not reached with the correct bearer token")
	}
}

// Behavior 75: a wrong bearer returns 401.
func TestMiddleware_WrongBearer_401(t *testing.T) {
	t.Parallel()
	h := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler reached with an incorrect bearer token")
	}), "configured-token")

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
