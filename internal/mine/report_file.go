package mine

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jgrancell/discord-mcp/internal/archive"
)

// writeReportFile writes the run's report to <root>/_runs/<RFC3339>.md,
// named from the run's start time on the injected clock (behavior 51a). A
// re-run at the same clock instant overwrites the prior file rather than
// duplicating it.
func writeReportFile(cfg Config, r Report) error {
	name := r.Start.Format(time.RFC3339) + ".md"
	if err := archive.WriteRunReport(cfg.Root, name, renderReport(cfg, r)); err != nil {
		return fmt.Errorf("write run report: %w", err)
	}
	return nil
}

// renderReport formats a run's Report as markdown, per the design's
// Run-report paragraph: this is the observability surface for a mode with
// no metrics endpoint, and the audit trail for what was read and when.
func renderReport(cfg Config, r Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Mine run report — %s\n\n", cfg.GuildName)
	fmt.Fprintf(&b, "- start: %s\n", r.Start.Format(time.RFC3339))
	fmt.Fprintf(&b, "- end: %s\n", r.End.Format(time.RFC3339))
	fmt.Fprintf(&b, "- duration: %s\n\n", r.End.Sub(r.Start))

	fmt.Fprintf(&b, "## Channels attempted (%d)\n\n", len(r.ChannelsAttempted))
	for _, id := range r.ChannelsAttempted {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## Channels denied (%d)\n\n", len(r.Denied))
	for _, dch := range r.Denied {
		fmt.Fprintf(&b, "- %s (%s)\n", dch.ID, dch.Name)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## Threads-capability denied (%d)\n\n", len(r.ThreadsCapabilityDenied))
	for _, dch := range r.ThreadsCapabilityDenied {
		fmt.Fprintf(&b, "- %s (%s)\n", dch.ID, dch.Name)
	}
	b.WriteString("\n")

	b.WriteString("## Messages archived\n\n")
	fmt.Fprintf(&b, "%s\n\n", strconv.Itoa(r.MessagesArchived))

	b.WriteString("## Windows\n\n")
	fmt.Fprintf(&b, "- opened: %d\n", r.WindowsOpened)
	fmt.Fprintf(&b, "- closed: %d\n\n", r.WindowsClosed)

	b.WriteString("## Threads archived\n\n")
	fmt.Fprintf(&b, "%d\n\n", r.ThreadsArchived)

	b.WriteString("## Requests\n\n")
	fmt.Fprintf(&b, "- made: %d\n", r.RequestsMade)
	fmt.Fprintf(&b, "- 429s absorbed: %d\n", r.RateLimited429)
	fmt.Fprintf(&b, "- global-limit hits: %d\n\n", r.GlobalLimitHits)

	fmt.Fprintf(&b, "## Failures (%d)\n\n", len(r.Failures))
	for _, f := range r.Failures {
		fmt.Fprintf(&b, "- %s (%s): %s\n", f.ID, f.Name, f.Error)
	}

	return b.String()
}
