package mine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// allChannelThreads returns every thread belonging to ch: active threads
// (pre-fetched once per run at the guild level) filtered to this channel,
// plus its archived public and private threads. A 403 on public archived
// threads is a capability gap, not a failure — it is swallowed here; a
// fully denied channel still surfaces via its /messages call. A 403 on
// private archived threads (missing Manage Threads) is recorded once per
// channel in report.ThreadsCapabilityDenied; public threads still archive.
// Any other error propagates — including a 401, which must abort the run
// rather than being treated as a per-channel gap.
func allChannelThreads(ctx context.Context, cfg Config, ch discord.Channel, activeThreads []discord.Thread, report *Report) ([]discord.Thread, error) {
	var out []discord.Thread
	for _, t := range activeThreads {
		if t.ParentID == ch.ID {
			out = append(out, t)
		}
	}

	pub, err := cfg.Discord.ArchivedThreads(ctx, ch.ID, false)
	if err != nil {
		if _, ok := asForbidden(err); ok {
			return out, nil
		}
		return nil, fmt.Errorf("list archived public threads for %s: %w", ch.ID, err)
	}
	out = append(out, pub...)

	priv, err := cfg.Discord.ArchivedThreads(ctx, ch.ID, true)
	if err != nil {
		if _, ok := asForbidden(err); ok {
			report.ThreadsCapabilityDenied = append(report.ThreadsCapabilityDenied, DeniedChannel{ID: ch.ID, Name: ch.Name})
			return out, nil
		}
		return nil, fmt.Errorf("list archived private threads for %s: %w", ch.ID, err)
	}
	out = append(out, priv...)
	return out, nil
}

// threadPointers maps each thread's origin message ID to the pointer line
// mined into its parent's window file.
func threadPointers(threads []discord.Thread) map[string]discord.ThreadPointer {
	out := make(map[string]discord.ThreadPointer, len(threads))
	for _, t := range threads {
		slug := archive.Slugify(t.Name)
		out[t.ID] = discord.ThreadPointer{
			Title:        t.Name,
			Path:         "threads/" + slug + "-" + t.ID + ".md",
			MessageCount: t.MessageCount,
		}
	}
	return out
}

// mineThreads walks and writes every thread's own message history into
// channelDir/threads/.
func mineThreads(ctx context.Context, cfg Config, channelDir string, ch discord.Channel, threads []discord.Thread, report *Report) error {
	threadsDir := filepath.Join(channelDir, "threads")
	if err := os.MkdirAll(threadsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir threads dir: %w", err)
	}

	for _, th := range threads {
		var msgs []discord.Message
		err := cfg.Discord.WalkMessages(ctx, th.ID, "0", 100, func(batch []discord.Message) error {
			if err := checkMessageContentIntent(report, batch); err != nil {
				return err
			}
			msgs = append(msgs, batch...)
			return nil
		})
		if err != nil {
			if _, ok := asForbidden(err); ok {
				continue
			}
			return fmt.Errorf("mine thread %s: %w", th.ID, err)
		}

		meta := render.ThreadMeta{
			GuildID: cfg.GuildID, GuildName: cfg.GuildName,
			ParentChannelID: ch.ID, ThreadID: th.ID, ThreadName: th.Name,
			Archived: th.Archived, Locked: th.Locked,
			CreatedAt:    firstTimestamp(msgs, cfg.now()),
			OwnerID:      th.OwnerID,
			MessageCount: len(msgs),
			AppliedTags:  th.AppliedTags,
			Format:       1,
		}
		body := render.Messages(msgs, render.MentionMap{})
		path := archive.ThreadPath(channelDir, archive.Slugify(th.Name), th.ID)
		if err := archive.WriteThread(path, meta, body); err != nil {
			return fmt.Errorf("write thread %s: %w", th.ID, err)
		}
		report.ThreadsArchived++
	}
	return nil
}

func firstTimestamp(msgs []discord.Message, fallback time.Time) time.Time {
	if len(msgs) == 0 {
		return fallback
	}
	return msgs[0].Timestamp
}
