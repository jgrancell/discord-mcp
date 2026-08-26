package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type wireUser struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	Bot        bool    `json:"bot"`
}

func (w wireUser) toUser(nick string) User {
	display := w.Username
	if w.GlobalName != nil && *w.GlobalName != "" {
		display = *w.GlobalName
	}
	return User{ID: w.ID, Username: w.Username, DisplayName: display, Bot: w.Bot, Nick: nick}
}

type wireMember struct {
	User wireUser `json:"user"`
	Nick *string  `json:"nick"`
}

func (w wireMember) toUser() User {
	nick := ""
	if w.Nick != nil {
		nick = *w.Nick
	}
	return w.User.toUser(nick)
}

// SearchMembers searches a guild's members by username, display name, or
// nickname prefix.
func (c *Client) SearchMembers(ctx context.Context, guildID, query string) ([]User, error) {
	q := url.Values{}
	q.Set("query", query)
	body, err := c.get(ctx, "/guilds/"+guildID+"/members/search", q, guildID)
	if err != nil {
		return nil, err
	}
	var wire []wireMember
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode member search: %w", err)
	}
	out := make([]User, len(wire))
	for i, w := range wire {
		out[i] = w.toUser()
	}
	return out, nil
}

// User fetches a Discord account by ID.
func (c *Client) User(ctx context.Context, id string) (User, error) {
	body, err := c.get(ctx, "/users/"+id, nil, id)
	if err != nil {
		return User{}, err
	}
	var w wireUser
	if err := json.Unmarshal(body, &w); err != nil {
		return User{}, fmt.Errorf("decode user %s: %w", id, err)
	}
	return w.toUser(""), nil
}
