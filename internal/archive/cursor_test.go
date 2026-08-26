package archive_test

import (
	"testing"

	"github.com/jgrancell/discord-mcp/internal/archive"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behavior 23: the resume cursor for a channel is max(last_message_id)
// across its window files.
func TestResumeCursor_ReturnsMaxLastMessageIDAcrossWindows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, archive.WindowPath(dir, "2024-11"), windowMeta("2024-11", "100", "200", true))
	mustWrite(t, archive.WindowPath(dir, "2024-12"), windowMeta("2024-12", "201", "500", true))
	mustWrite(t, archive.WindowPath(dir, "2026-08"), windowMeta("2026-08", "501", "999", false))

	got, err := archive.ResumeCursor(dir)
	if err != nil {
		t.Fatalf("ResumeCursor: %v", err)
	}
	if got != "999" {
		t.Errorf("ResumeCursor() = %q, want %q", got, "999")
	}
}

// Behavior 24: the resume cursor is derived correctly when a middle window
// file is missing — it still advances from the newest surviving file.
func TestResumeCursor_MiddleWindowMissing_StillAdvancesFromNewest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, archive.WindowPath(dir, "2024-11"), windowMeta("2024-11", "100", "200", true))
	// 2024-12 deliberately absent — the gap.
	mustWrite(t, archive.WindowPath(dir, "2026-08"), windowMeta("2026-08", "501", "999", false))

	got, err := archive.ResumeCursor(dir)
	if err != nil {
		t.Fatalf("ResumeCursor: %v", err)
	}
	if got != "999" {
		t.Errorf("ResumeCursor() = %q, want %q (unaffected by the missing middle file)", got, "999")
	}
}

// Behavior 25: a channel directory with no window files yields a zero
// cursor (start from the beginning, matching the walk's after=0).
func TestResumeCursor_NoWindowFiles_YieldsZeroCursor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // empty: no window files written

	got, err := archive.ResumeCursor(dir)
	if err != nil {
		t.Fatalf("ResumeCursor: %v", err)
	}
	if got != "0" {
		t.Errorf("ResumeCursor() = %q, want %q for an empty channel directory", got, "0")
	}
}

// mustWrite writes a window file for setup, failing the test on error.
func mustWrite(t *testing.T, path string, meta render.WindowMeta) {
	t.Helper()
	if _, err := archive.WriteWindow(path, meta, "content\n"); err != nil {
		t.Fatalf("WriteWindow(%s): %v", path, err)
	}
}
