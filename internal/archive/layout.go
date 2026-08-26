package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jgrancell/discord-mcp/internal/render"
)

// WindowPath returns the path to a channel's window file for the given
// "YYYY-MM" window string.
func WindowPath(channelDir, window string) string {
	return filepath.Join(channelDir, window+".md")
}

// ThreadPath returns the path to a thread file under a channel's threads/
// directory.
func ThreadPath(channelDir, slug, threadID string) string {
	return filepath.Join(channelDir, "threads", slug+"-"+threadID+".md")
}

var slugInvalidRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a Discord channel or thread name into a filesystem-safe
// slug for directory and file naming.
func Slugify(name string) string {
	s := slugInvalidRE.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "channel"
	}
	return s
}

// ResolveChannelDir returns the directory a channel archives into: an
// existing directory keyed by the channel ID if one exists (so a channel
// renamed upstream keeps writing to the same directory), or a new
// <slug>-<id> directory otherwise.
func ResolveChannelDir(root, channelID, slug string) (string, error) {
	channelsDir := filepath.Join(root, "channels")
	entries, err := os.ReadDir(channelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Join(channelsDir, slug+"-"+channelID), nil
		}
		return "", fmt.Errorf("resolve channel dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), "-"+channelID) {
			return filepath.Join(channelsDir, e.Name()), nil
		}
	}
	return filepath.Join(channelsDir, slug+"-"+channelID), nil
}

// listWindowMetas reads every window file's frontmatter in channelDir,
// sorted ascending by window string.
func listWindowMetas(channelDir string) ([]render.WindowMeta, error) {
	entries, err := os.ReadDir(channelDir)
	if err != nil {
		return nil, err
	}
	var metas []render.WindowMeta
	for _, e := range entries {
		if e.IsDir() || e.Name() == "_channel.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		meta, _, err := ReadWindow(filepath.Join(channelDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read window %s: %w", e.Name(), err)
		}
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Window < metas[j].Window })
	return metas, nil
}
