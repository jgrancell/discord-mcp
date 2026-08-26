package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jgrancell/discord-mcp/internal/render"
)

const frontmatterTimeLayout = "2006-01-02T15:04:05Z"

// WriteWindow writes a window file's frontmatter and body. A complete
// window is immutable: if the existing file is already complete, this is a
// no-op reporting wrote=false. Writes are atomic.
func WriteWindow(path string, meta render.WindowMeta, body string) (wrote bool, err error) {
	if existing, _, err := ReadWindow(path); err == nil && existing.Complete {
		return false, nil
	}
	content := render.WindowFrontmatter(meta) + body
	if err := atomicWrite(path, []byte(content)); err != nil {
		return false, err
	}
	return true, nil
}

// ReadWindow reads a window file's frontmatter and body back.
func ReadWindow(path string) (render.WindowMeta, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return render.WindowMeta{}, "", err
	}
	fields, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return render.WindowMeta{}, "", fmt.Errorf("read window %s: %w", path, err)
	}
	meta, err := windowMetaFromFields(fields)
	if err != nil {
		return render.WindowMeta{}, "", fmt.Errorf("read window %s: %w", path, err)
	}
	return meta, body, nil
}

// splitFrontmatter separates a "---\n...\n---\n" YAML-ish block from the
// body that follows it, into a flat key/value map. List items (e.g.
// applied_tags entries) are skipped; WindowMeta has no list fields.
func splitFrontmatter(content string) (map[string]string, string, error) {
	const fence = "---\n"
	if !strings.HasPrefix(content, fence) {
		return nil, "", fmt.Errorf("missing opening frontmatter fence")
	}
	rest := content[len(fence):]
	idx := strings.Index(rest, "\n---\n")
	if idx == -1 {
		return nil, "", fmt.Errorf("missing closing frontmatter fence")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(rest[:idx], "\n") {
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		fields[key] = val
	}
	body := rest[idx+len("\n---\n"):]
	return fields, body, nil
}

func windowMetaFromFields(f map[string]string) (render.WindowMeta, error) {
	var m render.WindowMeta
	var err error
	m.GuildID = unquote(f["guild_id"])
	m.GuildName = f["guild_name"]
	m.ChannelID = unquote(f["channel_id"])
	m.ChannelName = f["channel_name"]
	m.ChannelType = f["channel_type"]
	m.ParentCategory = f["parent_category"]
	m.Window = f["window"]
	if m.WindowStart, err = time.Parse(frontmatterTimeLayout, f["window_start"]); err != nil {
		return m, fmt.Errorf("window_start: %w", err)
	}
	if m.WindowEnd, err = time.Parse(frontmatterTimeLayout, f["window_end"]); err != nil {
		return m, fmt.Errorf("window_end: %w", err)
	}
	m.FirstMessageID = unquote(f["first_message_id"])
	m.LastMessageID = unquote(f["last_message_id"])
	if m.MessageCount, err = strconv.Atoi(f["message_count"]); err != nil {
		return m, fmt.Errorf("message_count: %w", err)
	}
	m.Complete = f["complete"] == "true"
	if m.MinedAt, err = time.Parse(frontmatterTimeLayout, f["mined_at"]); err != nil {
		return m, fmt.Errorf("mined_at: %w", err)
	}
	m.MinerVersion = f["miner_version"]
	if m.Format, err = strconv.Atoi(f["format"]); err != nil {
		return m, fmt.Errorf("format: %w", err)
	}
	return m, nil
}

func unquote(s string) string {
	return strings.Trim(s, `"`)
}

// atomicWrite renders data to a temp file in dir(path), fsyncs it, and
// renames it over path. A crash or failure mid-write leaves the previous
// file intact and no temp file behind.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort: once the rename below succeeds, tmpName no longer
	// exists and this becomes a harmless no-op failure. On any earlier
	// return, it removes the abandoned temp file.
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		// Best-effort close on an already-failing path — the write error
		// above is what the caller needs.
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
