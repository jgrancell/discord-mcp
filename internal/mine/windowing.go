package mine

import (
	"fmt"
	"os"
	"time"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// windowAccumulator groups an ascending message stream into calendar-month
// windows, closing (complete: true) each window as soon as a message from
// a later month arrives, per the design's windowing rule.
type windowAccumulator struct {
	cfg        Config
	channelDir string
	ch         discord.Channel
	pointers   map[string]discord.ThreadPointer
	report     *Report

	currentKey  string
	currentMsgs []discord.Message
}

func newWindowAccumulator(cfg Config, channelDir string, ch discord.Channel, pointers map[string]discord.ThreadPointer, report *Report) *windowAccumulator {
	return &windowAccumulator{cfg: cfg, channelDir: channelDir, ch: ch, pointers: pointers, report: report}
}

// add processes one ascending batch from WalkMessages.
func (a *windowAccumulator) add(batch []discord.Message) error {
	if err := checkMessageContentIntent(a.report, batch); err != nil {
		return err
	}
	for _, m := range batch {
		key := m.Timestamp.UTC().Format("2006-01")
		if a.currentKey != "" && key != a.currentKey {
			if err := a.closeCurrent(); err != nil {
				return err
			}
		}
		a.currentKey = key
		if ptr, ok := a.pointers[m.ID]; ok {
			p := ptr
			m.ThreadStarted = &p
		}
		a.currentMsgs = append(a.currentMsgs, m)
	}
	return nil
}

func (a *windowAccumulator) closeCurrent() error {
	if len(a.currentMsgs) == 0 {
		return nil
	}
	if err := a.writeWindow(true); err != nil {
		return err
	}
	a.report.WindowsClosed++
	a.currentMsgs = nil
	return nil
}

// flush writes the final, still-open window after the walk completes.
func (a *windowAccumulator) flush() error {
	if len(a.currentMsgs) == 0 {
		return nil
	}
	return a.writeWindow(false)
}

func (a *windowAccumulator) writeWindow(complete bool) error {
	if a.cfg.DryRun {
		return nil
	}
	meta := render.WindowMeta{
		GuildID: a.cfg.GuildID, GuildName: a.cfg.GuildName,
		ChannelID: a.ch.ID, ChannelName: a.ch.Name, ChannelType: discord.ChannelTypeName(a.ch.Type),
		Window:         a.currentKey,
		WindowStart:    monthStart(a.currentKey),
		WindowEnd:      monthStart(a.currentKey).AddDate(0, 1, 0),
		FirstMessageID: a.currentMsgs[0].ID,
		LastMessageID:  discord.MaxMessageID(a.currentMsgs),
		MessageCount:   len(a.currentMsgs),
		Complete:       complete,
		MinedAt:        a.cfg.now(),
		MinerVersion:   a.cfg.minerVersion(),
		Format:         1,
	}
	if err := os.MkdirAll(a.channelDir, 0o755); err != nil {
		return fmt.Errorf("mkdir channel dir: %w", err)
	}
	body := render.Messages(a.currentMsgs, render.MentionMap{})
	wrote, err := archive.WriteWindow(archive.WindowPath(a.channelDir, a.currentKey), meta, body)
	if err != nil {
		return fmt.Errorf("write window %s: %w", a.currentKey, err)
	}
	if wrote {
		a.report.WindowsOpened++
		a.report.MessagesArchived += len(a.currentMsgs)
	}
	return nil
}

func monthStart(key string) time.Time {
	t, err := time.Parse("2006-01", key)
	if err != nil {
		return time.Time{}
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
