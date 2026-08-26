package mine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Behavior 50: a denied channel appears in the run report with its ID and
// name — never skipped silently.
func TestRun_DeniedChannel_AppearsInReportWithIDAndName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const deniedID = "9988776655443322"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(deniedID, "admin-only", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case strings.Contains(r.URL.Path, deniedID) && strings.Contains(r.URL.Path, "/messages"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"403: Forbidden","code":50001}`))
		case strings.Contains(r.URL.Path, deniedID) && strings.Contains(r.URL.Path, "/threads/archived"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"403: Forbidden","code":50001}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "test-token")
	cfg := mine.Config{
		Discord: c, Root: root, GuildID: guildID, GuildName: "OMGCG",
		Now: func() time.Time { return fixedNow },
	}
	report, err := mine.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v (a denied channel must not abort the whole run)", err)
	}

	if len(report.Denied) != 1 {
		t.Fatalf("report.Denied = %+v, want exactly one entry", report.Denied)
	}
	if report.Denied[0].ID != deniedID {
		t.Errorf("report.Denied[0].ID = %q, want %q", report.Denied[0].ID, deniedID)
	}
	if report.Denied[0].Name != "admin-only" {
		t.Errorf("report.Denied[0].Name = %q, want %q", report.Denied[0].Name, "admin-only")
	}
}
