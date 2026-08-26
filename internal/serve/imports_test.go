package serve_test

import (
	"os/exec"
	"strings"
	"testing"
)

// Behavior 78: the serve package's transitive imports include neither
// internal/archive nor internal/mine — a compile-time-ish assertion that
// serve cannot persist (contract invariant 3: "No persistence in serve").
//
// This is an architectural fitness function: it is expected to already
// pass (there is nothing yet to violate it) and must keep passing as the
// package is implemented. A future change that makes serve import archive
// or mine is exactly the regression this test exists to catch.
func TestServe_TransitiveImports_ExcludeArchiveAndMine(t *testing.T) {
	t.Parallel()
	const module = "github.com/jgrancell/discord-mcp"

	out, err := exec.Command("go", "list", "-deps", module+"/internal/serve").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps internal/serve: %v\n%s", err, out)
	}

	forbidden := []string{module + "/internal/archive", module + "/internal/mine"}
	for _, line := range strings.Split(string(out), "\n") {
		for _, f := range forbidden {
			if line == f {
				t.Errorf("internal/serve transitively imports %s, which is forbidden (serve must not persist)", f)
			}
		}
	}
}
