package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behavior 12: a message with EditedTimestamp renders "*edited HH:MM*" with
// the current (post-edit) content.
func TestMessage_EditedTimestamp_RendersEditedMarker(t *testing.T) {
	t.Parallel()
	edited := time.Date(2026, 6, 3, 14, 25, 0, 0, time.UTC)
	msg := baseMessage()
	msg.EditedTimestamp = &edited

	got := render.Message(msg, emptyMentions())

	if !strings.Contains(got, "*edited 14:25*") {
		t.Errorf("Message() = %q, missing edited marker", got)
	}
	if !strings.Contains(got, msg.Content) {
		t.Errorf("Message() = %q, missing current content", got)
	}
}

// Behavior 13: reactions render as emoji-plus-count pairs; a message with no
// reactions renders no reactions segment.
func TestMessage_Reactions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		reactions []discord.Reaction
		want      []string
		notWant   string
	}{
		{
			name: "multiple reactions",
			reactions: []discord.Reaction{
				{Emoji: "👍", Count: 4},
				{Emoji: "✅", Count: 2},
			},
			want: []string{"👍4", "✅2"},
		},
		{
			name:      "no reactions",
			reactions: nil,
			notWant:   "reactions:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := baseMessage()
			msg.Reactions = tt.reactions

			got := render.Message(msg, emptyMentions())
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("Message() = %q, missing %q", got, want)
				}
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("Message() = %q, unexpected %q", got, tt.notWant)
			}
		})
	}
}

// Behavior 14: an attachment renders filename, content type, human-readable
// size, and URL as a markdown link.
func TestMessage_Attachment_RendersFilenameTypeSizeURL(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.Attachments = []discord.Attachment{
		{
			Filename:    "ruling.png",
			ContentType: "image/png",
			Size:        214 * 1024,
			URL:         "https://cdn.discordapp.com/attachments/1/2/ruling.png",
		},
	}

	got := render.Message(msg, emptyMentions())

	for _, want := range []string{"ruling.png", "image/png", "214 KB", "https://cdn.discordapp.com/attachments/1/2/ruling.png"} {
		if !strings.Contains(got, want) {
			t.Errorf("Message() = %q, missing %q", got, want)
		}
	}
}

// Behavior 15: an embed renders title, URL, and a truncated description; an
// embed with no title falls back to its URL.
func TestMessage_Embed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		embed discord.Embed
		want  []string
	}{
		{
			name: "titled embed",
			embed: discord.Embed{
				Title:       "Rules — Evolution",
				URL:         "https://omgcardgame.com/rules#evolution",
				Description: "Evolution power is the sum of all attached cards.",
			},
			want: []string{"Rules — Evolution", "https://omgcardgame.com/rules#evolution", "Evolution power is the sum"},
		},
		{
			name: "untitled embed falls back to URL",
			embed: discord.Embed{
				URL:         "https://omgcardgame.com/rules#evolution",
				Description: "Evolution power is the sum of all attached cards.",
			},
			want: []string{"https://omgcardgame.com/rules#evolution"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := baseMessage()
			msg.Embeds = []discord.Embed{tt.embed}

			got := render.Message(msg, emptyMentions())
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("Message() = %q, missing %q", got, want)
				}
			}
		})
	}
}

// Behavior 16: a sticker renders its name and URL.
func TestMessage_Sticker_RendersNameAndURL(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.Stickers = []discord.Sticker{
		{Name: "PogChamp", URL: "https://cdn.discordapp.com/stickers/1.png"},
	}

	got := render.Message(msg, emptyMentions())

	for _, want := range []string{"PogChamp", "https://cdn.discordapp.com/stickers/1.png"} {
		if !strings.Contains(got, want) {
			t.Errorf("Message() = %q, missing %q", got, want)
		}
	}
}

// Behavior 17: a message that started a thread renders the "→ thread:"
// pointer with a relative path and message count.
func TestMessage_ThreadPointer_RendersRelativeLinkAndCount(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.ThreadStarted = &discord.ThreadPointer{
		Title:        "Rest inheritance",
		Path:         "threads/rest-inheritance-1247891300000000000.md",
		MessageCount: 14,
	}

	got := render.Message(msg, emptyMentions())

	for _, want := range []string{
		"→ thread:",
		"[Rest inheritance](threads/rest-inheritance-1247891300000000000.md)",
		"(14 messages)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Message() = %q, missing %q", got, want)
		}
	}
}

// Behavior 18: system message types 7-11 render as empty output (skipped);
// type 6 (pin) and types 18/21 (thread created) render a single italic
// line.
func TestMessage_SystemMessageTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		msgType    int
		wantEmpty  bool
		wantItalic bool
	}{
		{name: "type 7 join skipped", msgType: 7, wantEmpty: true},
		{name: "type 8 boost skipped", msgType: 8, wantEmpty: true},
		{name: "type 9 boost tier 1 skipped", msgType: 9, wantEmpty: true},
		{name: "type 10 boost tier 2 skipped", msgType: 10, wantEmpty: true},
		{name: "type 11 boost tier 3 skipped", msgType: 11, wantEmpty: true},
		{name: "type 6 pin renders italic line", msgType: 6, wantItalic: true},
		{name: "type 18 thread created renders italic line", msgType: 18, wantItalic: true},
		{name: "type 21 thread starter message renders italic line", msgType: 21, wantItalic: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := baseMessage()
			msg.Type = tt.msgType

			got := render.Message(msg, emptyMentions())
			if tt.wantEmpty && got != "" {
				t.Errorf("Message() type %d = %q, want empty (skipped)", tt.msgType, got)
			}
			if tt.wantItalic {
				lines := nonBlankLines(got)
				if len(lines) != 1 {
					t.Fatalf("Message() type %d = %q, want exactly one line", tt.msgType, got)
				}
				if !strings.HasPrefix(lines[0], "*") || !strings.HasSuffix(lines[0], "*") {
					t.Errorf("Message() type %d line = %q, want italic (*...*)", tt.msgType, lines[0])
				}
			}
		})
	}
}

func nonBlankLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}
