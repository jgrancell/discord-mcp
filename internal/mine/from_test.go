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

// Behavior 48: --from <id> overrides the derived cursor for the named
// channel — a repair path that ignores the archive's own resume cursor.
func TestRun_FromFlag_OverridesDerivedCursorForNamedChannel(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	channelDir := filepath.Join(root, "channels", "rules-questions-"+testChannelID)
	seedMeta := render.WindowMeta{
		GuildID: guildID, GuildName: "OMGCG", ChannelID: testChannelID,
		ChannelName: "rules-questions", ChannelType: "text",
		Window: "2026-07", LastMessageID: "1249000000000000000", FirstMessageID: "1248000000000000000",
		MessageCount: 1, Complete: true, MinerVersion: "0.1.0", Format: 1,
	}
	if _, err := archive.WriteWindow(archive.WindowPath(channelDir, "2026-07"), seedMeta, "## 2026-07-01\n"); err != nil {
		t.Fatalf("seed WriteWindow: %v", err)
	}

	const repairFrom = "500000000000000000"
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
		Channel: testChannelID, From: repairFrom,
		Now: func() time.Time { return fixedNow },
	}
	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotAfter != repairFrom {
		t.Errorf("first messages request after=%q, want the --from override %q (not the derived cursor)", gotAfter, repairFrom)
	}
}
