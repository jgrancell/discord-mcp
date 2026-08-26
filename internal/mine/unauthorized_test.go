package mine_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Regression coverage (QA iteration 3), for the design's "401/403"
// rate-limit policy (design doc ~line 374): a 401 means the bot token is
// bad, and retrying is precisely how a Cloudflare IP ban is earned, so a
// 401 mid-walk must abort the entire run immediately — never retried, and
// never treated as a per-channel gap the way a 403 is. This is an
// orchestration-level check, one layer above discord.TestClient_401_NeverRetried
// in internal/discord/auth_test.go: it asserts that mine.Run itself stops
// the walk before a later channel is ever requested, and that the defer
// contract (behavior 51a, mine.go:95-103) still leaves a partial report
// file even though Run returns an error.
func TestRun_401MidWalk_AbortsRunSkipsLaterChannelsWritesPartialReport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const channelA = testChannelID
	const channelB = "7766554433221100"

	channelBHits := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(channelA, "rules-questions", 0) + `,` + channelJSON(channelB, "off-topic", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelA+"/threads/archived/public",
			r.URL.Path == "/channels/"+channelA+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelA+"/messages":
			// The 401, mid-walk: channel A's thread enumeration above has
			// already succeeded; the token turns out to be bad once the
			// walk reaches its messages.
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"401: Unauthorized","code":0}`))
		case r.URL.Path == "/channels/"+channelB+"/threads/archived/public",
			r.URL.Path == "/channels/"+channelB+"/threads/archived/private":
			channelBHits++
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelB+"/messages":
			channelBHits++
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

	_, err := mine.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("Run: want an error on a 401 mid-walk, got nil")
	}
	if !errors.Is(err, discord.ErrUnauthorized) {
		t.Errorf("Run() err = %v, want errors.Is(err, discord.ErrUnauthorized)", err)
	}
	if channelBHits != 0 {
		t.Errorf("channel B's endpoints received %d requests after channel A's 401, want 0 (the run must abort before reaching a later channel)", channelBHits)
	}

	// The defer contract (behavior 51a): Run always writes the run's report
	// file, clean finish or abort alike. The partial report reflects
	// exactly how far the walk got: channel A recorded as attempted.
	_, content := soleReportFile(t, root)
	if !strings.Contains(content, channelA) {
		t.Errorf("partial report content missing attempted channel ID %q:\n%s", channelA, content)
	}
}
