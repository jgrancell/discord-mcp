package render_test

import (
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// baseMessage returns a minimal, valid plain-text message that individual
// tests mutate for their scenario. Fixed timestamp: 2026-06-03T14:22:00Z.
func baseMessage() discord.Message {
	return discord.Message{
		ID:        "1247891234567890123",
		ChannelID: "1122334455667788",
		GuildID:   "1100000000000000000",
		Author: discord.Author{
			ID:          "999000111222333444",
			Username:    "janak",
			DisplayName: "Janak",
		},
		Timestamp: time.Date(2026, 6, 3, 14, 22, 0, 0, time.UTC),
		Content:   "Rested state inherits up the stack on evolution.",
		Type:      0, // DEFAULT
	}
}

// emptyMentions is a MentionMap with nothing resolvable — the common case
// for tests that don't exercise mention resolution.
func emptyMentions() render.MentionMap { return render.MentionMap{} }
