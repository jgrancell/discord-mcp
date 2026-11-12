// Command discord-mcp is a Model Context Protocol (MCP) server that exposes a
// small set of Discord messaging tools over streamable HTTP, plus a `mine`
// subcommand that backfills a guild into a local markdown archive.
//
// It talks to Discord over REST only (it never opens a gateway websocket), so
// it holds no persistent connection. The server binds loopback by default;
// binding any non-loopback address requires DISCORD_MCP_AUTH_TOKEN, which
// guards /mcp with a static bearer challenge. All diagnostics go to stderr.
//
// Usage:
//
//	discord-mcp [serve] [--port host:port]
//	discord-mcp mine --archive DIR [--channel ID [--from ID]] [--dry-run]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/jgrancell/discord-mcp/internal/config"
	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
	"github.com/jgrancell/discord-mcp/internal/resolve"
	"github.com/jgrancell/discord-mcp/internal/serve"
	"github.com/jgrancell/discord-mcp/internal/tools"
)

// discordAPIBase is the production Discord REST root. Tests construct clients
// against their own httptest URLs instead.
const discordAPIBase = "https://discord.com/api/v10"

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("discord-mcp: fatal", "err", err)
		os.Exit(1)
	}
}

// subcommand splits args into a mode and that mode's own arguments. No
// argument, or a leading flag, means serve, so older bare `--port`
// invocations keep working; any other leading word is returned as the mode
// for run to accept or reject.
func subcommand(args []string) (mode string, rest []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "serve", args
	}
	return args[0], args[1:]
}

func run(args []string) error {
	mode, rest := subcommand(args)
	switch mode {
	case "serve":
		return runServe(rest)
	case "mine":
		return runMine(rest)
	default:
		return fmt.Errorf("unknown subcommand %q (want serve or mine)", mode)
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := fs.String("port", "", "listen address (host:port); overrides PORT (default 127.0.0.1:9091)")
	if err := fs.Parse(args); err != nil {
		return err
	}

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

func runMine(args []string) error {
	fs := flag.NewFlagSet("mine", flag.ContinueOnError)
	var flags config.MineFlags
	fs.StringVar(&flags.Archive, "archive", "", "archive root directory; overrides DISCORD_ARCHIVE_DIR")
	fs.StringVar(&flags.Channel, "channel", "", "restrict the run to one channel ID")
	fs.StringVar(&flags.From, "from", "", "repair: restart --channel at this message ID")
	fs.BoolVar(&flags.DryRun, "dry-run", false, "enumerate and report the plan, write nothing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.LoadMine(os.Getenv, flags)
	if err != nil {
		return err
	}

	// Ctrl-C cancels the walk; mine.Run still writes the partial report.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	report, err := mine.Run(ctx, mine.Config{
		Discord: discord.New(discordAPIBase, cfg.BotToken),
		Root:    cfg.Root,
		GuildID: cfg.GuildID,
		Channel: cfg.Channel,
		From:    cfg.From,
		DryRun:  cfg.DryRun,
	})
	slog.Info("discord-mcp mine finished",
		"dry_run", cfg.DryRun,
		"channels", len(report.ChannelsAttempted),
		"denied", len(report.Denied),
		"failures", len(report.Failures),
		"messages", report.MessagesArchived,
		"threads", report.ThreadsArchived,
		"requests", report.RequestsMade,
		"rate_limited", report.RateLimited429,
		"duration", report.End.Sub(report.Start).Round(1e6),
	)
	return err
}
