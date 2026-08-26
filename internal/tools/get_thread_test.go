package tools_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/resolve"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

// Behavior 59: get_thread renders thread metadata plus its messages, and
// works when the mocked thread is archived.
func TestGetThread_RendersMetadataAndMessagesWhenArchived(t *testing.T) {
	t.Parallel()
	out := callText(t, session(t), "get_thread", map[string]any{"thread_id": threadID})

	for _, want := range []string{"Rest inheritance", "yes it does", "Janak"} {
		if !strings.Contains(out, want) {
			t.Errorf("get_thread output missing %q:\n%s", want, out)
		}
	}
}

// genMessages builds a JSON array of n synthetic messages in channelID,
// with snowflake IDs starting at idBase+1 and content labels "msg
// <contentStart>".."msg <contentStart+n-1>".
func genMessages(channelID string, idBase, contentStart, n int) string {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		id := idBase + i + 1
		fmt.Fprintf(&b, `{"id":"%d","channel_id":"%s","content":"msg %d","timestamp":"2026-06-03T14:22:00.000000+00:00","type":0,"author":{"id":"%s","username":"janak","global_name":"Janak"}}`,
			id, channelID, contentStart+i, userJanak)
	}
	b.WriteByte(']')
	return b.String()
}

// Behavior 60: get_thread paginates when the thread exceeds one page.
func TestGetThread_PaginatesWhenExceedingOnePage(t *testing.T) {
	t.Parallel()
	const base = 2000000000000000000
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/channels/"+threadID:
			_, _ = w.Write([]byte(`{"id":"` + threadID + `","name":"Long thread","parent_id":"` + rulesChannelID + `","guild_id":"` + testGuildID + `",` +
				`"thread_metadata":{"archived":false,"locked":false},"message_count":150,"owner_id":"` + userJanak + `"}`))
		case r.URL.Path == "/channels/"+threadID+"/messages":
			calls++
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(genMessages(threadID, base, 1, 100)))
			} else {
				_, _ = w.Write([]byte(genMessages(threadID, base+100, 101, 50)))
			}
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	d := discord.New(srv.URL, "test-token")
	cs := sessionWithDeps(t, tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID})

	out := callText(t, cs, "get_thread", map[string]any{"thread_id": threadID, "limit": 150})

	if calls < 2 {
		t.Errorf("upstream received %d messages calls, want at least 2 (one page cannot hold 150 messages)", calls)
	}
	for _, want := range []string{"msg 1", "msg 100", "msg 150"} {
		if !strings.Contains(out, want) {
			t.Errorf("get_thread output missing %q (paginated content), have:\n%.500s...", want, out)
		}
	}
}
