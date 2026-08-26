package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

type listChannelsInput struct {
	IncludeEmpty bool `json:"include_empty,omitempty" jsonschema:"include channels with no messages; default false"`
}

func listChannelsHandler(d Deps) mcp.ToolHandlerFor[listChannelsInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in listChannelsInput) (*mcp.CallToolResult, any, error) {
		channels, err := d.Discord.Channels(ctx, d.GuildID)
		if err != nil {
			return nil, nil, fmt.Errorf("list channels: %w", err)
		}

		categories := map[string]string{}
		for _, c := range channels {
			if c.Type == 4 {
				categories[c.ID] = c.Name
			}
		}

		var lines []string
		for _, c := range channels {
			if c.Type == 4 {
				continue // categories are structure, not a mineable channel
			}
			if !in.IncludeEmpty && c.LastMessageID == "" {
				continue
			}
			line := fmt.Sprintf("- %s (%s) [%s]", c.Name, c.ID, discord.ChannelTypeName(c.Type))
			if cat, ok := categories[c.ParentID]; ok {
				line += " — category: " + cat
			}
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			return text("No channels found."), nil, nil
		}
		return text(strings.Join(lines, "\n")), nil, nil
	}
}
