package archive

import "path/filepath"

// WriteRunReport writes a mine run's report file at root/_runs/name,
// overwriting any existing file of the same name — e.g. a re-run at the
// same injected clock instant. Writes are atomic.
func WriteRunReport(root, name, content string) error {
	return atomicWrite(filepath.Join(root, "_runs", name), []byte(content))
}
