package mine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/mine"
)

// Behavior 51a [OPERATOR RULING, Josh 2026-08-22]: every mine run writes
// exactly one report file under _runs/, named from the injected clock
// (RFC3339), containing the report fields in the design's "Run report"
// section; a re-run at the same injected clock instant overwrites rather
// than duplicates.

// Behavior 51a(a): a run writes exactly one file under _runs/, named from
// the injected clock in RFC3339.
func TestRun_WritesReportFile_NamedFromInjectedClock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(testChannelID, "rules-questions", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/messages":
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
	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	name, _ := soleReportFile(t, root)
	want := fixedNow.Format(time.RFC3339) + ".md"
	if name != want {
		t.Errorf("report file name = %q, want %q (RFC3339 of the injected clock)", name, want)
	}
}

// Behavior 51a(b): the report file's content includes the report's key
// fields per the design's Run-report paragraph — channels attempted,
// channels denied (with ID and name), messages archived, and labeled
// sections for windows, requests, and failures. Representative-field
// presence, not exact formatting.
func TestRun_ReportFileContent_IncludesKeyReportFields(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const deniedID = "9988776655443322"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` +
				channelJSON(testChannelID, "rules-questions", 0) + `,` +
				channelJSON(deniedID, "admin-only", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/messages":
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` +
					discordMessageJSON("100", testChannelID, "first", "2026-06-03T14:22:00.000000+00:00") + `,` +
					discordMessageJSON("101", testChannelID, "second", "2026-06-03T14:25:00.000000+00:00") + `]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case strings.Contains(r.URL.Path, deniedID) && strings.Contains(r.URL.Path, "/messages"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"403: Forbidden","code":50001}`))
		case strings.Contains(r.URL.Path, deniedID) && strings.Contains(r.URL.Path, "/threads/archived"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"403: Forbidden","code":50001}`))
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
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.ChannelsAttempted) == 0 {
		t.Fatal("test setup: report.ChannelsAttempted is empty")
	}
	if len(report.Denied) != 1 {
		t.Fatalf("test setup: report.Denied = %+v, want exactly one entry", report.Denied)
	}

	_, content := soleReportFile(t, root)

	for _, id := range report.ChannelsAttempted {
		if !strings.Contains(content, id) {
			t.Errorf("report content missing attempted channel ID %q:\n%s", id, content)
		}
	}
	denied := report.Denied[0]
	if !strings.Contains(content, denied.ID) {
		t.Errorf("report content missing denied channel ID %q:\n%s", denied.ID, content)
	}
	if !strings.Contains(content, denied.Name) {
		t.Errorf("report content missing denied channel name %q:\n%s", denied.Name, content)
	}
	if !strings.Contains(content, strconv.Itoa(report.MessagesArchived)) {
		t.Errorf("report content missing messages-archived count %d:\n%s", report.MessagesArchived, content)
	}

	// Labeled sections per the design's Run-report field list, even where
	// this run has nothing to report for them (0 failures) — the design
	// specifies these as always-present fields, not conditional ones.
	lower := strings.ToLower(content)
	for _, want := range []string{"window", "request", "fail"} {
		if !strings.Contains(lower, want) {
			t.Errorf("report content has no section referencing %q, want one per the design's Run-report fields:\n%s", want, content)
		}
	}
}

// Behavior 51a(c): two runs at the same injected clock instant leave
// exactly one file under _runs/ (overwrite semantics, not duplication), and
// the second run's content reflects its own fresh report rather than the
// first run's stale one.
func TestRun_SameClockInstantTwice_OverwritesReportFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(testChannelID, "rules-questions", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/private":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/messages":
			after := r.URL.Query().Get("after")
			if after == "0" || after == "" {
				_, _ = w.Write([]byte(`[` +
					discordMessageJSON("100", testChannelID, "first", "2026-06-03T14:22:00.000000+00:00") + `,` +
					discordMessageJSON("101", testChannelID, "second", "2026-06-03T14:25:00.000000+00:00") + `]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
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

	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	_, firstContent := soleReportFile(t, root)

	if _, err := mine.Run(context.Background(), cfg); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	name, secondContent := soleReportFile(t, root)

	want := fixedNow.Format(time.RFC3339) + ".md"
	if name != want {
		t.Errorf("report file name after second run = %q, want %q", name, want)
	}
	if firstContent == secondContent {
		t.Error("report file content identical after second run at the same clock instant: want the second run's fresh report (e.g. zero new messages archived) to have overwritten the first run's — got stale content, or the write was skipped")
	}
}

// Behavior 51a(d): ThreadsCapabilityDenied entries (channels where the bot
// lacks Manage Threads, so private archived threads are invisible) appear
// in the report content when present.
func TestRun_ThreadsCapabilityDenied_AppearsInReportFileContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/guilds/"+guildID+"/channels":
			_, _ = w.Write([]byte(`[` + channelJSON(testChannelID, "rules-questions", 0) + `]`))
		case r.URL.Path == "/guilds/"+guildID+"/threads/active":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/public":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.URL.Path == "/channels/"+testChannelID+"/threads/archived/private":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"403: Forbidden","code":50001}`))
		case r.URL.Path == "/channels/"+testChannelID+"/messages":
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
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.ThreadsCapabilityDenied) != 1 {
		t.Fatalf("test setup: report.ThreadsCapabilityDenied = %+v, want exactly one entry", report.ThreadsCapabilityDenied)
	}

	_, content := soleReportFile(t, root)

	want := report.ThreadsCapabilityDenied[0]
	if !strings.Contains(content, want.ID) {
		t.Errorf("report content missing ThreadsCapabilityDenied channel ID %q:\n%s", want.ID, content)
	}
	if !strings.Contains(content, want.Name) {
		t.Errorf("report content missing ThreadsCapabilityDenied channel name %q:\n%s", want.Name, content)
	}
}

// soleReportFile asserts root/_runs/ contains exactly one file and returns
// its name and content. Fails the test otherwise.
func soleReportFile(t *testing.T, root string) (name, content string) {
	t.Helper()
	dir := filepath.Join(root, "_runs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("_runs/ contains %d files, want exactly 1: %v", len(entries), entryNames(entries))
	}
	b, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile %s: %v", entries[0].Name(), err)
	}
	return entries[0].Name(), string(b)
}

func entryNames(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name()
	}
	return out
}
