package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterSystem exposes the device-state knobs that real Espresso/Compose
// tests routinely require — animation control (mandatory for stable tests)
// and orientation lock. All other system surface (network, doze, time,
// location, screen wake/lock, etc.) is intentionally absent from this
// testing-only server.
func RegisterSystem(s *mcp.Server, d *Deps) {
	type animArgs struct {
		DeviceArg
		Scale float64 `json:"scale" jsonschema:"animation scale (0 disables; 1 is default; max 10) — set to 0 before running UI tests"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "animations_set",
		Description: "Set window/transition/animator scales together. Required test setup: scale 0 disables animations, eliminating a major source of flakiness.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args animArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Animations.Set(ctx, dev, args.Scale); err != nil {
			return errResult(err)
		}
		return textResult("animation scales set")
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "animations_get",
		Description: "Read the three global animation scales (window/transition/animator).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		st, err := d.Animations.Get(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(st)
	})
}
