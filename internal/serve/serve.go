// Package serve wires the HTTP MCP server: mux, health check, and graceful
// shutdown. Imports neither internal/archive nor internal/mine.
package serve

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/auth"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

// Version is reported to MCP clients.
const Version = "0.1.0"

// Config configures the HTTP mux.
type Config struct {
	Tools tools.Deps

	// AuthToken guards /mcp with a static bearer challenge when set. Empty
	// means loopback-only trust: no inbound credential.
	AuthToken string
}

// NewMux builds the server's HTTP handler: /mcp (optionally
// bearer-guarded) and GET /healthz.
func NewMux(cfg Config) *http.ServeMux {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "discord-mcp",
		Title:   "Discord Knowledge Mining",
		Version: Version,
	}, nil)
	tools.Register(server, cfg.Tools)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)

	var mcpRoute http.Handler = mcpHandler
	if cfg.AuthToken != "" {
		mcpRoute = auth.Middleware(mcpHandler, cfg.AuthToken)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpRoute)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}
