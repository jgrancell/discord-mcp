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

// Behavior 62: resolve_user with user_id renders username, display name,
// and bot flag.
func TestResolveUser_ByUserID_RendersUsernameDisplayNameBotFlag(t *testing.T) {
	t.Parallel()
	out := callText(t, session(t), "resolve_user", map[string]any{"user_id": userJanak})

	for _, want := range []string{"janak", "Janak"} {
		if !strings.Contains(out, want) {
			t.Errorf("resolve_user output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "[bot]") {
		t.Errorf("resolve_user output = %q, Janak is not a bot", out)
	}
}

// Behavior 63: resolve_user with an exact-match name resolves to one user.
func TestResolveUser_ExactMatchName_ResolvesToOneUser(t *testing.T) {
	t.Parallel()
	out := callText(t, session(t), "resolve_user", map[string]any{"name": "janak"})

	if !strings.Contains(out, "Janak") {
		t.Errorf("resolve_user output missing %q:\n%s", "Janak", out)
	}
	if strings.Contains(strings.ToLower(out), "which one") {
		t.Errorf("resolve_user output asks for clarification on an exact match:\n%s", out)
	}
}

// Behavior 64: resolve_user with an ambiguous name returns a tool error
// listing candidates and resolves nothing.
func TestResolveUser_AmbiguousName_ErrorsListingCandidates(t *testing.T) {
	t.Parallel()
	out := callError(t, session(t), "resolve_user", map[string]any{"name": "ale"})

	for _, want := range []string{"Alex", "Alexandra"} {
		if !strings.Contains(out, want) {
			t.Errorf("ambiguous resolve_user error missing candidate %q:\n%s", want, out)
		}
	}
	if !strings.Contains(strings.ToLower(out), "which one") {
		t.Errorf("ambiguous resolve_user error does not ask which one:\n%s", out)
	}
}

// Behavior 65: resolve_user with a name matching nothing returns a "not
// found" tool error naming what was searched.
func TestResolveUser_NoMatch_ErrorsNamingSearchTerm(t *testing.T) {
	t.Parallel()
	out := callError(t, session(t), "resolve_user", map[string]any{"name": "nobody-like-this"})

	if !strings.Contains(strings.ToLower(out), "not found") {
		t.Errorf("resolve_user no-match error missing \"not found\":\n%s", out)
	}
	if !strings.Contains(out, "nobody-like-this") {
		t.Errorf("resolve_user no-match error does not name the search term:\n%s", out)
	}
}

// Behavior 66: resolve_user with both user_id and name, or neither, returns
// a tool error without calling upstream.
func TestResolveUser_BothOrNeitherInput_ErrorsWithoutCallingUpstream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "both user_id and name", args: map[string]any{"user_id": userJanak, "name": "janak"}},
		{name: "neither", args: map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			callError(t, cs, "resolve_user", tt.args)

			if calls != 0 {
				t.Errorf("upstream received %d calls, want 0", calls)
			}
		})
	}
}
