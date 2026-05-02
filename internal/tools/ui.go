package tools

import (
	"context"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/androidcli"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// RegisterUI exposes the screen-snapshot helpers used by tests for visual
// regression and debugging. Element discovery and waiting belong to the
// testing layer (find_node / find_all_nodes / wait_until_*).
func RegisterUI(s *mcp.Server, d *Deps) {
	type captureArgs struct {
		DeviceArg
		DisplayID string `json:"displayId,omitempty" jsonschema:"non-default display ID for multi-display devices"`
		SaveTo    string `json:"saveTo,omitempty" jsonschema:"optional host file path; extension chooses encoding (.png|.jpg|.jpeg)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_capture",
		Description: "Take a screenshot of the device. Returns the PNG inline; if saveTo is provided, also writes to disk.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args captureArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		png, err := d.Screenshot.Capture(ctx, dev, args.DisplayID)
		if err != nil {
			return errResult(err)
		}
		content := []mcp.Content{
			&mcp.ImageContent{Data: png, MIMEType: "image/png"},
		}
		if args.SaveTo != "" {
			path, err := ui.Save(png, args.SaveTo)
			if err != nil {
				return errResult(err)
			}
			content = append(content, &mcp.TextContent{Text: "saved to " + path})
		}
		return &mcp.CallToolResult{Content: content}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_layout",
		Description: "Return the on-screen UI hierarchy as a flat list of interactive elements with bounds. Convenience wrapper over the layout source — for fine-grained selection prefer find_node / find_all_nodes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		root, err := d.Layout.Tree(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		flat := ui.Flatten(root)
		if flat == nil {
			flat = []ui.Element{}
		}
		return jsonResult(map[string]any{"items": flat})
	})

	type resolveArgs struct {
		DeviceArg
		Label string `json:"label" jsonschema:"the visible label or text to locate on screen"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_resolve",
		Description: "Resolve a visible label to coordinates using the Android agent CLI's `screen resolve` (LLM-friendly visual lookup).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args resolveArgs) (*mcp.CallToolResult, any, error) {
		if d.AndroidCLI == nil || !d.AndroidCLI.Available() {
			return errResult(androidcli.ErrNotInstalled)
		}
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		png, err := d.Screenshot.Capture(ctx, dev, "")
		if err != nil {
			return errResult(err)
		}
		path, err := ui.WriteTempScreenshotForResolve("screen-resolve", png)
		if err != nil {
			return errResult(err)
		}
		defer os.Remove(path)
		res, err := d.AndroidCLI.Run(ctx, "screen", "resolve", "--screenshot="+path, "--string="+args.Label)
		if err != nil {
			return errResult(err)
		}
		return textResult(string(res.Stdout))
	})

	type diffArgs struct {
		PathA        string  `json:"pathA" jsonschema:"baseline PNG path"`
		PathB        string  `json:"pathB" jsonschema:"comparison PNG path"`
		DiffOutput   string  `json:"diffOutput,omitempty" jsonschema:"if set, write a diff PNG highlighting changed pixels"`
		Tolerance    int     `json:"tolerance,omitempty" jsonschema:"per-channel tolerance 0..255 (default 0)"`
		ThresholdPct float64 `json:"thresholdPct,omitempty" jsonschema:"max acceptable mismatch percentage (default 0)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_diff",
		Description: "Pixel-level visual regression comparison between two PNGs. Optionally writes a red-overlay diff image.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args diffArgs) (*mcp.CallToolResult, any, error) {
		res, err := ui.Diff(args.PathA, args.PathB, args.DiffOutput, args.Tolerance, args.ThresholdPct)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})
}
