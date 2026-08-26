package render

import (
	"fmt"
	"strings"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

const replyTruncateLen = 100

// Message renders a single Discord message into its markdown block. System
// message types 7-11 render as "" (skipped); 6, 18, and 21 render a single
// italic line.
func Message(msg discord.Message, mentions MentionMap) string {
	if isSkippedSystemType(msg.Type) {
		return ""
	}
	if isAnnotatedSystemType(msg.Type) {
		return systemLine(msg)
	}

	paras := []string{heading(msg)}
	if r := replyBlockquote(msg.ReferencedMessage); r != "" {
		paras = append(paras, r)
	}
	paras = append(paras, renderBody(msg.Content, mentions))
	if ann := annotationBlock(msg); ann != "" {
		paras = append(paras, ann)
	}
	return strings.Join(paras, "\n\n")
}

// Messages renders a sequence of messages, grouping consecutive messages on
// the same UTC day under one H2 day heading.
func Messages(msgs []discord.Message, mentions MentionMap) string {
	var b strings.Builder
	currentDay := ""
	wroteAny := false
	for _, m := range msgs {
		rendered := Message(m, mentions)
		if rendered == "" {
			continue
		}
		day := m.Timestamp.UTC().Format("2006-01-02")
		if wroteAny {
			b.WriteString("\n\n")
		}
		if day != currentDay {
			fmt.Fprintf(&b, "## %s\n\n", day)
			currentDay = day
		}
		b.WriteString(rendered)
		wroteAny = true
	}
	return b.String()
}

func heading(msg discord.Message) string {
	hhmm := msg.Timestamp.UTC().Format("15:04")
	jump := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", msg.GuildID, msg.ChannelID, msg.ID)
	return fmt.Sprintf("### %s — %s [#%s](%s)", hhmm, authorTag(msg.Author), msg.ID, jump)
}

// authorTag renders "Display (username)" plus a [bot]/[webhook] suffix.
func authorTag(a discord.Author) string {
	s := fmt.Sprintf("%s (%s)", authorName(a), a.Username)
	switch {
	case a.Webhook:
		s += " [webhook]"
	case a.Bot:
		s += " [bot]"
	}
	return s
}

func authorName(a discord.Author) string {
	if a.DisplayName != "" {
		return a.DisplayName
	}
	return a.Username
}

func replyBlockquote(parent *discord.Message) string {
	if parent == nil {
		return ""
	}
	hhmm := parent.Timestamp.UTC().Format("15:04")
	return fmt.Sprintf("> ↩ replying to %s %s: \"%s\"", authorName(parent.Author), hhmm, truncateRunes(parent.Content, replyTruncateLen))
}

func renderBody(content string, mentions MentionMap) string {
	content = resolveMentions(content, mentions)
	content = escapeLeadingHash(content)
	content = fenceWrap(content)
	return content
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
