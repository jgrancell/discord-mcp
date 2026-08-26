package mine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

func threadJSON(id, name, parentID string, messageCount int) string {
	return `{"id":"` + id + `","name":"` + name + `","parent_id":"` + parentID + `",` +
		`"message_count":` + strconv.Itoa(messageCount) + `,"thread_metadata":{"archived":false,"locked":false},` +
		`"owner_id":"999000111222333444"}`
}

// Behavior 45: a thread-creating message writes both the parent's pointer
// line and a thread file. Public threads created from a message share the
// message's snowflake as the thread ID — the same correlation Discord uses.
func TestRun_ThreadCreatingMessage_WritesPointerAndThreadFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const threadID = "100" // shares the origin message's ID

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(testChannelID, "rules-questions", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[` + threadJSON(threadID, "Rest inheritance", testChannelID, 14) + `]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/messages":
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` + discordMessageJSON(threadID, testChannelID, "does the L2 stay rested?", "2026-06-03T14:22:00.000000+00:00") + `]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case r.URL.Path == "/channels/"+threadID+"/messages":
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` + discordMessageJSON("101", threadID, "yes it does", "2026-06-03T14:23:00.000000+00:00") + `]`))
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
	windowBytes, err := os.ReadFile(filepath.Join(channelDir, "2026-06.md"))
	if err != nil {
		t.Fatalf("ReadFile window: %v", err)
	}
	if !strings.Contains(string(windowBytes), "→ thread:") {
		t.Errorf("window file missing thread pointer line:\n%s", windowBytes)
	}

	entries, err := os.ReadDir(filepath.Join(channelDir, "threads"))
	if err != nil {
		t.Fatalf("ReadDir threads/: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-"+threadID+".md") {
			found = true
		}
	}
	if !found {
		t.Errorf("threads/ directory %v does not contain a file for thread %s", entries, threadID)
	}
}

// Behavior 46: a forum channel produces thread files and a _channel.md with
// no window files (forum "channels" carry no messages of their own — every
// post is a thread).
func TestRun_ForumChannel_ProducesThreadFilesNoWindowFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const forumID = "1133445566778899"
	const postID = "200"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(forumID, "rules-forum", 15) + `]`)) // 15 = GUILD_FORUM
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[` + threadJSON(postID, "Evolution timing", forumID, 3) + `]}`))
		case r.URL.Path == "/channels/"+forumID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+forumID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+postID+"/messages":
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` + discordMessageJSON(postID, postID, "opening post", "2026-06-03T14:22:00.000000+00:00") + `]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case r.URL.Path == "/channels/"+forumID+"/messages":
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
	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	channelDir := filepath.Join(root, "channels", "rules-forum-"+forumID)
	entries, err := os.ReadDir(channelDir)
	if err != nil {
		t.Fatalf("ReadDir channel dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "_channel.md" {
			t.Errorf("forum channel directory has a window file %q, want none", e.Name())
		}
	}

	threadEntries, err := os.ReadDir(filepath.Join(channelDir, "threads"))
	if err != nil {
		t.Fatalf("ReadDir threads/: %v", err)
	}
	if len(threadEntries) == 0 {
		t.Error("forum channel produced no thread files, want at least one forum post file")
	}
}
