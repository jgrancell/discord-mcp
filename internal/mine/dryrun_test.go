package mine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Behavior 47: --dry-run writes nothing to the archive and still produces a
// plan report.
func TestRun_DryRun_WritesNothingProducesPlanReport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(testChannelID, "rules-questions", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/messages":
			_, _ = w.Write([]byte(`[` + discordMessageJSON("100", testChannelID, "hello", "2026-06-03T14:22:00.000000+00:00") + `]`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "test-token")
	cfg := mine.Config{
		Discord: c, Root: root, GuildID: guildID, GuildName: "OMGCG",
		DryRun: true,
		Now:    func() time.Time { return fixedNow },
	}
	report, err := mine.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(report.ChannelsAttempted) == 0 {
		t.Error("dry-run report has no attempted channels, want a plan naming at least one")
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("dry-run wrote to the archive root: %v, want no files written", entries)
	}

	if _, err := os.Stat(filepath.Join(root, "channels", "rules-questions-"+testChannelID)); !os.IsNotExist(err) {
		t.Errorf("dry-run created a channel directory, want none (err=%v)", err)
	}
}
