package archive_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/archive"
)

// Behavior 22: writing a window file and reading its frontmatter back
// round-trips every field.
func TestWriteWindow_ReadWindow_RoundTripsAllFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := archive.WindowPath(dir, "2026-06")
	meta := windowMeta("2026-06", "1247000000000000000", "1249999999999999999", true)
	body := "## 2026-06-03\n\n### 14:22 — Janak (janak)\n\nRested state inherits.\n"

	if _, err := archive.WriteWindow(path, meta, body); err != nil {
		t.Fatalf("WriteWindow: %v", err)
	}

	gotMeta, gotBody, err := archive.ReadWindow(path)
	if err != nil {
		t.Fatalf("ReadWindow: %v", err)
	}
	if gotMeta != meta {
		t.Errorf("ReadWindow() meta = %+v, want %+v", gotMeta, meta)
	}
	if gotBody != body {
		t.Errorf("ReadWindow() body = %q, want %q", gotBody, body)
	}
}

// Behavior 26: a window marked complete: true is not rewritten by a
// subsequent write pass.
func TestWriteWindow_CompleteWindow_NotRewritten(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := archive.WindowPath(dir, "2026-05")
	meta := windowMeta("2026-05", "1", "100", true)
	original := "## 2026-05-01\n\noriginal content\n"
	if _, err := archive.WriteWindow(path, meta, original); err != nil {
		t.Fatalf("initial WriteWindow: %v", err)
	}

	newMeta := windowMeta("2026-05", "1", "200", true)
	wrote, err := archive.WriteWindow(path, newMeta, "attempted overwrite\n")
	if err != nil {
		t.Fatalf("second WriteWindow: %v", err)
	}
	if wrote {
		t.Error("WriteWindow() wrote = true for a complete window, want false")
	}

	_, gotBody, err := archive.ReadWindow(path)
	if err != nil {
		t.Fatalf("ReadWindow: %v", err)
	}
	if gotBody != original {
		t.Errorf("ReadWindow() body = %q after no-op write, want original %q", gotBody, original)
	}
}

// Behavior 27: the single complete: false window is rewritten in place on a
// subsequent pass.
func TestWriteWindow_OpenWindow_RewrittenInPlace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := archive.WindowPath(dir, "2026-08")
	meta := windowMeta("2026-08", "1", "100", false)
	if _, err := archive.WriteWindow(path, meta, "first pass\n"); err != nil {
		t.Fatalf("initial WriteWindow: %v", err)
	}

	newMeta := windowMeta("2026-08", "1", "200", false)
	wrote, err := archive.WriteWindow(path, newMeta, "second pass\n")
	if err != nil {
		t.Fatalf("second WriteWindow: %v", err)
	}
	if !wrote {
		t.Error("WriteWindow() wrote = false for an open window, want true")
	}

	gotMeta, gotBody, err := archive.ReadWindow(path)
	if err != nil {
		t.Fatalf("ReadWindow: %v", err)
	}
	if gotBody != "second pass\n" {
		t.Errorf("ReadWindow() body = %q, want the rewritten content", gotBody)
	}
	if gotMeta.LastMessageID != "200" {
		t.Errorf("ReadWindow() last_message_id = %q, want %q", gotMeta.LastMessageID, "200")
	}
}

// Behavior 28: a write is atomic — an induced failure mid-write leaves the
// prior file intact and no temp file behind. The directory is made
// read-only after the first successful write so the temp-file create for
// the second write fails.
func TestWriteWindow_InducedFailure_LeavesPriorFileIntactNoTempFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions do not block writes")
	}
	t.Parallel()
	dir := t.TempDir()
	path := archive.WindowPath(dir, "2026-07")
	meta := windowMeta("2026-07", "1", "100", false)
	original := "content that must survive\n"
	if _, err := archive.WriteWindow(path, meta, original); err != nil {
		t.Fatalf("initial WriteWindow: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	newMeta := windowMeta("2026-07", "1", "200", false)
	if _, err := archive.WriteWindow(path, newMeta, "this must not land\n"); err == nil {
		t.Fatal("WriteWindow() into a read-only directory returned nil error, want failure")
	}

	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod writable: %v", err)
	}

	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after induced failure: %v", err)
	}
	if !strings.Contains(string(gotBytes), original) {
		t.Errorf("file content after induced failure = %q, want it to still contain %q", gotBytes, original)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") || strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
