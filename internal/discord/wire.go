package discord

import "time"

// wireMessage is Discord's REST message shape, decoded then converted to
// the domain Message type.
type wireMessage struct {
	ID                string           `json:"id"`
	ChannelID         string           `json:"channel_id"`
	GuildID           string           `json:"guild_id"`
	Content           string           `json:"content"`
	Timestamp         string           `json:"timestamp"`
	EditedTimestamp   *string          `json:"edited_timestamp"`
	Type              int              `json:"type"`
	Author            wireAuthor       `json:"author"`
	WebhookID         string           `json:"webhook_id"`
	Reactions         []wireReaction   `json:"reactions"`
	Attachments       []wireAttachment `json:"attachments"`
	Embeds            []wireEmbed      `json:"embeds"`
	StickerItems      []wireSticker    `json:"sticker_items"`
	ReferencedMessage *wireMessage     `json:"referenced_message"`
}

type wireAuthor struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	Bot        bool    `json:"bot"`
}

type wireReaction struct {
	Emoji wireEmoji `json:"emoji"`
	Count int       `json:"count"`
}

type wireEmoji struct {
	Name string `json:"name"`
}

type wireAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
}

type wireEmbed struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type wireSticker struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	FormatType int    `json:"format_type"`
}

func (a wireAuthor) toAuthor(webhookID string) Author {
	display := a.Username
	if a.GlobalName != nil && *a.GlobalName != "" {
		display = *a.GlobalName
	}
	return Author{
		ID:          a.ID,
		Username:    a.Username,
		DisplayName: display,
		Bot:         a.Bot,
		Webhook:     webhookID != "",
	}
}

func (m wireMessage) toMessage() Message {
	out := Message{
		ID:        m.ID,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
		Author:    m.Author.toAuthor(m.WebhookID),
		Content:   m.Content,
		Type:      m.Type,
	}
	if ts, err := time.Parse(time.RFC3339Nano, m.Timestamp); err == nil {
		out.Timestamp = ts.UTC()
	}
	if m.EditedTimestamp != nil {
		if ts, err := time.Parse(time.RFC3339Nano, *m.EditedTimestamp); err == nil {
			ts = ts.UTC()
			out.EditedTimestamp = &ts
		}
	}
	for _, r := range m.Reactions {
		out.Reactions = append(out.Reactions, Reaction{Emoji: r.Emoji.Name, Count: r.Count})
	}
	for _, a := range m.Attachments {
		out.Attachments = append(out.Attachments, Attachment(a))
	}
	for _, e := range m.Embeds {
		out.Embeds = append(out.Embeds, Embed(e))
	}
	for _, s := range m.StickerItems {
		out.Stickers = append(out.Stickers, Sticker{Name: s.Name, URL: stickerURL(s)})
	}
	if m.ReferencedMessage != nil {
		ref := m.ReferencedMessage.toMessage()
		out.ReferencedMessage = &ref
	}
	return out
}

func stickerURL(s wireSticker) string {
	ext := "png"
	switch s.FormatType {
	case 3:
		ext = "json"
	case 4:
		ext = "gif"
	}
	return "https://cdn.discordapp.com/stickers/" + s.ID + "." + ext
}
