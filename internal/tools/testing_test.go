package tools_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/tools"
)

// TestAddDeviceTool_MalformedDeviceArg_ReturnsInvalidArgumentsError guards
// the Priority-0 fix: addDeviceTool used to silently swallow a malformed
// `device` argument and fall through to auto-resolution instead of
// reporting "invalid arguments", unlike its sibling addMatcherTool. A
// type-mismatched `device` field (a number instead of a string) must now
// produce an error result without ever reaching device resolution — a
// zero-value Deps would panic on a nil Resolver if it did.
func TestAddDeviceTool_MalformedDeviceArg_ReturnsInvalidArgumentsError(t *testing.T) {
	ctx := context.Background()

	impl := &mcp.Implementation{Name: "velocity-test-mobile", Version: "test"}
	server := mcp.NewServer(impl, nil)
	client := mcp.NewClient(impl, nil)

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

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "press_back_unconditionally",
		Arguments: map[string]any{"device": 123}, // wrong type: should be a string
	})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected an error result for a malformed device argument, got: %+v", res)
	}
}
