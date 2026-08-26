package render

import (
	"regexp"
)

// MentionMap resolves raw Discord mention IDs to display names for
// rendering. Unresolved IDs are left as the original raw mention.
type MentionMap struct {
	Users    map[string]string
	Channels map[string]string
	Roles    map[string]string
}

var (
	userMentionRE    = regexp.MustCompile(`<@!?(\d+)>`)
	channelMentionRE = regexp.MustCompile(`<#(\d+)>`)
	roleMentionRE    = regexp.MustCompile(`<@&(\d+)>`)
	emojiRE          = regexp.MustCompile(`<a?:(\w+):\d+>`)
)

// resolveMentions expands <@id>/<#id>/<@&id>/<:name:id> tokens against m,
// leaving unresolvable IDs verbatim.
func resolveMentions(content string, m MentionMap) string {
	content = roleMentionRE.ReplaceAllStringFunc(content, func(match string) string {
		id := roleMentionRE.FindStringSubmatch(match)[1]
		if name, ok := m.Roles[id]; ok {
			return "@" + name
		}
		return match
	})
	content = userMentionRE.ReplaceAllStringFunc(content, func(match string) string {
		id := userMentionRE.FindStringSubmatch(match)[1]
		if name, ok := m.Users[id]; ok {
			return "@" + name
		}
		return match
	})
	content = channelMentionRE.ReplaceAllStringFunc(content, func(match string) string {
		id := channelMentionRE.FindStringSubmatch(match)[1]
		if name, ok := m.Channels[id]; ok {
			return "#" + name
		}
		return match
	})
	content = emojiRE.ReplaceAllStringFunc(content, func(match string) string {
		name := emojiRE.FindStringSubmatch(match)[1]
		return ":" + name + ":"
	})
	return content
}
