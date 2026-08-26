package config_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/config"
)

// envMap builds a getenv func over a fixed set of variables, mirroring how
// LoadServe is expected to read configuration (env with flag overrides,
// per mcp-authoring's "Errors, results, config").
func envMap(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func baseValidEnv() map[string]string {
	return map[string]string{
		"DISCORD_BOT_TOKEN":      "s3cr3t-bot-token",
		"DISCORD_GUILD_ID":       "1100000000000000000",
		"PORT":                   "127.0.0.1:9091",
		"DISCORD_MCP_AUTH_TOKEN": "",
	}
}

// Behavior 72: a non-loopback PORT with no DISCORD_MCP_AUTH_TOKEN fails
// validation, naming the variable — the fail-closed rule for the inbound
// gate.
func TestLoadServe_NonLoopbackPortNoAuthToken_FailsNamingTheVariable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		port string
	}{
		{name: "all interfaces shorthand", port: ":9091"},
		{name: "explicit all interfaces", port: "0.0.0.0:9091"},
		{name: "a LAN address", port: "192.168.1.50:9091"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := baseValidEnv()
			env["PORT"] = tt.port
			env["DISCORD_MCP_AUTH_TOKEN"] = ""

			_, err := config.LoadServe(envMap(env), "")
			if err == nil {
				t.Fatalf("LoadServe(port=%s, no auth token): want an error, got nil", tt.port)
			}
			if !strings.Contains(err.Error(), "DISCORD_MCP_AUTH_TOKEN") {
				t.Errorf("err = %q, want it to name DISCORD_MCP_AUTH_TOKEN", err.Error())
			}
		})
	}
}

// A loopback PORT with no auth token is fine (the default, documented
// trust model) — guards the positive case for behavior 72's rule.
func TestLoadServe_LoopbackPortNoAuthToken_Succeeds(t *testing.T) {
	t.Parallel()
	env := baseValidEnv()
	env["PORT"] = "127.0.0.1:9091"
	env["DISCORD_MCP_AUTH_TOKEN"] = ""

	if _, err := config.LoadServe(envMap(env), ""); err != nil {
		t.Fatalf("LoadServe(loopback, no auth token): %v, want success", err)
	}
}

// Behavior 76: missing DISCORD_BOT_TOKEN or DISCORD_GUILD_ID exits
// non-zero at startup (surfaced here as a validation error) naming the
// variable.
func TestLoadServe_MissingRequiredVars_ErrorsNamingTheVariable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		missing string
	}{
		{name: "missing bot token", missing: "DISCORD_BOT_TOKEN"},
		{name: "missing guild id", missing: "DISCORD_GUILD_ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := baseValidEnv()
			delete(env, tt.missing)

			_, err := config.LoadServe(envMap(env), "")
			if err == nil {
				t.Fatalf("LoadServe with %s missing: want an error, got nil", tt.missing)
			}
			if !strings.Contains(err.Error(), tt.missing) {
				t.Errorf("err = %q, want it to name %s", err.Error(), tt.missing)
			}
		})
	}
}

// Behavior 77: startup logs never contain the bot token value — only its
// presence and length.
func TestServe_SafeLogAttrs_NeverContainTheTokenValue(t *testing.T) {
	t.Parallel()
	env := baseValidEnv()
	const token = "s3cr3t-bot-token"
	env["DISCORD_BOT_TOKEN"] = token

	cfg, err := config.LoadServe(envMap(env), "")
	if err != nil {
		t.Fatalf("LoadServe: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("starting", cfg.SafeLogAttrs()...)

	out := buf.String()
	if strings.Contains(out, token) {
		t.Fatalf("startup log contains the raw bot token:\n%s", out)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode log line: %v\n%s", err, out)
	}
	if _, ok := decoded["bot_token_present"]; !ok {
		t.Errorf("log line %v missing bot_token_present", decoded)
	}
}
