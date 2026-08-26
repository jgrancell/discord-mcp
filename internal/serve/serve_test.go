package serve_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/resolve"
	"github.com/jgrancell/discord-mcp/internal/serve"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

// Behavior 71: GET /healthz returns 200 ok.
func TestNewMux_Healthz_Returns200OK(t *testing.T) {
	t.Parallel()
	d := discord.New("http://127.0.0.1:0", "test-token")
	mux := serve.NewMux(serve.Config{
		Tools: tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: "1100000000000000000"},
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("GET /healthz body = %q, want %q", rec.Body.String(), "ok")
	}
}
