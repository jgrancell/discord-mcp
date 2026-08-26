package discord

// ChannelTypeName maps Discord's raw numeric channel type to the short
// name used in the archive and tool output.
func ChannelTypeName(t int) string {
	switch t {
	case 0:
		return "text"
	case 2:
		return "voice"
	case 4:
		return "category"
	case 5:
		return "announcement"
	case 10:
		return "announcement_thread"
	case 11:
		return "public_thread"
	case 12:
		return "private_thread"
	case 13:
		return "stage_voice"
	case 15:
		return "forum"
	default:
		return "unknown"
	}
}
