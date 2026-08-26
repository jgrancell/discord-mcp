package archive

import "github.com/jgrancell/discord-mcp/internal/render"

// WriteThread writes a thread file's frontmatter and body, overwriting
// whatever was there. Threads are bounded conversations re-derived from
// live Discord state each run, so unlike windows they carry no
// complete/immutable flag.
func WriteThread(path string, meta render.ThreadMeta, body string) error {
	content := render.ThreadFrontmatter(meta) + body
	return atomicWrite(path, []byte(content))
}
