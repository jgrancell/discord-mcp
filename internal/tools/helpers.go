package tools

import (
	"context"
	"fmt"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// resolveInGuild fetches a channel or thread by ID and verifies it
// belongs to the configured guild — the privacy boundary enforced by
// every tool that takes a channel or thread ID.
func (d Deps) resolveInGuild(ctx context.Context, id string) (discord.Channel, error) {
	ch, err := d.Discord.Channel(ctx, id)
	if err != nil {
		return discord.Channel{}, fmt.Errorf("fetch channel %s: %w", id, err)
	}
	if ch.GuildID != d.GuildID {
		return discord.Channel{}, fmt.Errorf("channel %s belongs to a different guild (not %s)", id, d.GuildID)
	}
	return ch, nil
}

// renderMessages renders a batch of messages with no mention resolution —
// the tool surface is a live-fetch view; mention resolution is mine's
// per-run concern.
func renderMessages(msgs []discord.Message) string {
	out := render.Messages(msgs, render.MentionMap{})
	if out == "" {
		return "No messages."
	}
	return out
}

func userLine(u discord.User) string {
	s := fmt.Sprintf("%s (%s)", u.DisplayName, u.Username)
	if u.Bot {
		s += " [bot]"
	}
	if u.Nick != "" {
		s += fmt.Sprintf(" — nick: %s", u.Nick)
	}
	s += fmt.Sprintf(" — id: %s", u.ID)
	return s
}
