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
)

// Behavior 44: messages crossing a month boundary close the prior window
// with complete: true and open the next.
func TestRun_MessagesCrossMonthBoundary_ClosesPriorOpensNext(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	messages := `[` +
		discordMessageJSON("100", testChannelID, "june early", "2026-06-15T10:00:00.000000+00:00") + `,` +
		discordMessageJSON("101", testChannelID, "june late", "2026-06-28T10:00:00.000000+00:00") + `,` +
		discordMessageJSON("102", testChannelID, "july", "2026-07-02T10:00:00.000000+00:00") +
		`]`

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
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(messages))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
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

	channelDir := filepath.Join(root, "channels", "rules-questions-"+testChannelID)

	juneMeta, _, err := archive.ReadWindow(archive.WindowPath(channelDir, "2026-06"))
	if err != nil {
		t.Fatalf("ReadWindow(2026-06): %v", err)
	}
	if !juneMeta.Complete {
		t.Error("2026-06 window Complete = false, want true (closed by the July message)")
	}

	julyMeta, _, err := archive.ReadWindow(archive.WindowPath(channelDir, "2026-07"))
	if err != nil {
		t.Fatalf("ReadWindow(2026-07): %v", err)
	}
	if julyMeta.Complete {
		t.Error("2026-07 window Complete = true, want false (the newest, still-open window)")
	}
}
