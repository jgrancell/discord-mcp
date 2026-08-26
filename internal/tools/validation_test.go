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

// Behavior 68: a tool given a missing required input returns a tool error
// and makes no upstream call.
func TestGetMessages_MissingChannelID_ErrorsWithoutCallingUpstream(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	d := discord.New(srv.URL, "test-token")
	cs := sessionWithDeps(t, tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID})

	callError(t, cs, "get_messages", map[string]any{})

	if calls != 0 {
		t.Errorf("upstream received %d calls, want 0 for a missing required channel_id", calls)
	}
}

// Behavior 69: an upstream 500 surfaces as a tool error whose text names
// the operation.
func TestListChannels_Upstream500_ErrorsNamingTheOperation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	d := discord.New(srv.URL, "test-token")
	cs := sessionWithDeps(t, tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID})

	out := callError(t, cs, "list_channels", map[string]any{})

	if !strings.Contains(strings.ToLower(out), "channel") {
		t.Errorf("list_channels 500 error does not name the operation:\n%s", out)
	}
}

// Behavior 70: no tool result contains raw JSON — results are prose.
func TestTools_Results_ContainNoRawJSON(t *testing.T) {
	t.Parallel()
	cs := session(t)
	tests := []struct {
		tool string
		args map[string]any
	}{
		{tool: "list_channels", args: map[string]any{}},
		{tool: "get_messages", args: map[string]any{"channel_id": rulesChannelID}},
		{tool: "resolve_user", args: map[string]any{"user_id": userJanak}},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()
			out := callText(t, cs, tt.tool, tt.args)
			for _, jsonSignature := range []string{`{"`, `":"`, `":[`, `null`} {
				if strings.Contains(out, jsonSignature) {
					t.Errorf("%s output contains raw-JSON signature %q:\n%s", tt.tool, jsonSignature, out)
				}
			}
		})
	}
}
