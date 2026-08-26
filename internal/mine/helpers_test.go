package mine_test

import (
	"strconv"
	"time"
)

// discordMessageJSON builds a minimal Discord API message object at the
// given RFC3339 timestamp.
func discordMessageJSON(id, channelID, content, timestamp string) string {
	return `{"id":"` + id + `","channel_id":"` + channelID + `","content":"` + content + `",` +
		`"timestamp":"` + timestamp + `","type":0,` +
		`"author":{"id":"999000111222333444","username":"janak","global_name":"Janak","bot":false}}`
}

// channelJSON builds a minimal Discord API guild-channel object.
// discordType is Discord's raw numeric channel type (0 = text, 5 =
// announcement, 15 = forum).
func channelJSON(id, name string, discordType int) string {
	return `{"id":"` + id + `","name":"` + name + `","type":` + strconv.Itoa(discordType) + `,"guild_id":"1100000000000000000"}`
}

const guildID = "1100000000000000000"

var fixedNow = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
