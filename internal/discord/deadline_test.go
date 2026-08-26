package discord_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

// Behavior 42: every outbound request is built with a deadline. Exercised
// functionally: a client configured with a short HTTP timeout against a
// handler slower than that timeout must fail quickly rather than hang for
// the handler's full duration — proving a deadline is actually enforced on
// the wire, not merely documented.
func TestClient_Messages_RequestCarriesDeadline(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // held open well past the client's configured timeout
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	c := discord.New(srv.URL, "test-token", discord.WithHTTPTimeout(50*time.Millisecond))

	start := time.Now()
	_, err := c.Messages(context.Background(), "1122334455667788", discord.MessageParams{Limit: 50})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Messages: want a deadline error against a handler held open past the timeout")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Messages took %v to fail, want it bounded by the configured deadline, not the handler's hang", elapsed)
	}
}
