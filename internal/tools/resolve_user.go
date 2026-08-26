package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type resolveUserInput struct {
	UserID string `json:"user_id,omitempty" jsonschema:"a Discord user ID (snowflake) to look up"`
	Name   string `json:"name,omitempty" jsonschema:"a username, display name, or guild nickname to search for"`
}

func resolveUserHandler(d Deps) mcp.ToolHandlerFor[resolveUserInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in resolveUserInput) (*mcp.CallToolResult, any, error) {
		if (in.UserID == "") == (in.Name == "") {
			return nil, nil, fmt.Errorf("exactly one of user_id or name is required")
		}

		if in.UserID != "" {
			u, err := d.Discord.User(ctx, in.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("resolve user %s: %w", in.UserID, err)
			}
			return text(userLine(u)), nil, nil
		}

		named, err := d.Resolve.User(ctx, d.GuildID, in.Name)
		if err != nil {
			return nil, nil, err
		}
		u, err := d.Discord.User(ctx, named.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve user %s: %w", named.ID, err)
		}
		return text(userLine(u)), nil, nil
	}
}
