package tools_test

import (
	"strings"
	"testing"
)

// Behavior 53: list_channels renders channel names, IDs, types, and
// categories from mocked data.
func TestListChannels_RendersNamesIDsTypesFromMockedData(t *testing.T) {
	t.Parallel()
	out := callText(t, session(t), "list_channels", map[string]any{})

	for _, want := range []string{"rules-questions", rulesChannelID} {
		if !strings.Contains(out, want) {
			t.Errorf("list_channels output missing %q:\n%s", want, out)
		}
	}
}

// Behavior 54: list_channels omits empty channels by default and includes
// them with include_empty.
func TestListChannels_EmptyChannelVisibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		includeEmpty bool
		wantVisible  bool
	}{
		{name: "default omits empty channels", includeEmpty: false, wantVisible: false},
		{name: "include_empty includes them", includeEmpty: true, wantVisible: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := callText(t, session(t), "list_channels", map[string]any{"include_empty": tt.includeEmpty})
			contains := strings.Contains(out, "scratch")
			if contains != tt.wantVisible {
				t.Errorf("list_channels(include_empty=%v) contains %q = %v, want %v:\n%s", tt.includeEmpty, "scratch", contains, tt.wantVisible, out)
			}
		})
	}
}
