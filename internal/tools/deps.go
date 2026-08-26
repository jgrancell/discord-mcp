package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/resolve"
)

// Deps carries what every tool handler needs. One Discord client serves
// all sessions; GuildID is the hard privacy scope every tool enforces.
type Deps struct {
	Discord *discord.Client
	Resolve *resolve.Resolver
	GuildID string
}

// text wraps a string as a tool result.
func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}
}
