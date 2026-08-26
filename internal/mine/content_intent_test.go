package mine

import (
	"errors"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

// Regression coverage (QA iteration 3), for the design's Message Content
// Intent heuristic (design doc ~line 492, Failure modes table): "Message
// payloads decode with empty content and empty embeds/attachments" is the
// signature of the privileged intent being disabled, silently, with no
// error from Discord. checkMessageContentIntent is unexported, so this file
// is a white-box `package mine` test — the repo's other mine tests are all
// `package mine_test` black-box, but that package cannot reach an
// unexported identifier, and per the repo's stated testing convention
// (golang.md: "same package" is the default; `_test` is reserved for
// black-box-only cases) a same-package test is the correct tool here.
//
// The "only embeds" case is a KNOWN DEFECT, left deliberately red: the
// design's heuristic is "empty content AND empty embeds AND empty
// attachments" (all three), but the current implementation
// (helpers.go:36-40) checks only Content and Attachments — it never looks
// at Embeds. A batch that is blank except for an embed is therefore
// misclassified as the intent being disabled. This test asserts the
// design's intended behavior (no error) and fails against today's code
// with ErrMessageContentIntentDisabled; the Developer's next change is to
// add the Embeds check.
func TestCheckMessageContentIntent_HeuristicCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		batch   []discord.Message
		wantErr bool
	}{
		{
			name: "all blank content, no attachments, no embeds aborts",
			batch: []discord.Message{
				{ID: "100", Content: ""},
				{ID: "101", Content: ""},
			},
			wantErr: true,
		},
		{
			name: "batch with text content passes",
			batch: []discord.Message{
				{ID: "100", Content: "does the L2 stay rested when you evolve onto it?"},
			},
			wantErr: false,
		},
		{
			name: "batch with only attachments passes",
			batch: []discord.Message{
				{
					ID: "100", Content: "",
					Attachments: []discord.Attachment{
						{Filename: "ruling.png", ContentType: "image/png", Size: 214000, URL: "https://cdn.discordapp.com/attachments/x/ruling.png"},
					},
				},
			},
			wantErr: false,
		},
		{
			// KNOWN DEFECT, see the file comment: fails against current
			// code because checkMessageContentIntent never inspects
			// Embeds.
			name: "batch with only embeds passes",
			batch: []discord.Message{
				{
					ID: "100", Content: "",
					Embeds: []discord.Embed{
						{Title: "Rules — Evolution", URL: "https://omgcardgame.com/rules#evolution", Description: "Evolution power is the sum..."},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := &Report{}
			err := checkMessageContentIntent(report, tt.batch)

			if tt.wantErr {
				if !errors.Is(err, ErrMessageContentIntentDisabled) {
					t.Errorf("checkMessageContentIntent() err = %v, want errors.Is(err, ErrMessageContentIntentDisabled)", err)
				}
				return
			}
			if err != nil {
				t.Errorf("checkMessageContentIntent() err = %v, want nil (batch carries signal: text, attachment, or embed content)", err)
			}
		})
	}
}
