package mine_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Regression coverage (QA iteration 3), for the design's Message Content
// Intent startup assertion (design doc ~line 492, Failure modes table):
// mine samples the first batch of the first channel; if every message in
// it is blank (empty content, no attachments), the run aborts rather than
// archiving blanks — Discord returns this silently, with no error, so mine
// is the only place that can catch it. This is an orchestration-level
// check, one layer above the table-driven checkMessageContentIntent unit
// tests in content_intent_test.go: it asserts that mine.Run itself stops
// the walk before a later channel is ever requested, and that the defer
// contract (behavior 51a) still leaves a partial report file recording how
// far the run got.
func TestRun_FirstBatchAllBlank_AbortsRunSkipsLaterChannelsRecordsPartialReport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const channelA = testChannelID
	const channelB = "7766554433221100"

	channelBHits := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(channelA, "rules-questions", 0) + `,` + channelJSON(channelB, "off-topic", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelA+"/threads/archived/public",
			r.URL.Path == "/channels/"+channelA+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelA+"/messages":
			// Every message in the first batch is blank: empty content,
			// and discordMessageJSON emits neither "attachments" nor
			// "embeds", so both decode as empty too — the signature of
			// Message Content Intent being disabled.
			_, _ = w.Write([]byte(`[` +
				discordMessageJSON("100", channelA, "", "2026-06-03T14:22:00.000000+00:00") + `,` +
				discordMessageJSON("101", channelA, "", "2026-06-03T14:25:00.000000+00:00") + `]`))
		case r.URL.Path == "/channels/"+channelB+"/threads/archived/public",
			r.URL.Path == "/channels/"+channelB+"/threads/archived/private":
			channelBHits++
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+channelB+"/messages":
			channelBHits++
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "test-token")
	cfg := mine.Config{
		Discord: c, Root: root, GuildID: guildID, GuildName: "OMGCG",
		Now: func() time.Time { return fixedNow },
	}

	report, err := mine.Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("Run: want an error when the first channel's first batch is all-blank, got nil")
	}
	if !errors.Is(err, mine.ErrMessageContentIntentDisabled) {
		t.Errorf("Run() err = %v, want errors.Is(err, mine.ErrMessageContentIntentDisabled)", err)
	}
	if channelBHits != 0 {
		t.Errorf("channel B's endpoints received %d requests after channel A's blank-content abort, want 0 (the run must abort before reaching a later channel)", channelBHits)
	}
	if len(report.ChannelsAttempted) != 1 || report.ChannelsAttempted[0] != channelA {
		t.Errorf("report.ChannelsAttempted = %v, want exactly [%q] (the abort stops the walk after channel A, before channel B is ever attempted)", report.ChannelsAttempted, channelA)
	}

	// The defer contract (behavior 51a): Run always writes the run's report
	// file. The partial report records the abort by way of the run's true
	// partial state — channel A attempted, channel B never reached.
	_, content := soleReportFile(t, root)
	if !strings.Contains(content, channelA) {
		t.Errorf("partial report content missing attempted channel ID %q:\n%s", channelA, content)
	}
	if strings.Contains(content, channelB) {
		t.Errorf("partial report content contains channel B's ID %q, want it absent (channel B was never attempted before the abort):\n%s", channelB, content)
	}
}
