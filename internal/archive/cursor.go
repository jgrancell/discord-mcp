package archive

import (
	"fmt"
	"os"
)

// ResumeCursor derives a channel's resume cursor: max(last_message_id)
// across its window files. A channel directory with no window files (or
// that does not exist yet) yields "0" — start from the beginning.
func ResumeCursor(dir string) (string, error) {
	metas, err := listWindowMetas(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "0", nil
		}
		return "", fmt.Errorf("resume cursor: %w", err)
	}
	if len(metas) == 0 {
		return "0", nil
	}
	max := metas[0].LastMessageID
	for _, m := range metas[1:] {
		if snowflakeLess(max, m.LastMessageID) {
			max = m.LastMessageID
		}
	}
	return max, nil
}

// snowflakeLess reports whether a < b as Discord snowflake IDs: numeric
// comparison via decimal digit strings (shorter is smaller; equal length
// compares lexicographically).
func snowflakeLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
