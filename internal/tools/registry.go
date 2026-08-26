// Package tools is the MCP surface: one file per tool, registered against
// a live Discord client. The only package importing the MCP SDK.
package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

const toolCount = 6

// Register adds every tool to the server.
func Register(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_channels",
		Description: "List the guild's channels: names, IDs, types, and categories. The entry point — nothing else can be called without a channel ID.",
	}, listChannelsHandler(d))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_threads",
		Description: "List active and archived threads (and forum posts) under a channel",
	}, listThreadsHandler(d))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_messages",
		Description: "Page through a channel or thread's messages with before/after/around/limit",
	}, getMessagesHandler(d))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_context",
		Description: "Expand a message into the conversation around it — the follow-up to a grep hit in the archive",
	}, getContextHandler(d))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_thread",
		Description: "Read a thread or forum post's metadata and full message history, including archived threads",
	}, getThreadHandler(d))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "resolve_user",
		Description: "Look up a Discord user by ID, or search by username/display name/nickname",
	}, resolveUserHandler(d))
}

// Count returns the number of registered tools.
func Count() int { return toolCount }
