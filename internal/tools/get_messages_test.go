package tools_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/resolve"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

// Behavior 55: get_messages renders the mocked messages in ascending time
// order with authors and timestamps.
func TestGetMessages_RendersAscendingOrderWithAuthorsAndTimestamps(t *testing.T) {
	t.Parallel()
	out := callText(t, session(t), "get_messages", map[string]any{"channel_id": rulesChannelID})

	firstIdx := strings.Index(out, "does the L2 stay rested?")
	secondIdx := strings.Index(out, "second")
	if firstIdx == -1 || secondIdx == -1 || firstIdx > secondIdx {
		t.Fatalf("get_messages output not in ascending order:\n%s", out)
	}
	for _, want := range []string{"Alex", "Janak", "14:22", "14:25"} {
		if !strings.Contains(out, want) {
			t.Errorf("get_messages output missing %q:\n%s", want, out)
		}
	}
}

// Behavior 56: get_messages passes before/after/around/limit through to
// the Discord query string as given.
func TestGetMessages_PassesQueryParamsThroughVerbatim(t *testing.T) {
	t.Parallel()
	var gotQuery map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/channels/"+rulesChannelID+"/messages" {
			gotQuery = map[string][]string(r.URL.Query())
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + rulesChannelID + `","name":"rules-questions","type":0,"guild_id":"` + testGuildID + `"}`))
	}))
	t.Cleanup(srv.Close)

	d := discord.New(srv.URL, "test-token")
	deps := tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID}
	cs := sessionWithDeps(t, deps)

	callText(t, cs, "get_messages", map[string]any{
		"channel_id": rulesChannelID,
		"before":     "500",
		"limit":      25,
	})

	if got := gotQuery["before"]; len(got) != 1 || got[0] != "500" {
		t.Errorf("query[before] = %v, want [500]", got)
	}
	if got := gotQuery["limit"]; len(got) != 1 || got[0] != "25" {
		t.Errorf("query[limit] = %v, want [25]", got)
	}
}

// Behavior 57: get_messages with limit above 100 returns a tool error
// rather than calling upstream.
func TestGetMessages_LimitAbove100_ErrorsWithoutCallingUpstream(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	d := discord.New(srv.URL, "test-token")
	deps := tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID}
	cs := sessionWithDeps(t, deps)

	callError(t, cs, "get_messages", map[string]any{"channel_id": rulesChannelID, "limit": 500})

	if calls != 0 {
		t.Errorf("upstream received %d calls, want 0 (limit validation must fail before any request)", calls)
	}
}
