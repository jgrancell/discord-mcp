package mine_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Behavior 49: SIGINT mid-walk flushes the open window with a cursor, and a
// subsequent run resumes without re-reading the closed portion.
//
// Modeled here as context cancellation (mine.Run's SIGINT handler is
// expected to cancel the context it passes down; testing the context path
// directly is the deterministic, no-real-signal equivalent). Two channels
// are processed strictly sequentially per the design's walk order; the
// second channel's first request cancels the context, proving the first
// channel's data survives while the second channel's walk never starts.
func TestRun_ContextCanceledMidWalk_FlushesOpenWindowThenResumesWithoutRereading(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const channelA = testChannelID
	const channelB = "7766554433221100"

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	channelBCalls := 0
	channelAAfterOnRun2 := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(channelA, "rules-questions", 0) + `,` + channelJSON(channelB, "off-topic", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelA+"/threads/archived/public",
			r.URL.Path == "/channels/"+channelA+"/threads/archived/private",
			r.URL.Path == "/channels/"+channelB+"/threads/archived/public",
			r.URL.Path == "/channels/"+channelB+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelA+"/messages":
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` + discordMessageJSON("100", channelA, "hello from A", "2026-06-03T14:22:00.000000+00:00") + `]`))
			} else {
				channelAAfterOnRun2 = after
				_, _ = w.Write([]byte(`[]`))
			}
		case r.URL.Path == "/channels/"+channelB+"/messages":
			channelBCalls++
			if channelBCalls == 1 {
				// First run reaches channel B: cancel now. Channel A's
				// entire per-channel sequence has already completed
				// (strictly sequential walk order).
				cancel1()
				return
			}
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` + discordMessageJSON("200", channelB, "hello from B", "2026-06-03T15:00:00.000000+00:00") + `]`))
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

	_, err := mine.Run(ctx1, cfg)
	if err == nil {
		t.Fatal("Run: want an error when the context is canceled mid-walk, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run() err = %v, want errors.Is(err, context.Canceled)", err)
	}

	channelADir := filepath.Join(root, "channels", "rules-questions-"+channelA)
	meta, _, err := archive.ReadWindow(archive.WindowPath(channelADir, "2026-06"))
	if err != nil {
		t.Fatalf("ReadWindow(channel A, 2026-06) after cancellation: %v (channel A's data must survive)", err)
	}
	if meta.LastMessageID != "100" {
		t.Errorf("channel A window last_message_id = %q, want %q", meta.LastMessageID, "100")
	}

	channelBDir := filepath.Join(root, "channels", "off-topic-"+channelB)
	if _, err := os.Stat(channelBDir); !os.IsNotExist(err) {
		t.Errorf("channel B directory exists after mid-walk cancellation (err=%v), want it never started", err)
	}

	// A subsequent run, fresh context, resumes: channel A is not re-read
	// from the beginning, and channel B (never reached in run 1) proceeds.
	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if channelAAfterOnRun2 == "" || channelAAfterOnRun2 == "0" {
		t.Errorf("channel A after= on run 2 = %q, want the resume cursor from run 1 (not a fresh start)", channelAAfterOnRun2)
	}
	if _, err := os.Stat(filepath.Join(channelBDir, "2026-06.md")); err != nil {
		t.Errorf("channel B window file missing after the resumed run: %v", err)
	}
}
