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

// Behavior 58: get_context issues one around request with the right center
// and renders the target message marked distinctly from its neighbours.
func TestGetContext_IssuesOneAroundRequestMarksTargetDistinctly(t *testing.T) {
	t.Parallel()
	var gotAround string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/channels/"+rulesChannelID:
			_, _ = w.Write([]byte(`{"id":"` + rulesChannelID + `","name":"rules-questions","type":0,"guild_id":"` + testGuildID + `"}`))
		case r.URL.Path == "/channels/"+rulesChannelID+"/messages":
			calls++
			gotAround = r.URL.Query().Get("around")
			_, _ = w.Write([]byte(`[` +
				`{"id":"100","channel_id":"` + rulesChannelID + `","content":"before","timestamp":"2026-06-03T14:20:00.000000+00:00","type":0,"author":{"id":"` + userAlex + `","username":"alex","global_name":"Alex"}},` +
				`{"id":"150","channel_id":"` + rulesChannelID + `","content":"the target message","timestamp":"2026-06-03T14:22:00.000000+00:00","type":0,"author":{"id":"` + userJanak + `","username":"janak","global_name":"Janak"}},` +
				`{"id":"200","channel_id":"` + rulesChannelID + `","content":"after","timestamp":"2026-06-03T14:24:00.000000+00:00","type":0,"author":{"id":"` + userAlex + `","username":"alex","global_name":"Alex"}}` +
				`]`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	d := discord.New(srv.URL, "test-token")
	cs := sessionWithDeps(t, tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID})

	out := callText(t, cs, "get_context", map[string]any{
		"channel_id": rulesChannelID,
		"message_id": "150",
	})

	if calls != 1 {
		t.Errorf("upstream received %d messages calls, want exactly 1", calls)
	}
	if gotAround != "150" {
		t.Errorf("around= %q, want %q", gotAround, "150")
	}

	targetIdx := strings.Index(out, "the target message")
	if targetIdx == -1 {
		t.Fatalf("get_context output missing the target message:\n%s", out)
	}
	// The target must be visually distinguished from its neighbours: some
	// marker text appears on or immediately around its heading that does
	// not appear near the neighbouring messages.
	beforeIdx := strings.Index(out, "before")
	afterIdx := strings.Index(out, "after")
	if beforeIdx == -1 || afterIdx == -1 {
		t.Fatalf("get_context output missing a neighbour message:\n%s", out)
	}
	targetRegion := out[max(0, targetIdx-120):targetIdx]
	neighbourRegion := out[max(0, beforeIdx-120):beforeIdx] + out[max(0, afterIdx-120):afterIdx]
	distinguishingMarkers := []string{"target", "here", "→", "**"}
	marked := false
	for _, m := range distinguishingMarkers {
		if strings.Contains(targetRegion, m) && !strings.Contains(neighbourRegion, m) {
			marked = true
		}
	}
	if !marked {
		t.Errorf("target message not visibly distinguished from its neighbours:\n%s", out)
	}
}
