package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterDevice registers device discovery & info tools.
func RegisterDevice(s *mcp.Server, d *Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "device_list",
		Description: "List Android devices visible to adb (physical and emulator), including their state.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		devs, err := d.Resolver.List(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(devs)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "device_get_screen_size",
		Description: "Get the screen size and density of the device.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		size, err := d.Screen.Get(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(size)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "device_get_orientation",
		Description: "Get the current screen orientation (portrait or landscape).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		o, err := d.Screen.GetOrientation(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]string{"orientation": o})
	})

	type setOrientationArgs struct {
		DeviceArg
		Orientation string `json:"orientation" jsonschema:"portrait or landscape"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "device_set_orientation",
		Description: "Lock the screen orientation to portrait or landscape.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptrFalse()},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args setOrientationArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Screen.SetOrientation(ctx, dev, args.Orientation); err != nil {
			return errResult(err)
		}
		return textResult("orientation set to " + args.Orientation)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "device_get_props",
		Description: "Return a curated subset of device properties: model, brand, manufacturer, SDK level, release, fingerprint, ABI list.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		p, err := d.Resolver.GetProps(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(p)
	})
}
