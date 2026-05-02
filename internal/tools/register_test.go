package tools_test

import (
	"context"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/tools"
)

// TestRegisterAll_MatchesCatalog connects an in-memory MCP server with
// RegisterAll wired up and asserts that the live tools/list response is
// exactly the set of names declared in Catalog(). It catches drift in either
// direction: a tool added to RegisterAll without updating Catalog (and thus
// missing from `--list-tools`), or a name in Catalog that no Register* func
// actually registers.
func TestRegisterAll_MatchesCatalog(t *testing.T) {
	ctx := context.Background()

	impl := &mcp.Implementation{Name: "velocity-test-mobile", Version: "test"}
	server := mcp.NewServer(impl, nil)
	client := mcp.NewClient(impl, nil)

	// Handlers close over Deps lazily; registration itself doesn't dereference
	// any client, so a zero-value Deps is sufficient for listing tools.
	tools.RegisterAll(server, &tools.Deps{})

	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer ss.Close()
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer cs.Close()

	got := map[string]int{}
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("iterating tools: %v", err)
		}
		got[tool.Name]++
	}

	for name, n := range got {
		if n > 1 {
			t.Errorf("tool %q registered %d times", name, n)
		}
	}

	want := map[string]struct{}{}
	for _, name := range tools.Catalog() {
		want[name] = struct{}{}
	}

	var missing, extra []string
	for name := range want {
		if _, ok := got[name]; !ok {
			missing = append(missing, name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("Catalog lists %d tool(s) not registered by RegisterAll: %v", len(missing), missing)
	}
	if len(extra) > 0 {
		t.Errorf("RegisterAll registers %d tool(s) not in Catalog: %v", len(extra), extra)
	}
}

// TestCatalog_NoDuplicates guards against accidental duplicate entries in the
// hand-maintained Catalog() list.
func TestCatalog_NoDuplicates(t *testing.T) {
	seen := map[string]struct{}{}
	for _, name := range tools.Catalog() {
		if _, dup := seen[name]; dup {
			t.Errorf("duplicate name in Catalog(): %q", name)
		}
		seen[name] = struct{}{}
	}
}
