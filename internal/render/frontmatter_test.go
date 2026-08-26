package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behavior 20: window frontmatter renders every documented key, with
// complete: false on an open window. format: 1 is the message-format
// version (operator decision, required from day one).
func TestWindowFrontmatter_RendersAllDocumentedKeysOpenWindow(t *testing.T) {
	t.Parallel()
	meta := render.WindowMeta{
		GuildID:        "1100000000000000000",
		GuildName:      "OMGCG",
		ChannelID:      "1122334455667788",
		ChannelName:    "rules-questions",
		ChannelType:    "text",
		ParentCategory: "Game",
		Window:         "2026-06",
		WindowStart:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		WindowEnd:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		FirstMessageID: "1247000000000000000",
		LastMessageID:  "1249999999999999999",
		MessageCount:   412,
		Complete:       false,
		MinedAt:        time.Date(2026, 6, 30, 23, 14, 2, 0, time.UTC),
		MinerVersion:   "0.1.0",
		Format:         1,
	}

	got := render.WindowFrontmatter(meta)

	if !strings.HasPrefix(got, "---\n") {
		t.Fatalf("WindowFrontmatter() = %q, does not open with --- frontmatter fence", got)
	}
	for _, want := range []string{
		`guild_id: "1100000000000000000"`,
		"guild_name: OMGCG",
		`channel_id: "1122334455667788"`,
		"channel_name: rules-questions",
		"channel_type: text",
		"parent_category: Game",
		"window: 2026-06",
		"window_start: 2026-06-01T00:00:00Z",
		"window_end: 2026-07-01T00:00:00Z",
		"timezone: UTC",
		`first_message_id: "1247000000000000000"`,
		`last_message_id: "1249999999999999999"`,
		"message_count: 412",
		"complete: false",
		"mined_at: 2026-06-30T23:14:02Z",
		"miner_version: 0.1.0",
		"format: 1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("WindowFrontmatter() missing %q in:\n%s", want, got)
		}
	}
}

// Behavior 21: thread frontmatter includes thread_id, parent_channel_id,
// archived, locked, and applied_tags for a forum post.
func TestThreadFrontmatter_IncludesThreadSpecificKeysForForumPost(t *testing.T) {
	t.Parallel()
	meta := render.ThreadMeta{
		GuildID:         "1100000000000000000",
		GuildName:       "OMGCG",
		ParentChannelID: "1122334455667788",
		ThreadID:        "1247891300000000000",
		ThreadName:      "Rest inheritance",
		Archived:        true,
		Locked:          false,
		CreatedAt:       time.Date(2026, 6, 3, 14, 22, 0, 0, time.UTC),
		OwnerID:         "999000111222333444",
		MessageCount:    14,
		AppliedTags:     []string{"rules-question", "resolved"},
		Format:          1,
	}

	got := render.ThreadFrontmatter(meta)

	for _, want := range []string{
		`thread_id: "1247891300000000000"`,
		`parent_channel_id: "1122334455667788"`,
		"archived: true",
		"locked: false",
		"applied_tags:",
		"rules-question",
		"resolved",
		"format: 1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ThreadFrontmatter() missing %q in:\n%s", want, got)
		}
	}
}
