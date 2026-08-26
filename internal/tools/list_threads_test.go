package tools_test

import (
	"strings"
	"testing"
)

// Behavior 61: list_threads returns active and archived threads together
// and labels which is which.
func TestListThreads_ReturnsActiveAndArchivedLabeled(t *testing.T) {
	t.Parallel()
	out := callText(t, session(t), "list_threads", map[string]any{"channel_id": rulesChannelID})

	for _, want := range []string{"Card game night", "Rest inheritance"} {
		if !strings.Contains(out, want) {
			t.Errorf("list_threads output missing %q:\n%s", want, out)
		}
	}

	activeIdx := strings.Index(out, "Card game night")
	archivedIdx := strings.Index(out, "Rest inheritance")
	activeRegion := out[max(0, activeIdx-80):activeIdx]
	archivedRegion := out[max(0, archivedIdx-80):archivedIdx]

	if !strings.Contains(strings.ToLower(activeRegion), "active") {
		t.Errorf("active thread not labeled as active nearby:\n%s", activeRegion)
	}
	if !strings.Contains(strings.ToLower(archivedRegion), "archiv") {
		t.Errorf("archived thread not labeled as archived nearby:\n%s", archivedRegion)
	}
}
