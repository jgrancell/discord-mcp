package config_test

import (
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/config"
)

func mineEnv() map[string]string {
	return map[string]string{
		"DISCORD_BOT_TOKEN":   "s3cr3t-bot-token",
		"DISCORD_GUILD_ID":    "1100000000000000000",
		"DISCORD_ARCHIVE_DIR": "/srv/archive",
	}
}

func TestLoadMine_MissingRequiredValue_FailsNamingIt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		unset string
		want  string
	}{
		{name: "bot token", unset: "DISCORD_BOT_TOKEN", want: "DISCORD_BOT_TOKEN"},
		{name: "guild", unset: "DISCORD_GUILD_ID", want: "DISCORD_GUILD_ID"},
		{name: "archive root", unset: "DISCORD_ARCHIVE_DIR", want: "--archive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := mineEnv()
			delete(env, tt.unset)
			_, err := config.LoadMine(envMap(env), config.MineFlags{})
			if err == nil {
				t.Fatalf("LoadMine with %s unset: got nil error, want one naming %s", tt.unset, tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not name %s", err, tt.want)
			}
		})
	}
}

func TestLoadMine_ArchiveFlag_OverridesEnv(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadMine(envMap(mineEnv()), config.MineFlags{Archive: "/tmp/elsewhere"})
	if err != nil {
		t.Fatalf("LoadMine: %v", err)
	}
	if cfg.Root != "/tmp/elsewhere" {
		t.Errorf("Root = %q, want the --archive flag value %q", cfg.Root, "/tmp/elsewhere")
	}
}

func TestLoadMine_ArchiveFromEnv_WhenNoFlag(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadMine(envMap(mineEnv()), config.MineFlags{})
	if err != nil {
		t.Fatalf("LoadMine: %v", err)
	}
	if cfg.Root != "/srv/archive" {
		t.Errorf("Root = %q, want DISCORD_ARCHIVE_DIR %q", cfg.Root, "/srv/archive")
	}
}

// --from repairs one channel's cursor; without --channel it would apply to
// every channel at once, which is never what a repair means.
func TestLoadMine_FromWithoutChannel_Fails(t *testing.T) {
	t.Parallel()
	_, err := config.LoadMine(envMap(mineEnv()), config.MineFlags{From: "1200000000000000000"})
	if err == nil {
		t.Fatal("LoadMine with --from and no --channel: got nil error")
	}
	if !strings.Contains(err.Error(), "--channel") {
		t.Errorf("error %q does not name --channel", err)
	}
}

func TestLoadMine_CarriesRunFlags(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadMine(envMap(mineEnv()), config.MineFlags{
		Channel: "1300000000000000000",
		From:    "1200000000000000000",
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("LoadMine: %v", err)
	}
	if cfg.Channel != "1300000000000000000" || cfg.From != "1200000000000000000" || !cfg.DryRun {
		t.Errorf("got Channel=%q From=%q DryRun=%v, want the flag values", cfg.Channel, cfg.From, cfg.DryRun)
	}
	if cfg.BotToken != "s3cr3t-bot-token" || cfg.GuildID != "1100000000000000000" {
		t.Errorf("env values not carried: BotToken set=%v GuildID=%q", cfg.BotToken != "", cfg.GuildID)
	}
}
