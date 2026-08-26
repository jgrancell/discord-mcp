// Command discord-mcp is a Model Context Protocol (MCP) server that exposes a
// small set of Discord messaging tools over streamable HTTP.
//
// It talks to Discord over REST only (it never opens a gateway websocket), so
// the bot requires no privileged intents and holds no persistent connection.
// The server binds loopback by default; binding any non-loopback address
// requires DISCORD_MCP_AUTH_TOKEN, which guards /mcp with a static bearer
// challenge. All diagnostics go to stderr.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/jgrancell/discord-mcp/internal/config"
	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/resolve"
	"github.com/jgrancell/discord-mcp/internal/serve"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

// discordAPIBase is the production Discord REST root. Tests construct clients
// against their own httptest URLs instead.
const discordAPIBase = "https://discord.com/api/v10"

func main() {
	if err := run(); err != nil {
		slog.Error("discord-mcp: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	port := flag.String("port", "", "listen address (host:port); overrides PORT (default 127.0.0.1:9091)")
	flag.Parse()

	cfg, err := config.LoadServe(os.Getenv, *port)
	if err != nil {
		return err
	}

	client := discord.New(discordAPIBase, cfg.BotToken)
	deps := tools.Deps{
		Discord: client,
		Resolve: resolve.New(client),
		GuildID: cfg.GuildID,
	}
	mux := serve.NewMux(serve.Config{Tools: deps, AuthToken: cfg.AuthToken})

	slog.Info("discord-mcp serving", cfg.SafeLogAttrs()...)
	return http.ListenAndServe(cfg.Port, mux)
}
