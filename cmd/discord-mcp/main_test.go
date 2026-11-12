package main

import (
	"slices"
	"testing"
)

// The deployed pods run `discord-mcp serve`, older invocations ran bare
// `discord-mcp --port …`, and mining runs `discord-mcp mine …`; all three
// must route to the right mode with their own flags intact.
func TestSubcommand_RoutesModeAndKeepsFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		args     []string
		wantMode string
		wantRest []string
	}{
		{name: "no args serves", args: nil, wantMode: "serve", wantRest: nil},
		{name: "explicit serve", args: []string{"serve"}, wantMode: "serve", wantRest: []string{}},
		{name: "serve with port", args: []string{"serve", "--port", "127.0.0.1:9999"}, wantMode: "serve", wantRest: []string{"--port", "127.0.0.1:9999"}},
		{name: "bare flags serve", args: []string{"--port", "127.0.0.1:9999"}, wantMode: "serve", wantRest: []string{"--port", "127.0.0.1:9999"}},
		{name: "mine with flags", args: []string{"mine", "--dry-run", "--archive", "/a"}, wantMode: "mine", wantRest: []string{"--dry-run", "--archive", "/a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mode, rest := subcommand(tt.args)
			if mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", mode, tt.wantMode)
			}
			if !slices.Equal(rest, tt.wantRest) {
				t.Errorf("rest = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}

func TestSubcommand_UnknownWordIsReportedNotServed(t *testing.T) {
	t.Parallel()
	mode, _ := subcommand([]string{"mien"})
	if mode != "mien" {
		t.Errorf("mode = %q, want the unknown word returned so run() can reject it", mode)
	}
}
