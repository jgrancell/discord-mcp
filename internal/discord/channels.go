package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type wireChannel struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Type           int             `json:"type"`
	GuildID        string          `json:"guild_id"`
	ParentID       string          `json:"parent_id"`
	LastMessageID  string          `json:"last_message_id"`
	ThreadMetadata *wireThreadMeta `json:"thread_metadata"`
	MessageCount   int             `json:"message_count"`
	OwnerID        string          `json:"owner_id"`
	AppliedTags    []string        `json:"applied_tags"`
}

type wireThreadMeta struct {
	Archived         bool   `json:"archived"`
	Locked           bool   `json:"locked"`
	ArchiveTimestamp string `json:"archive_timestamp"`
}

func (w wireChannel) toChannel() Channel {
	c := Channel{
		ID: w.ID, Name: w.Name, Type: w.Type, GuildID: w.GuildID,
		ParentID: w.ParentID, LastMessageID: w.LastMessageID,
		MessageCount: w.MessageCount, OwnerID: w.OwnerID, AppliedTags: w.AppliedTags,
	}
	if w.ThreadMetadata != nil {
		c.ThreadMetadata = &ThreadMetadata{Archived: w.ThreadMetadata.Archived, Locked: w.ThreadMetadata.Locked}
	}
	return c
}

func (w wireChannel) toThread() Thread {
	t := Thread{
		ID: w.ID, Name: w.Name, ParentID: w.ParentID, GuildID: w.GuildID,
		MessageCount: w.MessageCount, OwnerID: w.OwnerID, AppliedTags: w.AppliedTags,
	}
	if w.ThreadMetadata != nil {
		t.Archived = w.ThreadMetadata.Archived
		t.Locked = w.ThreadMetadata.Locked
	}
	return t
}

// Channels returns a guild's channels (text, announcement, forum,
// category, and any other guild-scoped channel).
func (c *Client) Channels(ctx context.Context, guildID string) ([]Channel, error) {
	body, err := c.get(ctx, "/guilds/"+guildID+"/channels", nil, guildID)
	if err != nil {
		return nil, err
	}
	var wire []wireChannel
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode channels: %w", err)
	}
	out := make([]Channel, len(wire))
	for i, w := range wire {
		out[i] = w.toChannel()
	}
	return out, nil
}

// Channel fetches a single channel or thread by ID. Discord returns the
// same object shape for both; ThreadMetadata is non-nil for a thread.
func (c *Client) Channel(ctx context.Context, id string) (Channel, error) {
	body, err := c.get(ctx, "/channels/"+id, nil, id)
	if err != nil {
		return Channel{}, err
	}
	var w wireChannel
	if err := json.Unmarshal(body, &w); err != nil {
		return Channel{}, fmt.Errorf("decode channel %s: %w", id, err)
	}
	return w.toChannel(), nil
}

type wireThreadList struct {
	Threads []wireChannel `json:"threads"`
	HasMore bool          `json:"has_more"`
}

// ActiveThreads returns every active thread in a guild.
func (c *Client) ActiveThreads(ctx context.Context, guildID string) ([]Thread, error) {
	body, err := c.get(ctx, "/guilds/"+guildID+"/threads/active", nil, guildID)
	if err != nil {
		return nil, err
	}
	var list wireThreadList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decode active threads: %w", err)
	}
	return toThreads(list.Threads), nil
}

// ArchivedThreads returns a channel's archived threads (public, or private
// if private is true — the latter requires Manage Threads).
func (c *Client) ArchivedThreads(ctx context.Context, channelID string, private bool) ([]Thread, error) {
	kind := "public"
	if private {
		kind = "private"
	}
	path := "/channels/" + channelID + "/threads/archived/" + kind

	var out []Thread
	before := ""
	for {
		q := url.Values{}
		if before != "" {
			q.Set("before", before)
		}
		body, err := c.get(ctx, path, q, channelID)
		if err != nil {
			return nil, err
		}
		var list wireThreadList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("decode archived threads: %w", err)
		}
		out = append(out, toThreads(list.Threads)...)
		if !list.HasMore || len(list.Threads) == 0 {
			return out, nil
		}
		last := list.Threads[len(list.Threads)-1]
		if last.ThreadMetadata == nil || last.ThreadMetadata.ArchiveTimestamp == "" {
			return out, nil
		}
		before = last.ThreadMetadata.ArchiveTimestamp
	}
}

func toThreads(wire []wireChannel) []Thread {
	out := make([]Thread, len(wire))
	for i, w := range wire {
		out[i] = w.toThread()
	}
	return out
}
