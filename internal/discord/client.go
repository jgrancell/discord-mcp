// Package discord is the only package that speaks HTTP to Discord: a
// typed, read-only REST client with bot-token auth, snowflake ordering,
// and rate-limit handling.
package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const userAgent = "discord-mcp (github.com/jgrancell/discord-mcp, 0.1.0)"

// Client is a typed, read-only Discord REST client. Concurrency-safe.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client

	now    func() time.Time
	sleep  func(time.Duration)
	jitter func() time.Duration

	mu              sync.Mutex
	currentInterval time.Duration
	lastRequestAt   time.Time
	pendingSleep    time.Duration

	// Request-level counters, the observability surface for a mode with no
	// metrics endpoint (see Stats). Independent atomics rather than mu,
	// since a caller may read Stats concurrently with in-flight requests.
	requestsMade    atomic.Int64
	rateLimited429  atomic.Int64
	globalLimitHits atomic.Int64
}

// Stats is a snapshot of a Client's cumulative request-level counters.
type Stats struct {
	RequestsMade    int
	RateLimited429  int
	GlobalLimitHits int
}

// Stats returns a snapshot of the client's request counters. Concurrency-safe.
func (c *Client) Stats() Stats {
	return Stats{
		RequestsMade:    int(c.requestsMade.Load()),
		RateLimited429:  int(c.rateLimited429.Load()),
		GlobalLimitHits: int(c.globalLimitHits.Load()),
	}
}

// Option configures a Client.
type Option func(*Client)

// WithClock injects the clock a Client uses for pacing decisions, so tests
// run with no real sleeping.
func WithClock(now func() time.Time, sleep func(time.Duration)) Option {
	return func(c *Client) { c.now = now; c.sleep = sleep }
}

// WithJitter injects the jitter added to every rate-limit sleep.
func WithJitter(jitter func() time.Duration) Option {
	return func(c *Client) { c.jitter = jitter }
}

// WithMinInterval sets the floor delay enforced between outbound requests.
func WithMinInterval(d time.Duration) Option {
	return func(c *Client) { c.currentInterval = d }
}

// WithHTTPTimeout sets the underlying http.Client's timeout, bounding
// every request to a deadline regardless of context.
func WithHTTPTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// New returns a Client against baseURL (the Discord API root, or a test
// server's URL), authenticating with token.
func New(baseURL, token string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		now:        time.Now,
		sleep:      time.Sleep,
		jitter:     defaultJitter,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func defaultJitter() time.Duration {
	return 50 * time.Millisecond
}

// MessageParams are the query parameters for listing channel messages.
// Fields are passed through to Discord verbatim when set.
type MessageParams struct {
	Before string
	After  string
	Around string
	Limit  int
}

func (p MessageParams) values() url.Values {
	v := url.Values{}
	if p.Before != "" {
		v.Set("before", p.Before)
	}
	if p.After != "" {
		v.Set("after", p.After)
	}
	if p.Around != "" {
		v.Set("around", p.Around)
	}
	if p.Limit != 0 {
		v.Set("limit", strconv.Itoa(p.Limit))
	}
	return v
}

// Messages returns a channel's messages matching params, sorted ascending
// by snowflake ID regardless of the order Discord returned them in.
func (c *Client) Messages(ctx context.Context, channelID string, params MessageParams) ([]Message, error) {
	body, err := c.get(ctx, "/channels/"+channelID+"/messages", params.values(), channelID)
	if err != nil {
		return nil, err
	}
	var wire []wireMessage
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode messages: %w", err)
	}
	msgs := make([]Message, len(wire))
	for i, w := range wire {
		msgs[i] = w.toMessage()
	}
	sort.Slice(msgs, func(i, j int) bool { return snowflakeLess(msgs[i].ID, msgs[j].ID) })
	return msgs, nil
}

// MaxMessageID returns the highest snowflake ID among msgs, regardless of
// slice order.
func MaxMessageID(msgs []Message) string {
	if len(msgs) == 0 {
		return ""
	}
	max := msgs[0].ID
	for _, m := range msgs[1:] {
		if snowflakeLess(max, m.ID) {
			max = m.ID
		}
	}
	return max
}

// WalkMessages pages forward through a channel's messages starting after
// the given cursor, calling fn with each batch in ascending order. The
// walk stops when a batch shorter than limit is seen.
func (c *Client) WalkMessages(ctx context.Context, channelID, after string, limit int, fn func([]Message) error) error {
	cursor := after
	for {
		batch, err := c.Messages(ctx, channelID, MessageParams{After: cursor, Limit: limit})
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		if err := fn(batch); err != nil {
			return err
		}
		cursor = MaxMessageID(batch)
		if len(batch) < limit {
			return nil
		}
	}
}

// snowflakeLess reports whether a < b as Discord snowflake IDs.
func snowflakeLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
