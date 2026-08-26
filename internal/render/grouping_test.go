package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
	"github.com/jgrancell/discord-mcp/internal/render"
)

// Behavior 19: two messages on the same UTC day share one H2; a message on
// the next day opens a new H2.
func TestMessages_DayGrouping_SharesH2ForSameDayOpensNewH2NextDay(t *testing.T) {
	t.Parallel()

	sameDayA := baseMessage()
	sameDayA.ID = "1"
	sameDayA.Timestamp = time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC)
	sameDayA.Content = "first message of the day"

	sameDayB := baseMessage()
	sameDayB.ID = "2"
	sameDayB.Timestamp = time.Date(2026, 6, 3, 22, 0, 0, 0, time.UTC)
	sameDayB.Content = "second message, same UTC day"

	nextDay := baseMessage()
	nextDay.ID = "3"
	nextDay.Timestamp = time.Date(2026, 6, 4, 0, 30, 0, 0, time.UTC)
	nextDay.Content = "first message of the next day"

	got := render.Messages([]discord.Message{sameDayA, sameDayB, nextDay}, emptyMentions())

	h2Count := strings.Count(got, "## 2026-06-03") + strings.Count(got, "## 2026-06-04")
	if strings.Count(got, "## 2026-06-03") != 1 {
		t.Errorf("Messages() = %q, want exactly one ## 2026-06-03 heading", got)
	}
	if strings.Count(got, "## 2026-06-04") != 1 {
		t.Errorf("Messages() = %q, want exactly one ## 2026-06-04 heading", got)
	}
	if h2Count != 2 {
		t.Errorf("Messages() produced %d day headings, want 2", h2Count)
	}

	day3Idx := strings.Index(got, "## 2026-06-03")
	day4Idx := strings.Index(got, "## 2026-06-04")
	if day3Idx == -1 || day4Idx == -1 || day3Idx > day4Idx {
		t.Fatalf("Messages() day headings out of order:\n%s", got)
	}
	between := got[day3Idx:day4Idx]
	if !strings.Contains(between, "first message of the day") || !strings.Contains(between, "second message, same UTC day") {
		t.Errorf("Messages() day-3 section missing expected messages:\n%s", between)
	}
}
