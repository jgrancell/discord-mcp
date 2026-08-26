package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behavior 1: a plain message renders an H3 with UTC HH:MM, display name,
// username in parentheses, the message ID, and a jump URL.
func TestMessage_Plain_RendersH3Heading(t *testing.T) {
	t.Parallel()
	got := render.Message(baseMessage(), emptyMentions())

	wantFragments := []string{
		"### 14:22",
		"Janak (janak)",
		"[#1247891234567890123]",
		"https://discord.com/channels/1100000000000000000/1122334455667788/1247891234567890123",
	}
	for _, want := range wantFragments {
		if !strings.Contains(got, want) {
			t.Errorf("Message() = %q, missing %q", got, want)
		}
	}
}

// Behavior 2: a bot account renders [bot] after the username; a webhook
// message renders [webhook].
func TestMessage_BotAndWebhookAuthors_RenderTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		bot  bool
		hook bool
		want string
	}{
		{name: "bot account", bot: true, want: "Janak (janak) [bot]"},
		{name: "webhook", hook: true, want: "Janak (janak) [webhook]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := baseMessage()
			msg.Author.Bot = tt.bot
			msg.Author.Webhook = tt.hook

			got := render.Message(msg, emptyMentions())
			if !strings.Contains(got, tt.want) {
				t.Errorf("Message() = %q, missing %q", got, tt.want)
			}
		})
	}
}

// Behavior 3: a message with ReferencedMessage renders the "↩ replying to"
// blockquote with the parent's author, time, and content truncated to 100
// characters with an ellipsis.
func TestMessage_WithReferencedMessage_RendersReplyBlockquote(t *testing.T) {
	t.Parallel()
	longContent := strings.Repeat("a", 150)
	msg := baseMessage()
	msg.ReferencedMessage = &discord.Message{
		Author:    discord.Author{Username: "someuser", DisplayName: "someuser"},
		Timestamp: baseMessage().Timestamp.Add(-2 * time.Minute), // 14:20
		Content:   longContent,
	}

	got := render.Message(msg, emptyMentions())

	if !strings.Contains(got, "> ↩ replying to someuser 14:20:") {
		t.Fatalf("Message() = %q, missing reply blockquote prefix", got)
	}
	truncated := longContent[:100]
	if !strings.Contains(got, truncated) {
		t.Errorf("Message() = %q, missing truncated parent content %q", got, truncated)
	}
	if strings.Contains(got, longContent) {
		t.Errorf("Message() = %q, parent content was not truncated", got)
	}
}

// Behavior 4: a reply whose parent is absent from the payload renders the
// message without a reply line, and does not error.
func TestMessage_ReplyParentAbsent_RendersWithoutReplyLineNoError(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.ReferencedMessage = nil // absent: nothing to render

	got := render.Message(msg, emptyMentions())

	if strings.Contains(got, "replying to") {
		t.Errorf("Message() = %q, unexpected reply line with no ReferencedMessage", got)
	}
	if !strings.Contains(got, "Rested state inherits up the stack on evolution.") {
		t.Errorf("Message() = %q, missing body", got)
	}
}
