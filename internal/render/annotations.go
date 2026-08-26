package render

import (
	"fmt"
	"strings"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

const embedDescriptionTruncateLen = 200

// escapeLeadingHash backslash-escapes any line that starts with "#" so it
// cannot forge a markdown heading.
func escapeLeadingHash(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			lines[i] = "\\" + line
		}
	}
	return strings.Join(lines, "\n")
}

// fenceWrap wraps s in a backtick fence one longer than the longest
// fence-delimiter line s already contains, so an embedded fence cannot
// terminate the surrounding document structure. Content with no fence is
// returned unchanged.
func fenceWrap(s string) string {
	longest := 0
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if len(t) < 3 || strings.Trim(t, "`") != "" {
			continue
		}
		if len(t) > longest {
			longest = len(t)
		}
	}
	if longest == 0 {
		return s
	}
	fence := strings.Repeat("`", longest+1)
	return fence + "\n" + s + "\n" + fence
}

// annotationBlock joins the edited/reactions line, attachments, embeds,
// stickers, and thread pointer into one paragraph, in that order.
func annotationBlock(msg discord.Message) string {
	var lines []string
	if l := editedReactionsLine(msg); l != "" {
		lines = append(lines, l)
	}
	if l := attachmentsLine(msg.Attachments); l != "" {
		lines = append(lines, l)
	}
	for _, e := range msg.Embeds {
		lines = append(lines, embedLine(e))
	}
	for _, s := range msg.Stickers {
		lines = append(lines, stickerLine(s))
	}
	if msg.ThreadStarted != nil {
		lines = append(lines, threadPointerLine(*msg.ThreadStarted))
	}
	return strings.Join(lines, "\n")
}

func editedReactionsLine(msg discord.Message) string {
	var parts []string
	if msg.EditedTimestamp != nil {
		parts = append(parts, fmt.Sprintf("*edited %s*", msg.EditedTimestamp.UTC().Format("15:04")))
	}
	if len(msg.Reactions) > 0 {
		rs := make([]string, len(msg.Reactions))
		for i, r := range msg.Reactions {
			rs[i] = fmt.Sprintf("%s%d", r.Emoji, r.Count)
		}
		parts = append(parts, "reactions: "+strings.Join(rs, " "))
	}
	return strings.Join(parts, " · ")
}

func attachmentsLine(atts []discord.Attachment) string {
	if len(atts) == 0 {
		return ""
	}
	parts := make([]string, len(atts))
	for i, a := range atts {
		parts[i] = fmt.Sprintf("[%s](%s) (%s, %s)", a.Filename, a.URL, a.ContentType, humanSize(a.Size))
	}
	return "attachments: " + strings.Join(parts, ", ")
}

func embedLine(e discord.Embed) string {
	title := e.Title
	if title == "" {
		title = e.URL
	}
	line := fmt.Sprintf("embed: [%s](%s)", title, e.URL)
	if e.Description != "" {
		line += fmt.Sprintf(" — %q", truncateRunes(e.Description, embedDescriptionTruncateLen))
	}
	return line
}

func stickerLine(s discord.Sticker) string {
	return fmt.Sprintf("sticker: [%s](%s)", s.Name, s.URL)
}

func threadPointerLine(tp discord.ThreadPointer) string {
	return fmt.Sprintf("→ thread: [%s](%s) (%d messages)", tp.Title, tp.Path, tp.MessageCount)
}

func humanSize(n int64) string {
	const kb, mb, gb = 1024, 1024 * 1024, 1024 * 1024 * 1024
	switch {
	case n >= gb:
		return formatUnit(n, gb, "GB")
	case n >= mb:
		return formatUnit(n, mb, "MB")
	case n >= kb:
		return formatUnit(n, kb, "KB")
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func formatUnit(n, unit int64, suffix string) string {
	if n%unit == 0 {
		return fmt.Sprintf("%d %s", n/unit, suffix)
	}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(unit), suffix)
}

func isSkippedSystemType(t int) bool {
	return t >= 7 && t <= 11
}

func isAnnotatedSystemType(t int) bool {
	switch t {
	case 6, 18, 21:
		return true
	}
	return false
}

func systemLine(msg discord.Message) string {
	switch msg.Type {
	case 6:
		return fmt.Sprintf("*%s pinned a message*", authorName(msg.Author))
	case 18, 21:
		return fmt.Sprintf("*%s started a thread*", authorName(msg.Author))
	}
	return ""
}
