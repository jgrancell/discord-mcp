package mine

import (
	"context"
	"errors"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

// isContextErr reports whether err is (or wraps) context cancellation or a
// deadline — a run-ending condition, never a per-channel one.
func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// asForbidden extracts a *discord.ForbiddenError from err, if any.
func asForbidden(err error) (*discord.ForbiddenError, bool) {
	var fe *discord.ForbiddenError
	if errors.As(err, &fe) {
		return fe, true
	}
	return nil, false
}

// checkMessageContentIntent samples the first non-empty message batch
// fetched in a run: if every message has empty content and no
// attachments, the bot is almost certainly missing the privileged Message
// Content Intent, which Discord accepts silently rather than erroring.
// Checked at most once per run, off the walk's first real batch — no
// extra API call.
func checkMessageContentIntent(report *Report, batch []discord.Message) error {
	if report.contentIntentChecked || len(batch) == 0 {
		return nil
	}
	report.contentIntentChecked = true
	for _, m := range batch {
		if m.Content != "" || len(m.Attachments) > 0 || len(m.Embeds) > 0 {
			return nil
		}
	}
	return ErrMessageContentIntentDisabled
}
