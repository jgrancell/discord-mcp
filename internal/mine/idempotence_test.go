package mine_test

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Behavior 51: two consecutive full runs with no new upstream messages
// produce byte-identical archive contents, excluding _runs/ (operator
// ruling 2026-08-22: per-run report files are a durable audit trail and are
// legitimately unique per run — see snapshotFiles).
func TestRun_TwoConsecutiveRunsNoNewMessages_ByteIdenticalArchive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

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
				_, _ = w.Write([]byte(`[` +
					discordMessageJSON("100", testChannelID, "first", "2026-06-03T14:22:00.000000+00:00") + `,` +
					discordMessageJSON("101", testChannelID, "second", "2026-06-03T14:25:00.000000+00:00") + `]`))
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
		t.Fatalf("first Run: %v", err)
	}
	first := snapshotFiles(t, root)

	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	second := snapshotFiles(t, root)

	if len(first) != len(second) {
		t.Fatalf("file count changed between runs: first %d, second %d\nfirst: %v\nsecond: %v", len(first), len(second), keys(first), keys(second))
	}
	for path, wantBytes := range first {
		gotBytes, ok := second[path]
		if !ok {
			t.Errorf("file %s present after run 1 but missing after run 2", path)
			continue
		}
		if string(gotBytes) != string(wantBytes) {
			t.Errorf("file %s not byte-identical between runs:\nrun 1: %q\nrun 2: %q", path, wantBytes, gotBytes)
		}
	}
}

// snapshotFiles reads every regular file under root into a map keyed by its
// path relative to root, excluding the _runs/ subtree. Per-run report files
// under _runs/ are a durable audit trail, not archive content: they are
// legitimately unique per run (their name is derived from the run's start
// time), so behavior 51's idempotence comparison excludes them. See design
// doc, operator ruling 2026-08-22, behavior 51.
func snapshotFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "_runs" {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = b
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotFiles: %v", err)
	}
	return out
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
