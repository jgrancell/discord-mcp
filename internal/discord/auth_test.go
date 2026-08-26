package discord_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

// Behavior 38: a 401 aborts the entire run immediately without retrying.
func TestClient_401_NeverRetried(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"401: Unauthorized","code":0}`))
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "bad-token")
	_, err := c.Messages(context.Background(), "1122334455667788", discord.MessageParams{Limit: 50})
	if err == nil {
		t.Fatal("Messages: want an error on 401, got nil")
	}
	if !errors.Is(err, discord.ErrUnauthorized) {
		t.Errorf("err = %v, want errors.Is(err, discord.ErrUnauthorized)", err)
	}
	if calls != 1 {
		t.Errorf("Discord received %d requests, want exactly 1 (401 is never retried)", calls)
	}
}

// Behavior 39: a 403 on one channel records it as denied and the walk
// continues to the next channel — at the client level, a 403 on one call
// does not corrupt state or block a later call to a different channel.
func TestClient_403_RecordsDeniedAndClientContinues(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "denied-channel") {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"403: Forbidden","code":50001}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "test-token")
	ctx := context.Background()

	_, err := c.Messages(ctx, "denied-channel", discord.MessageParams{Limit: 50})
	if err == nil {
		t.Fatal("Messages: want an error on 403, got nil")
	}
	var forbidden *discord.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("err = %T (%v), want *discord.ForbiddenError", err, err)
	}
	if forbidden.ChannelID != "denied-channel" {
		t.Errorf("ForbiddenError.ChannelID = %q, want %q", forbidden.ChannelID, "denied-channel")
	}

	// The client itself is unaffected: a later call to a different channel
	// succeeds normally.
	if _, err := c.Messages(ctx, "allowed-channel", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("Messages(allowed-channel) after a prior 403: %v, want success", err)
	}
}

// Behavior 41: every outbound request carries Authorization: Bot <token>
// and a User-Agent.
func TestClient_Messages_SetsAuthorizationAndUserAgent(t *testing.T) {
	t.Parallel()
	var gotAuth, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	c := discord.New(srv.URL, "s3cr3t-token")
	if _, err := c.Messages(context.Background(), "1122334455667788", discord.MessageParams{Limit: 50}); err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if gotAuth != "Bot s3cr3t-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bot s3cr3t-token")
	}
	if gotUA == "" {
		t.Error("User-Agent header is empty, want a non-empty identifying value")
	}
}
