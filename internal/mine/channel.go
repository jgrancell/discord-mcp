package mine

import (
	"context"
	"fmt"
	"os"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/discord"
)

// mineChannel mines one channel: its threads, and — unless it is a forum
// (whose "messages" are entirely its threads) — its windowed message log.
func mineChannel(ctx context.Context, cfg Config, ch discord.Channel, activeThreads []discord.Thread, report *Report) error {
	slug := archive.Slugify(ch.Name)
	channelDir, err := archive.ResolveChannelDir(cfg.Root, ch.ID, slug)
	if err != nil {
		return fmt.Errorf("resolve channel dir: %w", err)
	}

	threads, err := allChannelThreads(ctx, cfg, ch, activeThreads, report)
	if err != nil {
		return err
	}
	pointers := threadPointers(threads)

	if !cfg.DryRun && len(threads) > 0 {
		if err := mineThreads(ctx, cfg, channelDir, ch, threads, report); err != nil {
			return err
		}
	}

	if ch.Type == 15 { // forum: no window messages, only threads
		return finishChannel(cfg, channelDir)
	}

	cursor := cfg.From
	if cursor == "" || cfg.Channel != ch.ID {
		cursor, err = archive.ResumeCursor(channelDir)
		if err != nil {
			return fmt.Errorf("resume cursor: %w", err)
		}
	}

	acc := newWindowAccumulator(cfg, channelDir, ch, pointers, report)
	if err := cfg.Discord.WalkMessages(ctx, ch.ID, cursor, 100, acc.add); err != nil {
		return err
	}
	if err := acc.flush(); err != nil {
		return err
	}

	return finishChannel(cfg, channelDir)
}

func finishChannel(cfg Config, channelDir string) error {
	if cfg.DryRun {
		return nil
	}
	if err := os.MkdirAll(channelDir, 0o755); err != nil {
		return fmt.Errorf("mkdir channel dir: %w", err)
	}
	if err := archive.WriteChannelIndex(channelDir); err != nil {
		return fmt.Errorf("write channel index: %w", err)
	}
	return nil
}
