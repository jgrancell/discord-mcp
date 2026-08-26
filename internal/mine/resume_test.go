package mine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
	"github.com/jgrancell/discord-mcp/internal/render"
)

const testChannelID = "1122334455667788"

// Behavior 43: a resumed run against an archive containing prior windows
// issues its first request with after=<derived cursor>, not from the
// beginning.
func TestRun_ResumedRun_FirstRequestUsesDerivedCursor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	channelDir := filepath.Join(root, "channels", "rules-questions-"+testChannelID)
	meta := render.WindowMeta{
		GuildID: guildID, GuildName: "OMGCG", ChannelID: testChannelID,
		ChannelName: "rules-questions", ChannelType: "text",
		Window: "2026-07", LastMessageID: "1249000000000000000", FirstMessageID: "1248000000000000000",
		MessageCount: 1, Complete: true, MinerVersion: "0.1.0", Format: 1,
	}
	if _, err := archive.WriteWindow(archive.WindowPath(channelDir, "2026-07"), meta, "## 2026-07-01\n"); err != nil {
		t.Fatalf("seed WriteWindow: %v", err)
	}

	var gotAfter string
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
			if gotAfter == "" {
				gotAfter = r.URL.Query().Get("after")
			}
			_, _ = w.Write([]byte(`[]`))
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
	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotAfter != "1249000000000000000" {
		t.Errorf("first messages request after=%q, want the derived cursor %q", gotAfter, "1249000000000000000")
	}
}
