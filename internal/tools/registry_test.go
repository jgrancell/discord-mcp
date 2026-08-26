package tools_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jgrancell/discord-mcp/internal/tools"
)

// Behavior 52: ListTools returns tools.Count() tools, each with a
// non-empty description.
func TestRegister_ListTools_ReturnsCountToolsWithDescriptions(t *testing.T) {
	t.Parallel()
	cs := session(t)

	res, err := cs.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if got := len(res.Tools); got != tools.Count() {
		t.Fatalf("listed tools = %d, want %d (tools.Count())", got, tools.Count())
	}
	for _, tool := range res.Tools {
		if tool.Description == "" {
			t.Errorf("tool %q has no description", tool.Name)
		}
	}
}
