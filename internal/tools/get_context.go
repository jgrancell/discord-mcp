package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

type getContextInput struct {
	ChannelID string `json:"channel_id" jsonschema:"channel or thread ID the message lives in"`
	MessageID string `json:"message_id" jsonschema:"the message to center on"`
	Span      int    `json:"span,omitempty" jsonschema:"messages on each side, 1-50; default 15"`
}

func getContextHandler(d Deps) mcp.ToolHandlerFor[getContextInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in getContextInput) (*mcp.CallToolResult, any, error) {
		if in.ChannelID == "" || in.MessageID == "" {
			return nil, nil, fmt.Errorf("channel_id and message_id are required")
		}
		if _, err := d.resolveInGuild(ctx, in.ChannelID); err != nil {
			return nil, nil, err
		}

		span := in.Span
		if span <= 0 {
			span = 15
		}
		msgs, err := d.Discord.Messages(ctx, in.ChannelID, discord.MessageParams{
			Around: in.MessageID, Limit: span*2 + 1,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("get context: %w", err)
		}
		return text(markTarget(renderMessages(msgs), in.MessageID)), nil, nil
	}
}

// markTarget visibly distinguishes the target message from its neighbours
// by inserting a marker line directly above its heading.
func markTarget(rendered, targetID string) string {
	marker := "[#" + targetID + "]"
	idx := strings.Index(rendered, marker)
	if idx == -1 {
		return rendered
	}
	lineStart := strings.LastIndex(rendered[:idx], "### ")
	if lineStart == -1 {
		lineStart = 0
	}
	return rendered[:lineStart] + "**target message:**\n" + rendered[lineStart:]
}
