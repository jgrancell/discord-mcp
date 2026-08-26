package render

import (
	"fmt"
	"strings"
	"time"
)

// WindowMeta is a window file's frontmatter: the resume cursor and every
// documented archive key.
type WindowMeta struct {
	GuildID        string
	GuildName      string
	ChannelID      string
	ChannelName    string
	ChannelType    string
	ParentCategory string
	Window         string
	WindowStart    time.Time
	WindowEnd      time.Time
	FirstMessageID string
	LastMessageID  string
	MessageCount   int
	Complete       bool
	MinedAt        time.Time
	MinerVersion   string
	Format         int
}

// ThreadMeta is a thread file's frontmatter.
type ThreadMeta struct {
	GuildID         string
	GuildName       string
	ParentChannelID string
	ThreadID        string
	ThreadName      string
	Archived        bool
	Locked          bool
	CreatedAt       time.Time
	OwnerID         string
	MessageCount    int
	AppliedTags     []string
	Format          int
}

const rfc3339UTC = "2006-01-02T15:04:05Z"

// WindowFrontmatter renders a window file's YAML frontmatter block.
func WindowFrontmatter(m WindowMeta) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "guild_id: %q\n", m.GuildID)
	fmt.Fprintf(&b, "guild_name: %s\n", m.GuildName)
	fmt.Fprintf(&b, "channel_id: %q\n", m.ChannelID)
	fmt.Fprintf(&b, "channel_name: %s\n", m.ChannelName)
	fmt.Fprintf(&b, "channel_type: %s\n", m.ChannelType)
	fmt.Fprintf(&b, "parent_category: %s\n", m.ParentCategory)
	fmt.Fprintf(&b, "window: %s\n", m.Window)
	fmt.Fprintf(&b, "window_start: %s\n", m.WindowStart.UTC().Format(rfc3339UTC))
	fmt.Fprintf(&b, "window_end: %s\n", m.WindowEnd.UTC().Format(rfc3339UTC))
	b.WriteString("timezone: UTC\n")
	fmt.Fprintf(&b, "first_message_id: %q\n", m.FirstMessageID)
	fmt.Fprintf(&b, "last_message_id: %q\n", m.LastMessageID)
	fmt.Fprintf(&b, "message_count: %d\n", m.MessageCount)
	fmt.Fprintf(&b, "complete: %t\n", m.Complete)
	fmt.Fprintf(&b, "mined_at: %s\n", m.MinedAt.UTC().Format(rfc3339UTC))
	fmt.Fprintf(&b, "miner_version: %s\n", m.MinerVersion)
	fmt.Fprintf(&b, "format: %d\n", m.Format)
	b.WriteString("---\n")
	return b.String()
}

// ThreadFrontmatter renders a thread file's YAML frontmatter block.
func ThreadFrontmatter(m ThreadMeta) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "guild_id: %q\n", m.GuildID)
	fmt.Fprintf(&b, "guild_name: %s\n", m.GuildName)
	fmt.Fprintf(&b, "parent_channel_id: %q\n", m.ParentChannelID)
	fmt.Fprintf(&b, "thread_id: %q\n", m.ThreadID)
	fmt.Fprintf(&b, "thread_name: %s\n", m.ThreadName)
	fmt.Fprintf(&b, "archived: %t\n", m.Archived)
	fmt.Fprintf(&b, "locked: %t\n", m.Locked)
	fmt.Fprintf(&b, "created_at: %s\n", m.CreatedAt.UTC().Format(rfc3339UTC))
	fmt.Fprintf(&b, "owner_id: %q\n", m.OwnerID)
	fmt.Fprintf(&b, "message_count: %d\n", m.MessageCount)
	b.WriteString("applied_tags:\n")
	for _, tag := range m.AppliedTags {
		fmt.Fprintf(&b, "  - %s\n", tag)
	}
	fmt.Fprintf(&b, "format: %d\n", m.Format)
	b.WriteString("---\n")
	return b.String()
}
