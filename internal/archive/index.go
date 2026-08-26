package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteChannelIndex rebuilds _channel.md purely from the channel's window
// frontmatter — a cache, not truth.
func WriteChannelIndex(channelDir string) error {
	metas, err := listWindowMetas(channelDir)
	if err != nil {
		return fmt.Errorf("write channel index: %w", err)
	}

	var name, id, ctype string
	total := 0
	if len(metas) > 0 {
		name, id, ctype = metas[0].ChannelName, metas[0].ChannelID, metas[0].ChannelType
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "channel_id: %q\n", id)
	fmt.Fprintf(&b, "channel_type: %s\n\n", ctype)
	b.WriteString("## Windows\n\n")
	for _, m := range metas {
		total += m.MessageCount
		status := "open"
		if m.Complete {
			status = "complete"
		}
		fmt.Fprintf(&b, "- %s: %s–%s (%d messages, %s)\n", m.Window, m.FirstMessageID, m.LastMessageID, m.MessageCount, status)
	}
	fmt.Fprintf(&b, "\ntotal messages: %d\n", total)

	return atomicWrite(filepath.Join(channelDir, "_channel.md"), []byte(b.String()))
}

// WriteGuildIndex rebuilds guild.md purely from every channel's window
// frontmatter.
func WriteGuildIndex(root string) error {
	channelsDir := filepath.Join(root, "channels")
	entries, err := os.ReadDir(channelsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("write guild index: %w", err)
		}
	}
	var dirNames []string
	for _, e := range entries {
		if e.IsDir() {
			dirNames = append(dirNames, e.Name())
		}
	}
	sort.Strings(dirNames)

	type row struct {
		name    string
		id      string
		windows int
		total   int
	}
	var rows []row
	var guildName, guildID string
	for _, dn := range dirNames {
		metas, err := listWindowMetas(filepath.Join(channelsDir, dn))
		if err != nil {
			return fmt.Errorf("write guild index: %w", err)
		}
		if len(metas) == 0 {
			continue
		}
		if guildName == "" {
			guildName, guildID = metas[0].GuildName, metas[0].GuildID
		}
		total := 0
		for _, m := range metas {
			total += m.MessageCount
		}
		rows = append(rows, row{name: metas[0].ChannelName, id: metas[0].ChannelID, windows: len(metas), total: total})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", guildName)
	fmt.Fprintf(&b, "guild_id: %q\n\n", guildID)
	b.WriteString("## Channels\n\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "- %s (%s): %d windows, %d messages\n", r.name, r.id, r.windows, r.total)
	}

	return atomicWrite(filepath.Join(root, "guild.md"), []byte(b.String()))
}
