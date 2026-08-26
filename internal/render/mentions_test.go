package render_test

import (
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behaviors 5-8: <@id>, <#id>, <@&id>, and emoji shortcodes resolve through
// the MentionMap; an unresolvable ID is left verbatim, not dropped.
func TestMessage_MentionResolution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  string
		mentions render.MentionMap
		want     string
		notWant  string
	}{
		{
			name:    "user mention resolved",
			content: "ask <@123> about it",
			mentions: render.MentionMap{
				Users: map[string]string{"123": "someuser"},
			},
			want: "ask @someuser about it",
		},
		{
			name:    "channel mention resolved",
			content: "see <#456>",
			mentions: render.MentionMap{
				Channels: map[string]string{"456": "rules-questions"},
			},
			want: "see #rules-questions",
		},
		{
			name:    "role mention resolved",
			content: "ping <@&789>",
			mentions: render.MentionMap{
				Roles: map[string]string{"789": "Judges"},
			},
			want: "ping @Judges",
		},
		{
			name:     "unresolvable user mention left verbatim",
			content:  "who is <@999>?",
			mentions: render.MentionMap{},
			want:     "<@999>",
			notWant:  "@<nil>",
		},
		{
			name:    "static custom emoji shortcode",
			content: "nice <:pog:123>",
			want:    "nice :pog:",
		},
		{
			name:    "animated custom emoji shortcode",
			content: "nice <a:pog:123>",
			want:    "nice :pog:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := baseMessage()
			msg.Content = tt.content

			got := render.Message(msg, tt.mentions)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Message() = %q, want substring %q", got, tt.want)
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("Message() = %q, unexpected substring %q", got, tt.notWant)
			}
		})
	}
}
