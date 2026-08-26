//go:build e2e

// Package e2e drives the real binary against a real Discord guild.
// Kept out of `go test ./...` behind the e2e build tag; run explicitly
// once a runner script exists. Each stub below names one behavior from the
// design's End-to-end group (discord-mcp-design.md, "Behaviors to test").
package e2e

import "testing"

// Behavior 79: mine --dry-run against the real guild enumerates channels
// and reports a plan without writing.
func TestEndToEnd_MineDryRun_EnumeratesChannelsWritesNothing(t *testing.T) {
	t.Skip("e2e: requires the real binary and real guild credentials — not yet wired to a runner")
}

// Behavior 80: mine --channel <id> against a low-traffic real channel
// produces a window file whose message count matches the channel, with
// non-empty message bodies — the Message Content intent assertion.
func TestEndToEnd_MineOneChannel_ProducesWindowWithNonEmptyBodies(t *testing.T) {
	t.Skip("e2e: requires the real binary and real guild credentials — not yet wired to a runner")
}

// Behavior 81: serve plus a real MCP session calls get_messages against
// the real guild and returns rendered text.
func TestEndToEnd_ServeGetMessages_ReturnsRenderedText(t *testing.T) {
	t.Skip("e2e: requires the real binary and real guild credentials — not yet wired to a runner")
}
