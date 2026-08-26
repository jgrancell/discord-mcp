// Package config loads server and mine configuration from environment
// variables and flags, failing fast on missing required values.
package config

import (
	"fmt"
	"net"
)

// ServeConfig is serve mode's fully-validated configuration.
type ServeConfig struct {
	BotToken  string
	GuildID   string
	Port      string
	AuthToken string
}

// LoadServe reads serve configuration via getenv, with portFlag (from
// --port) overriding PORT when non-empty. Required values missing, or a
// non-loopback Port with no DISCORD_MCP_AUTH_TOKEN, is a returned error
// naming the variable — the caller logs and exits non-zero before serving.
func LoadServe(getenv func(string) string, portFlag string) (ServeConfig, error) {
	cfg := ServeConfig{
		BotToken:  getenv("DISCORD_BOT_TOKEN"),
		GuildID:   getenv("DISCORD_GUILD_ID"),
		AuthToken: getenv("DISCORD_MCP_AUTH_TOKEN"),
	}
	if cfg.BotToken == "" {
		return ServeConfig{}, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}
	if cfg.GuildID == "" {
		return ServeConfig{}, fmt.Errorf("DISCORD_GUILD_ID is required")
	}

	cfg.Port = portFlag
	if cfg.Port == "" {
		cfg.Port = getenv("PORT")
	}
	if cfg.Port == "" {
		cfg.Port = "127.0.0.1:9091"
	}

	if !isLoopback(cfg.Port) && cfg.AuthToken == "" {
		return ServeConfig{}, fmt.Errorf("PORT %q is not loopback-only; DISCORD_MCP_AUTH_TOKEN is required to bind it", cfg.Port)
	}
	return cfg, nil
}

// isLoopback reports whether addr (host:port) binds only the loopback
// interface.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" {
		return false // ":9091" — all interfaces
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// SafeLogAttrs returns slog key/value pairs describing this config for
// startup logging — presence and length of secrets, never their value.
func (c ServeConfig) SafeLogAttrs() []any {
	return []any{
		"bot_token_present", c.BotToken != "",
		"bot_token_length", len(c.BotToken),
		"guild_id", c.GuildID,
		"port", c.Port,
		"auth_token_present", c.AuthToken != "",
	}
}
