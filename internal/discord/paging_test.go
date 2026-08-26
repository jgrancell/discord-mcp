package discord_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

// discordMessageJSON builds a minimal Discord API message object.
func discordMessageJSON(id, content string) string {
	return `{"id":"` + id + `","channel_id":"1122334455667788","content":"` + content + `",` +
		`"timestamp":"2026-06-03T14:22:00.000000+00:00","type":0,` +
		`"author":{"id":"999000111222333444","username":"janak","global_name":"Janak","bot":false}}`
}

// Behavior 31: a message batch returned newest-first is sorted ascending by
// snowflake before it reaches the caller.
func TestClient_Messages_SortsBatchAscendingBySnowflake(t *testing.T) {
	t.Parallel()
	// Discord returns newest-first: 300, 200, 100.
	body := `[` + discordMessageJSON("300", "third") + `,` +
		discordMessageJSON("200", "second") + `,` +
		discordMessageJSON("100", "first") + `]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "test-token")
	got, err := c.Messages(context.Background(), "1122334455667788", discord.MessageParams{Limit: 100})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Messages() returned %d messages, want 3", len(got))
	}
	wantIDs := []string{"100", "200", "300"}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Errorf("Messages()[%d].ID = %q, want %q (ascending order)", i, got[i].ID, want)
		}
	}
}

// Behavior 32: a forward walk advances the cursor to max(id) in the batch,
// not to the first or last array element — MaxMessageID compares snowflake
// values even when the slice is not already sorted.
func TestMaxMessageID_ReturnsHighestSnowflakeRegardlessOfSliceOrder(t *testing.T) {
	t.Parallel()
	msgs := []discord.Message{
		{ID: "1249999999999999999"},
		{ID: "1247000000000000000"}, // smallest, appears last
		{ID: "1248500000000000000"},
	}

	got := discord.MaxMessageID(msgs)
	want := "1249999999999999999"
	if got != want {
		t.Errorf("MaxMessageID() = %q, want %q", got, want)
	}
}

// Behavior 33: a batch shorter than limit terminates the walk — no further
// page is requested once a short page is seen.
func TestClient_WalkMessages_ShortBatchTerminatesWalk(t *testing.T) {
	t.Parallel()
	const limit = 3
	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			// Full page: exactly `limit` messages.
			_, _ = w.Write([]byte(`[` + discordMessageJSON("300", "c") + `,` +
				discordMessageJSON("200", "b") + `,` + discordMessageJSON("100", "a") + `]`))
		case 2:
			// Short page: fewer than `limit` messages — the walk must stop here.
			_, _ = w.Write([]byte(`[` + discordMessageJSON("400", "d") + `]`))
		default:
			t.Errorf("unexpected additional page request (call %d)", calls)
			w.Write([]byte(`[]`))
		}
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "test-token")
	var batches [][]discord.Message
	err := c.WalkMessages(context.Background(), "1122334455667788", "0", limit, func(batch []discord.Message) error {
		batches = append(batches, batch)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkMessages: %v", err)
	}
	if calls != 2 {
		t.Errorf("Discord received %d requests, want 2 (walk should stop after the short page)", calls)
	}
	if len(batches) != 2 {
		t.Errorf("fn was invoked %d times, want 2", len(batches))
	}
}
