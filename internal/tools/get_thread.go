package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

type getThreadInput struct {
	ThreadID string `json:"thread_id" jsonschema:"thread or forum post ID"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max messages, 1-500; default 200"`
}

const threadPageSize = 100

func getThreadHandler(d Deps) mcp.ToolHandlerFor[getThreadInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in getThreadInput) (*mcp.CallToolResult, any, error) {
		if in.ThreadID == "" {
			return nil, nil, fmt.Errorf("thread_id is required")
		}
		ch, err := d.resolveInGuild(ctx, in.ThreadID)
		if err != nil {
			return nil, nil, err
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 200
		}

		var msgs []discord.Message
		cursor := "0"
		for len(msgs) < limit {
			batch, err := d.Discord.Messages(ctx, in.ThreadID, discord.MessageParams{After: cursor, Limit: threadPageSize})
			if err != nil {
				return nil, nil, fmt.Errorf("get thread %s: %w", in.ThreadID, err)
			}
			if len(batch) == 0 {
				break
			}
			msgs = append(msgs, batch...)
			cursor = discord.MaxMessageID(batch)
			if len(batch) < threadPageSize {
				break
			}
		}
		if len(msgs) > limit {
			msgs = msgs[:limit]
		}

		var b strings.Builder
		fmt.Fprintf(&b, "Thread: %s (%s)\n", ch.Name, ch.ID)
		status := "active"
		if ch.ThreadMetadata != nil && ch.ThreadMetadata.Archived {
			status = "archived"
		}
		fmt.Fprintf(&b, "Status: %s", status)
		if ch.ThreadMetadata != nil && ch.ThreadMetadata.Locked {
			b.WriteString(", locked")
		}
		b.WriteString("\n\n")
		b.WriteString(renderMessages(msgs))
		return text(b.String()), nil, nil
	}
}
