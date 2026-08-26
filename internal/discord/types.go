package discord

import "time"

// Author is a message's sender: a user, a bot, or a webhook.
type Author struct {
	ID          string
	Username    string
	DisplayName string
	Bot         bool
	Webhook     bool
}

// Reaction is an aggregate emoji reaction on a message.
type Reaction struct {
	Emoji string
	Count int
}

// Attachment is a file attached to a message. Metadata and URL only.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	URL         string
}

// Embed is a rich embed attached to a message.
type Embed struct {
	Title       string
	URL         string
	Description string
}

// Sticker is a sticker sent with a message.
type Sticker struct {
	Name string
	URL  string
}

// ThreadPointer marks the position of a thread-creating message and links
// to the thread's own archive file.
type ThreadPointer struct {
	Title        string
	Path         string
	MessageCount int
}

// Message is a single Discord message, decoded from the REST API into the
// shape render and archive need.
type Message struct {
	ID                string
	ChannelID         string
	GuildID           string
	Author            Author
	Timestamp         time.Time
	Content           string
	Type              int
	EditedTimestamp   *time.Time
	ReferencedMessage *Message
	Reactions         []Reaction
	Attachments       []Attachment
	Embeds            []Embed
	Stickers          []Sticker
	ThreadStarted     *ThreadPointer
}

// ThreadMetadata carries the thread-specific state Discord nests under
// "thread_metadata" on a channel object.
type ThreadMetadata struct {
	Archived bool
	Locked   bool
}

// Channel is a Discord guild channel: text, announcement, forum, or
// category. GET /channels/{id} returns this same shape for a thread,
// discriminated by ThreadMetadata being non-nil.
type Channel struct {
	ID             string
	Name           string
	Type           int
	GuildID        string
	ParentID       string
	LastMessageID  string
	ThreadMetadata *ThreadMetadata
	MessageCount   int
	OwnerID        string
	AppliedTags    []string
}

// Thread is a Discord thread or forum post.
type Thread struct {
	ID           string
	Name         string
	ParentID     string
	GuildID      string
	Archived     bool
	Locked       bool
	MessageCount int
	OwnerID      string
	AppliedTags  []string
}

// User is a Discord account, optionally with a guild nickname.
type User struct {
	ID          string
	Username    string
	DisplayName string
	Bot         bool
	Nick        string
}
