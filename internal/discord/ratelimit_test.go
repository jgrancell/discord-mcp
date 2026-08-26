package discord_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

var fixedStart = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

// Behavior 34: X-RateLimit-Remaining: 0 causes a sleep of Reset-After plus
// jitter before the next request, asserted on the injected clock with no
// real sleeping.
func TestClient_ZeroRemaining_SleepsResetAfterBeforeNextRequest(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset-After", "2.5")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	clock := newFakeClock(fixedStart)
	c := discord.New(srv.URL, "test-token",
		discord.WithClock(clock.Now, clock.Sleep),
		discord.WithJitter(zeroJitter),
		discord.WithMinInterval(0),
	)

	ctx := context.Background()
	if _, err := c.Messages(ctx, "1122334455667788", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("first Messages: %v", err)
	}
	if _, err := c.Messages(ctx, "1122334455667788", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("second Messages: %v", err)
	}

	found := false
	for _, d := range clock.Sleeps() {
		if d == 2500*time.Millisecond {
			found = true
		}
	}
	if !found {
		t.Errorf("Sleeps() = %v, want a 2.5s sleep before the second request", clock.Sleeps())
	}
}

// Behavior 35: a 429 with retry_after sleeps that duration and retries the
// same request, which then succeeds.
func TestClient_429WithRetryAfter_SleepsAndRetriesSucceeds(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"rate limited","retry_after":1.5,"global":false}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	clock := newFakeClock(fixedStart)
	c := discord.New(srv.URL, "test-token",
		discord.WithClock(clock.Now, clock.Sleep),
		discord.WithJitter(zeroJitter),
		discord.WithMinInterval(0),
	)

	_, err := c.Messages(context.Background(), "1122334455667788", discord.MessageParams{Limit: 50})
	if err != nil {
		t.Fatalf("Messages: %v (want success after one retry)", err)
	}
	if calls != 2 {
		t.Errorf("Discord received %d requests, want 2 (original + one retry)", calls)
	}
	sleeps := clock.Sleeps()
	if len(sleeps) == 0 || sleeps[0] != 1500*time.Millisecond {
		t.Errorf("Sleeps() = %v, want the first sleep to be 1.5s", sleeps)
	}
}

// Behavior 36: five consecutive 429s on one request fail the channel and
// return an error naming it.
func TestClient_FiveConsecutive429s_FailsNamingTheChannel(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"rate limited","retry_after":0.01,"global":false}`))
	}))
	t.Cleanup(srv.Close)

	clock := newFakeClock(fixedStart)
	c := discord.New(srv.URL, "test-token",
		discord.WithClock(clock.Now, clock.Sleep),
		discord.WithJitter(zeroJitter),
		discord.WithMinInterval(0),
	)

	const channelID = "1122334455667788"
	_, err := c.Messages(context.Background(), channelID, discord.MessageParams{Limit: 50})
	if err == nil {
		t.Fatal("Messages: want an error after five consecutive 429s, got nil")
	}
	if !strings.Contains(err.Error(), channelID) {
		t.Errorf("error = %q, want it to name the channel %q", err.Error(), channelID)
	}
	if calls != 5 {
		t.Errorf("Discord received %d requests, want exactly 5 attempts before giving up", calls)
	}
}

// Behavior 37: a 429 with global: true halves the request interval for all
// subsequent calls in the run and does not restore it.
func TestClient_Global429_HalvesIntervalPermanently(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.Header().Set("X-RateLimit-Global", "true")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"global rate limited","retry_after":0.01,"global":true}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	clock := newFakeClock(fixedStart)
	c := discord.New(srv.URL, "test-token",
		discord.WithClock(clock.Now, clock.Sleep),
		discord.WithJitter(zeroJitter),
		discord.WithMinInterval(250*time.Millisecond),
	)

	ctx := context.Background()
	// Call 1: absorbs the global 429 (retries once, succeeds).
	if _, err := c.Messages(ctx, "1122334455667788", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("first Messages: %v", err)
	}

	// Call 2: floor delay must now be halved to 125ms.
	if _, err := c.Messages(ctx, "1122334455667788", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("second Messages: %v", err)
	}
	sleeps := clock.Sleeps()
	if len(sleeps) == 0 || sleeps[len(sleeps)-1] != 125*time.Millisecond {
		t.Fatalf("floor delay before call 2 = %v (last sleep), want 125ms (halved from 250ms)", sleeps)
	}

	// Call 3: still halved — never restored.
	if _, err := c.Messages(ctx, "1122334455667788", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("third Messages: %v", err)
	}
	sleeps = clock.Sleeps()
	if sleeps[len(sleeps)-1] != 125*time.Millisecond {
		t.Errorf("floor delay before call 3 = %v (last sleep), want still 125ms (not restored)", sleeps)
	}
}

// Behavior 40: a 5xx retries with exponential backoff and succeeds on a
// later attempt.
func TestClient_5xx_RetriesWithExponentialBackoffThenSucceeds(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	clock := newFakeClock(fixedStart)
	c := discord.New(srv.URL, "test-token",
		discord.WithClock(clock.Now, clock.Sleep),
		discord.WithJitter(zeroJitter),
		discord.WithMinInterval(0),
	)

	_, err := c.Messages(context.Background(), "1122334455667788", discord.MessageParams{Limit: 50})
	if err != nil {
		t.Fatalf("Messages: %v (want success on the third attempt)", err)
	}
	if calls != 3 {
		t.Errorf("Discord received %d requests, want 3 (two failures + one success)", calls)
	}
	sleeps := clock.Sleeps()
	if len(sleeps) < 2 {
		t.Fatalf("Sleeps() = %v, want at least 2 backoff sleeps", sleeps)
	}
	if sleeps[1] <= sleeps[0] {
		t.Errorf("backoff sleeps = %v, want increasing (exponential) durations", sleeps)
	}
}
