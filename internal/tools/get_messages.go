package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

type getMessagesInput struct {
	ChannelID string `json:"channel_id" jsonschema:"channel or thread ID"`
	Before    string `json:"before,omitempty" jsonschema:"return messages older than this message ID"`
	After     string `json:"after,omitempty" jsonschema:"return messages newer than this message ID"`
	Around    string `json:"around,omitempty" jsonschema:"return messages centered on this message ID"`
	Limit     int    `json:"limit,omitempty" jsonschema:"how many messages, 1-100; default 50"`
}

func getMessagesHandler(d Deps) mcp.ToolHandlerFor[getMessagesInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in getMessagesInput) (*mcp.CallToolResult, any, error) {
		if in.ChannelID == "" {
			return nil, nil, fmt.Errorf("channel_id is required")
		}
		if in.Limit > 100 {
			return nil, nil, fmt.Errorf("limit %d exceeds the maximum of 100", in.Limit)
		}
		if _, err := d.resolveInGuild(ctx, in.ChannelID); err != nil {
			return nil, nil, err
		}

		msgs, err := d.Discord.Messages(ctx, in.ChannelID, discord.MessageParams{
			Before: in.Before, After: in.After, Around: in.Around, Limit: in.Limit,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("get messages: %w", err)
		}
		return text(renderMessages(msgs)), nil, nil
	}
}
