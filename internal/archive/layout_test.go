package archive_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/archive"
)

// Behavior 29: the channel directory name is <slug>-<channel_id>, and a
// channel renamed upstream writes to the same directory (resolution is
// keyed on ID, not on the current slug).
func TestResolveChannelDir(t *testing.T) {
	t.Parallel()

	t.Run("new channel computes slug-id directory", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		got, err := archive.ResolveChannelDir(root, "1122334455667788", "rules-questions")
		if err != nil {
			t.Fatalf("ResolveChannelDir: %v", err)
		}
		want := filepath.Join(root, "channels", "rules-questions-1122334455667788")
		if got != want {
			t.Errorf("ResolveChannelDir() = %q, want %q", got, want)
		}
	})

	t.Run("renamed channel resolves to its existing directory", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		existing := filepath.Join(root, "channels", "rules-questions-1122334455667788")
		if err := os.MkdirAll(existing, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		got, err := archive.ResolveChannelDir(root, "1122334455667788", "rules-qs")
		if err != nil {
			t.Fatalf("ResolveChannelDir: %v", err)
		}
		if got != existing {
			t.Errorf("ResolveChannelDir() = %q, want the existing directory %q (unaffected by the rename)", got, existing)
		}
		if strings.Contains(got, "rules-qs") {
			t.Errorf("ResolveChannelDir() = %q, must not create a second directory under the new slug", got)
		}
	})
}

// Behavior 30: _channel.md and guild.md are rebuilt purely from window
// frontmatter — deleting both and re-running reproduces them
// byte-identically.
func TestWriteIndexes_RebuildFromWindowFrontmatter_ByteIdenticalAfterDeletion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	channelDir := filepath.Join(root, "channels", "rules-questions-1122334455667788")
	if err := os.MkdirAll(channelDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	mustWrite(t, archive.WindowPath(channelDir, "2024-11"), windowMeta("2024-11", "100", "200", true))
	mustWrite(t, archive.WindowPath(channelDir, "2026-08"), windowMeta("2026-08", "501", "999", false))

	t.Run("_channel.md", func(t *testing.T) {
		if err := archive.WriteChannelIndex(channelDir); err != nil {
			t.Fatalf("WriteChannelIndex (first pass): %v", err)
		}
		first, err := os.ReadFile(filepath.Join(channelDir, "_channel.md"))
		if err != nil {
			t.Fatalf("ReadFile _channel.md: %v", err)
		}

		if err := os.Remove(filepath.Join(channelDir, "_channel.md")); err != nil {
			t.Fatalf("Remove _channel.md: %v", err)
		}
		if err := archive.WriteChannelIndex(channelDir); err != nil {
			t.Fatalf("WriteChannelIndex (second pass): %v", err)
		}
		second, err := os.ReadFile(filepath.Join(channelDir, "_channel.md"))
		if err != nil {
			t.Fatalf("ReadFile _channel.md (rebuilt): %v", err)
		}

		if string(first) != string(second) {
			t.Errorf("_channel.md not byte-identical after rebuild:\nfirst:  %q\nsecond: %q", first, second)
		}
	})

	t.Run("guild.md", func(t *testing.T) {
		if err := archive.WriteGuildIndex(root); err != nil {
			t.Fatalf("WriteGuildIndex (first pass): %v", err)
		}
		first, err := os.ReadFile(filepath.Join(root, "guild.md"))
		if err != nil {
			t.Fatalf("ReadFile guild.md: %v", err)
		}

		if err := os.Remove(filepath.Join(root, "guild.md")); err != nil {
			t.Fatalf("Remove guild.md: %v", err)
		}
		if err := archive.WriteGuildIndex(root); err != nil {
			t.Fatalf("WriteGuildIndex (second pass): %v", err)
		}
		second, err := os.ReadFile(filepath.Join(root, "guild.md"))
		if err != nil {
			t.Fatalf("ReadFile guild.md (rebuilt): %v", err)
		}

		if string(first) != string(second) {
			t.Errorf("guild.md not byte-identical after rebuild:\nfirst:  %q\nsecond: %q", first, second)
		}
	})
}
