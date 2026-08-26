package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listThreadsInput struct {
	ChannelID       string `json:"channel_id" jsonschema:"channel or forum ID to enumerate threads in"`
	IncludeArchived *bool  `json:"include_archived,omitempty" jsonschema:"include archived threads and closed forum posts; default true"`
	Limit           int    `json:"limit,omitempty" jsonschema:"max threads to return, 1-100; default 50"`
}

func listThreadsHandler(d Deps) mcp.ToolHandlerFor[listThreadsInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in listThreadsInput) (*mcp.CallToolResult, any, error) {
		if in.ChannelID == "" {
			return nil, nil, fmt.Errorf("channel_id is required")
		}
		if _, err := d.resolveInGuild(ctx, in.ChannelID); err != nil {
			return nil, nil, err
		}
		includeArchived := in.IncludeArchived == nil || *in.IncludeArchived
		limit := in.Limit
		if limit <= 0 {
			limit = 50
		}

		active, err := d.Discord.ActiveThreads(ctx, d.GuildID)
		if err != nil {
			return nil, nil, fmt.Errorf("list threads: %w", err)
		}
		var lines []string
		for _, th := range active {
			if th.ParentID != in.ChannelID {
				continue
			}
			lines = append(lines, fmt.Sprintf("- [active] %s (%s) — %d messages", th.Name, th.ID, th.MessageCount))
		}

		if includeArchived {
			pub, err := d.Discord.ArchivedThreads(ctx, in.ChannelID, false)
			if err != nil {
				return nil, nil, fmt.Errorf("list archived threads: %w", err)
			}
			for _, th := range pub {
				lines = append(lines, fmt.Sprintf("- [archived] %s (%s) — %d messages", th.Name, th.ID, th.MessageCount))
			}
			if priv, err := d.Discord.ArchivedThreads(ctx, in.ChannelID, true); err == nil {
				for _, th := range priv {
					lines = append(lines, fmt.Sprintf("- [archived, private] %s (%s) — %d messages", th.Name, th.ID, th.MessageCount))
				}
			}
		}

		if len(lines) > limit {
			lines = lines[:limit]
		}
		if len(lines) == 0 {
			return text("No threads found."), nil, nil
		}
		return text(strings.Join(lines, "\n")), nil, nil
	}
}
