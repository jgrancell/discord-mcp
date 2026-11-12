package config

import "fmt"

// MineFlags are the mine subcommand's command-line values.
type MineFlags struct {
	Archive string // --archive; overrides DISCORD_ARCHIVE_DIR
	Channel string // --channel; restrict the run to one channel ID
	From    string // --from; repair one channel's cursor (needs Channel)
	DryRun  bool   // --dry-run; enumerate and report, write nothing
}

// MineConfig is mine mode's fully-validated configuration.
type MineConfig struct {
	BotToken string
	GuildID  string
	Root     string
	Channel  string
	From     string
	DryRun   bool
}

// LoadMine reads mine configuration via getenv plus flags. The archive root
// has no default: a backfill writes identifiable community chat to disk, so
// where it lands is always stated, never assumed.
func LoadMine(getenv func(string) string, flags MineFlags) (MineConfig, error) {
	cfg := MineConfig{
		BotToken: getenv("DISCORD_BOT_TOKEN"),
		GuildID:  getenv("DISCORD_GUILD_ID"),
		Root:     flags.Archive,
		Channel:  flags.Channel,
		From:     flags.From,
		DryRun:   flags.DryRun,
	}
	if cfg.Root == "" {
		cfg.Root = getenv("DISCORD_ARCHIVE_DIR")
	}
	if cfg.BotToken == "" {
		return MineConfig{}, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}
	if cfg.GuildID == "" {
		return MineConfig{}, fmt.Errorf("DISCORD_GUILD_ID is required")
	}
	if cfg.Root == "" {
		return MineConfig{}, fmt.Errorf("an archive directory is required: pass --archive or set DISCORD_ARCHIVE_DIR")
	}
	if cfg.From != "" && cfg.Channel == "" {
		return MineConfig{}, fmt.Errorf("--from repairs a single channel's cursor and requires --channel")
	}
	return cfg, nil
}
