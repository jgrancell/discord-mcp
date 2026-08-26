package tools_test

import (
	"strings"
	"testing"
)

// Behavior 67: a channel ID belonging to a different guild returns a tool
// error and no content — asserted for every tool that takes a channel or
// thread ID.
func TestTools_ForeignGuildChannelID_ErrorsForEveryToolTakingAnID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tool string
		args map[string]any
	}{
		{tool: "list_threads", args: map[string]any{"channel_id": otherGuildChannel}},
		{tool: "get_messages", args: map[string]any{"channel_id": otherGuildChannel}},
		{tool: "get_context", args: map[string]any{"channel_id": otherGuildChannel, "message_id": "1"}},
		{tool: "get_thread", args: map[string]any{"thread_id": otherGuildChannel}},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()
			out := callError(t, session(t), tt.tool, tt.args)
			if !strings.Contains(strings.ToLower(out), "guild") {
				t.Errorf("%s foreign-guild error does not mention the guild mismatch:\n%s", tt.tool, out)
			}
		})
	}
}
