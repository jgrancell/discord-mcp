package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrUnauthorized indicates a 401 from Discord: the bot token is invalid
// or revoked. Never retried — retrying 401s is how a Cloudflare IP ban is
// earned.
var ErrUnauthorized = errors.New("discord: unauthorized (401): bot token invalid or revoked")

// ForbiddenError indicates a 403 from Discord: the bot lacks access to a
// specific channel or thread. Recorded as denied, never retried.
type ForbiddenError struct {
	ChannelID string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("discord: forbidden (403) for channel %s", e.ChannelID)
}

type rateLimitBody struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

const maxAttempts = 5

// get issues a GET request against path with the given query, applying
// pacing, rate-limit handling, and retries. resourceID names the
// channel/thread the request concerns, for 403 attribution.
func (c *Client) get(ctx context.Context, path string, query url.Values, resourceID string) ([]byte, error) {
	c.waitTurn()
	body, err := c.sendWithRetry(ctx, path, query, resourceID)
	c.mu.Lock()
	c.lastRequestAt = c.now()
	c.mu.Unlock()
	return body, err
}

// waitTurn applies any pending zero-remaining sleep, then the floor delay,
// before the next request is sent.
func (c *Client) waitTurn() {
	c.mu.Lock()
	pending := c.pendingSleep
	c.pendingSleep = 0
	interval := c.currentInterval
	last := c.lastRequestAt
	c.mu.Unlock()

	if pending > 0 {
		c.sleep(pending + c.jitter())
	}
	if !last.IsZero() && interval > 0 {
		if elapsed := c.now().Sub(last); elapsed < interval {
			c.sleep(interval - elapsed)
		}
	}
}

func (c *Client) sendWithRetry(ctx context.Context, path string, query url.Values, resourceID string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		body, status, header, err := c.do(ctx, path, query)
		if err != nil {
			return nil, fmt.Errorf("discord: request %s: %w", path, err)
		}

		switch {
		case status == http.StatusUnauthorized:
			return nil, ErrUnauthorized
		case status == http.StatusForbidden:
			return nil, &ForbiddenError{ChannelID: resourceID}
		case status == http.StatusTooManyRequests:
			c.rateLimited429.Add(1)
			var rl rateLimitBody
			_ = json.Unmarshal(body, &rl)
			if rl.Global || header.Get("X-RateLimit-Global") == "true" {
				c.globalLimitHits.Add(1)
				c.halveInterval()
			}
			lastErr = fmt.Errorf("discord: rate limited (429) on %s", path)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("%s: gave up after %d attempts: %w", resourceID, maxAttempts, lastErr)
			}
			c.sleep(secondsToDuration(rl.RetryAfter) + c.jitter())
			continue
		case status >= 500:
			lastErr = fmt.Errorf("discord: server error (%d) on %s", status, path)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("%s: gave up after %d attempts: %w", resourceID, maxAttempts, lastErr)
			}
			c.sleep(backoff(attempt) + c.jitter())
			continue
		case status >= 400:
			return nil, fmt.Errorf("discord: request %s failed: %d: %s", path, status, string(body))
		}

		c.updateRateState(header)
		return body, nil
	}
	return nil, lastErr
}

func (c *Client) halveInterval() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentInterval /= 2
}

func (c *Client) updateRateState(header http.Header) {
	if header.Get("X-RateLimit-Remaining") != "0" {
		return
	}
	resetAfter, err := strconv.ParseFloat(header.Get("X-RateLimit-Reset-After"), 64)
	if err != nil {
		return
	}
	c.mu.Lock()
	c.pendingSleep = secondsToDuration(resetAfter)
	c.mu.Unlock()
}

func secondsToDuration(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

func backoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
}

// do issues one HTTP GET, returning the raw body, status, and headers.
func (c *Client) do(ctx context.Context, path string, query url.Values) ([]byte, int, http.Header, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("User-Agent", userAgent)

	c.requestsMade.Add(1)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, nil, err
	}
	return body, resp.StatusCode, resp.Header, nil
}
