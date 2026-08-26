package archive_test

import (
	"time"

	"github.com/jgrancell/discord-mcp/internal/render"
)

// windowMeta returns a valid, complete-by-default window frontmatter value
// for the given window string ("2026-06") and last message ID; individual
// tests override fields as needed.
func windowMeta(window, firstID, lastID string, complete bool) render.WindowMeta {
	return render.WindowMeta{
		GuildID:        "1100000000000000000",
		GuildName:      "OMGCG",
		ChannelID:      "1122334455667788",
		ChannelName:    "rules-questions",
		ChannelType:    "text",
		ParentCategory: "Game",
		Window:         window,
		WindowStart:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		WindowEnd:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		FirstMessageID: firstID,
		LastMessageID:  lastID,
		MessageCount:   2,
		Complete:       complete,
		MinedAt:        time.Date(2026, 6, 30, 23, 14, 2, 0, time.UTC),
		MinerVersion:   "0.1.0",
		Format:         1,
	}
}
