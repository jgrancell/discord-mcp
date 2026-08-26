package render_test

import (
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behavior 9: message content beginning with # renders with the hash
// escaped, so the output parses with the intended heading structure intact
// (an unescaped leading # would forge a spurious Markdown heading).
func TestMessage_LeadingHash_IsEscaped(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.Content = "# does this forge a heading?"

	got := render.Message(msg, emptyMentions())

	if !strings.Contains(got, `\# does this forge a heading?`) {
		t.Errorf("Message() = %q, leading # not escaped", got)
	}
	// The only real headings in a rendered message are the H3 message
	// heading itself; the body must not introduce another line starting
	// with an unescaped #.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "# ") {
			t.Errorf("Message() produced an unescaped top-level heading: %q", line)
		}
	}
}

// Behavior 10: a triple-backtick fence in user content is re-fenced with a
// longer fence than any fence the content contains, so it cannot terminate
// the surrounding document structure.
func TestMessage_TripleBacktickFence_RenderedInsideLongerFence(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.Content = "```go\nfmt.Println(\"hi\")\n```"

	got := render.Message(msg, emptyMentions())

	if !containsFenceLongerThan(got, 3) {
		t.Errorf("Message() = %q, no fence longer than the content's 3-backtick fence", got)
	}
	if !strings.Contains(got, `fmt.Println("hi")`) {
		t.Errorf("Message() = %q, fenced content missing", got)
	}
}

// Behavior 11: content already containing a longer fence (4 backticks) gets
// wrapped in a fence longer still (5+).
func TestMessage_NestedLongerFence_GetsFenceLongerStill(t *testing.T) {
	t.Parallel()
	msg := baseMessage()
	msg.Content = "````\nnested fenced content\n````"

	got := render.Message(msg, emptyMentions())

	if !containsFenceLongerThan(got, 4) {
		t.Errorf("Message() = %q, no fence longer than the content's 4-backtick fence", got)
	}
}

// containsFenceLongerThan reports whether s contains a line consisting
// solely of a backtick run longer than n.
func containsFenceLongerThan(s string, n int) bool {
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Trim(trimmed, "`") != "" {
			continue
		}
		if len(trimmed) > n {
			return true
		}
	}
	return false
}
