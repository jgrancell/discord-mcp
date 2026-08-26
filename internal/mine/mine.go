// Package mine orchestrates the guild backfill walk into the markdown
// archive.
package mine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/discord"
)

// ErrMessageContentIntentDisabled indicates the first sampled batch of
// messages carried empty content and no attachments across every message
// — the signature of a bot missing the privileged Message Content Intent,
// which Discord accepts silently rather than erroring. Aborts the run: an
// archive of blanks is worse than no archive.
var ErrMessageContentIntentDisabled = errors.New("discord: message content intent appears disabled (all sampled messages are blank) — enable it at Developer Portal → Bot → Privileged Gateway Intents → Message Content Intent")

// Config configures a mining run.
type Config struct {
	Discord   *discord.Client
	Root      string
	GuildID   string
	GuildName string

	Channel string // restrict the run to one channel ID; "" means all
	From    string // repair: override the derived cursor for Channel
	DryRun  bool   // enumerate and report, write nothing

	Now func() time.Time

	MinerVersion string
}

func (c Config) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Config) minerVersion() string {
	if c.MinerVersion != "" {
		return c.MinerVersion
	}
	return "0.1.0"
}

// DeniedChannel is a channel the bot could not read.
type DeniedChannel struct {
	ID   string
	Name string
}

// ChannelFailure records a channel the walk could not finish.
type ChannelFailure struct {
	ID    string
	Name  string
	Error string
}

// Report is the outcome of a mining run.
type Report struct {
	Start             time.Time
	End               time.Time
	ChannelsAttempted []string
	Denied            []DeniedChannel
	Failures          []ChannelFailure
	MessagesArchived  int
	ThreadsArchived   int
	WindowsOpened     int
	WindowsClosed     int

	// ThreadsCapabilityDenied records channels where the bot lacks Manage
	// Threads: private archived threads are invisible, but public threads
	// and channel messages still archive normally.
	ThreadsCapabilityDenied []DeniedChannel

	// RequestsMade, RateLimited429, and GlobalLimitHits are the run's
	// request-level counters, sourced from the Discord client's Stats
	// snapshot — the observability surface for a mode with no metrics
	// endpoint.
	RequestsMade    int
	RateLimited429  int
	GlobalLimitHits int

	// contentIntentChecked marks that the Message Content Intent startup
	// assertion has already sampled its one batch this run.
	contentIntentChecked bool
}

// Run walks the guild's channels and threads, writing the markdown
// archive. Strictly sequential, per the design's rate-limit policy.
//
// Whether Run completes normally or aborts early (context cancellation, a
// 401, or a blank-content abort), it always sets report.End, snapshots the
// Discord client's request counters into the report, and — unless
// cfg.DryRun — writes the run's report file to _runs/ before returning:
// behavior 51a applies uniformly, not just to a clean finish, so an
// aborted run leaves whatever partial report the walk produced.
func Run(ctx context.Context, cfg Config) (report Report, err error) {
	report.Start = cfg.now()
	defer func() {
		report.End = cfg.now()
		stats := cfg.Discord.Stats()
		report.RequestsMade = stats.RequestsMade
		report.RateLimited429 = stats.RateLimited429
		report.GlobalLimitHits = stats.GlobalLimitHits

		if cfg.DryRun {
			return
		}
		if writeErr := writeReportFile(cfg, report); writeErr != nil {
			err = errors.Join(err, writeErr)
		}
	}()

	channels, chErr := cfg.Discord.Channels(ctx, cfg.GuildID)
	if chErr != nil {
		err = chErr
		return report, err
	}
	activeThreads, atErr := cfg.Discord.ActiveThreads(ctx, cfg.GuildID)
	if atErr != nil {
		err = fmt.Errorf("list active threads: %w", atErr)
		return report, err
	}

	for _, ch := range channels {
		if !mineable(ch.Type) {
			continue
		}
		if cfg.Channel != "" && ch.ID != cfg.Channel {
			continue
		}
		if cerr := ctx.Err(); cerr != nil {
			err = cerr
			return report, err
		}

		report.ChannelsAttempted = append(report.ChannelsAttempted, ch.ID)
		if merr := mineChannel(ctx, cfg, ch, activeThreads, &report); merr != nil {
			// A 401 means the bot token is bad; retrying is exactly how a
			// Cloudflare IP ban is earned, so this aborts the entire run
			// immediately rather than continuing to the next channel. A
			// blank-content sample is the same class of abort: writing an
			// archive of blanks is worse than writing nothing.
			if isContextErr(merr) || errors.Is(merr, discord.ErrUnauthorized) || errors.Is(merr, ErrMessageContentIntentDisabled) {
				err = merr
				return report, err
			}
			if _, ok := asForbidden(merr); ok {
				report.Denied = append(report.Denied, DeniedChannel{ID: ch.ID, Name: ch.Name})
				continue
			}
			report.Failures = append(report.Failures, ChannelFailure{ID: ch.ID, Name: ch.Name, Error: merr.Error()})
			continue
		}
	}

	if !cfg.DryRun {
		if ierr := archive.WriteGuildIndex(cfg.Root); ierr != nil {
			err = fmt.Errorf("write guild index: %w", ierr)
			return report, err
		}
	}

	return report, nil
}

// mineable reports whether a channel type is one mine walks directly for
// messages (text, announcement, forum). Voice/category/stage channels are
// skipped.
func mineable(discordType int) bool {
	switch discordType {
	case 0, 5, 15:
		return true
	default:
		return false
	}
}
