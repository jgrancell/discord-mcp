package tools_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/resolve"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

const (
	testGuildID       = "1100000000000000000"
	otherGuildID      = "9999999999999999999"
	rulesChannelID    = "1122334455667788"
	emptyChannelID    = "9900000000000000"
	otherGuildChannel = "5544332211009988"
	threadID          = "1247891300000000000"
	activeThreadID    = "1247891400000000000"
	userJanak         = "999000111222333444"
	userAlex          = "999000111222333445"
	userAlexandra     = "999000111222333446"
)

// mockAPI serves canned Discord REST responses for the tool surface.
func mockAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+testGuildID+"/channels":
			_, _ = w.Write([]byte(`[` +
				`{"id":"` + rulesChannelID + `","name":"rules-questions","type":0,"guild_id":"` + testGuildID + `","last_message_id":"1249999999999999999"},` +
				`{"id":"` + emptyChannelID + `","name":"scratch","type":0,"guild_id":"` + testGuildID + `"}` +
				`]`))

		case r.URL.Path == "/channels/"+rulesChannelID:
			_, _ = w.Write([]byte(`{"id":"` + rulesChannelID + `","name":"rules-questions","type":0,"guild_id":"` + testGuildID + `"}`))
		case r.URL.Path == "/channels/"+emptyChannelID:
			_, _ = w.Write([]byte(`{"id":"` + emptyChannelID + `","name":"scratch","type":0,"guild_id":"` + testGuildID + `"}`))
		case r.URL.Path == "/channels/"+otherGuildChannel:
			_, _ = w.Write([]byte(`{"id":"` + otherGuildChannel + `","name":"foreign","type":0,"guild_id":"` + otherGuildID + `"}`))
		case r.URL.Path == "/channels/"+threadID:
			_, _ = w.Write([]byte(`{"id":"` + threadID + `","name":"Rest inheritance","parent_id":"` + rulesChannelID + `","guild_id":"` + testGuildID + `",` +
				`"thread_metadata":{"archived":true,"locked":false},"message_count":2,"owner_id":"` + userJanak + `"}`))

		case r.URL.Path == "/channels/"+rulesChannelID+"/messages":
			_, _ = w.Write([]byte(`[` +
				`{"id":"200","channel_id":"` + rulesChannelID + `","content":"second","timestamp":"2026-06-03T14:25:00.000000+00:00","type":0,"author":{"id":"` + userJanak + `","username":"janak","global_name":"Janak"}},` +
				`{"id":"100","channel_id":"` + rulesChannelID + `","content":"does the L2 stay rested?","timestamp":"2026-06-03T14:22:00.000000+00:00","type":0,"author":{"id":"` + userAlex + `","username":"alex","global_name":"Alex"}}` +
				`]`))
		case r.URL.Path == "/channels/"+threadID+"/messages":
			_, _ = w.Write([]byte(`[` +
				`{"id":"101","channel_id":"` + threadID + `","content":"yes it does","timestamp":"2026-06-03T14:23:00.000000+00:00","type":0,"author":{"id":"` + userJanak + `","username":"janak","global_name":"Janak"}}` +
				`]`))

		case r.URL.Path == "/guilds/"+testGuildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[{"id":"` + activeThreadID + `","name":"Card game night","parent_id":"` + rulesChannelID + `",` +
				`"thread_metadata":{"archived":false,"locked":false},"message_count":5,"owner_id":"` + userJanak + `"}]}`))
		case r.URL.Path == "/channels/"+rulesChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[{"id":"` + threadID + `","name":"Rest inheritance","parent_id":"` + rulesChannelID + `",` +
				`"thread_metadata":{"archived":true,"locked":false},"message_count":2,"owner_id":"` + userJanak + `"}]}`))
		case r.URL.Path == "/channels/"+rulesChannelID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))

		case r.URL.Path == "/guilds/"+testGuildID+"/members/search":
			q := r.URL.Query().Get("query")
			switch q {
			case "janak":
				_, _ = w.Write([]byte(`[{"user":{"id":"` + userJanak + `","username":"janak","global_name":"Janak","bot":false},"nick":null}]`))
			case "alex", "ale":
				_, _ = w.Write([]byte(`[` +
					`{"user":{"id":"` + userAlex + `","username":"alex","global_name":"Alex","bot":false},"nick":null},` +
					`{"user":{"id":"` + userAlexandra + `","username":"alexandra","global_name":"Alexandra","bot":false},"nick":null}` +
					`]`))
			case "nobody-like-this":
				_, _ = w.Write([]byte(`[]`))
			default:
				_, _ = w.Write([]byte(`[]`))
			}
		case r.URL.Path == "/users/"+userJanak:
			_, _ = w.Write([]byte(`{"id":"` + userJanak + `","username":"janak","global_name":"Janak","bot":false}`))

		default:
			t.Errorf("unexpected API call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func session(t *testing.T) *mcp.ClientSession {
	t.Helper()
	api := mockAPI(t)
	d := discord.New(api.URL, "test-token")
	deps := tools.Deps{Discord: d, Resolve: resolve.New(d), GuildID: testGuildID}
	return sessionWithDeps(t, deps)
}

// sessionWithDeps wires an in-memory MCP session against the given Deps —
// used by tests that need a custom mock upstream (assertions on outbound
// requests, call counts, or a foreign guild).
func sessionWithDeps(t *testing.T, deps tools.Deps) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "discord-mcp-test", Version: "test"}, nil)
	tools.Register(server, deps)

	st, ct := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	cl := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := cl.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// callText calls a tool and returns its rendered text, failing the test if
// the call errored or the tool itself returned an error result.
func callText(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("CallTool %s: empty content", name)
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool %s: content[0] = %T", name, res.Content[0])
	}
	if res.IsError {
		t.Fatalf("CallTool %s returned tool error: %s", name, tc.Text)
	}
	return tc.Text
}

// callError calls a tool expecting a tool-level error and returns its text.
func callError(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if !res.IsError {
		t.Fatalf("CallTool %s: want a tool error, got a normal result", name)
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool %s: content[0] = %T", name, res.Content[0])
	}
	return tc.Text
}
