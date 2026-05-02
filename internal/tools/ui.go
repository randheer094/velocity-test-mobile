package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// RegisterUI registers screen capture, layout, resolve, assertions, and diff tools.
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
		Description: "Return the on-screen UI hierarchy as a flat list of interactive elements with bounds. Prefers `android layout` JSON; falls back to UIAutomator.",
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
		return jsonResult(flat)
	})

	type resolveArgs struct {
		DeviceArg
		Label string `json:"label" jsonschema:"the visible label or text to locate on screen"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_resolve",
		Description: "Resolve a visible label to coordinates using `android screen resolve` (LLM-friendly visual lookup). Requires the Android agent CLI.",
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

	type waitArgs struct {
		DeviceArg
		Text        string `json:"text,omitempty" jsonschema:"text substring or /regex/"`
		ContentDesc string `json:"contentDesc,omitempty"`
		ResourceID  string `json:"resourceId,omitempty"`
		Class       string `json:"class,omitempty"`
		TimeoutMs   int    `json:"timeoutMs,omitempty" jsonschema:"max wait in milliseconds (default 10000)"`
		IntervalMs  int    `json:"intervalMs,omitempty" jsonschema:"poll interval in milliseconds (default 250)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "wait_for_element",
		Description: "Poll the UI hierarchy until an element matching the predicate appears, or timeout. Returns the matched element with bounds.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args waitArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		timeout := args.TimeoutMs
		if timeout <= 0 {
			timeout = 10000
		}
		interval := args.IntervalMs
		if interval <= 0 {
			interval = 250
		}
		predicate := ui.Predicate{
			Text:        args.Text,
			ContentDesc: args.ContentDesc,
			ResourceID:  args.ResourceID,
			Class:       args.Class,
		}
		if predicate == (ui.Predicate{}) {
			return errResult(fmt.Errorf("at least one of text/contentDesc/resourceId/class is required"))
		}
		deadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
		attempts := 0
		for {
			attempts++
			root, err := d.Layout.Tree(ctx, dev)
			if err == nil {
				if got, ok := ui.Match(root, predicate); ok {
					return jsonResult(map[string]any{
						"matched":  true,
						"attempts": attempts,
						"element":  got,
					})
				}
			}
			if time.Now().After(deadline) {
				return jsonResult(map[string]any{
					"matched":  false,
					"attempts": attempts,
				})
			}
			select {
			case <-ctx.Done():
				return errResult(ctx.Err())
			case <-time.After(time.Duration(interval) * time.Millisecond):
			}
		}
	})

	type assertArgs struct {
		DeviceArg
		Text      string `json:"text" jsonschema:"text substring or /regex/ that must appear"`
		TimeoutMs int    `json:"timeoutMs,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "assert_text_visible",
		Description: "Assert that a substring (or /regex/) is visible on screen within the timeout.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args assertArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		timeout := args.TimeoutMs
		if timeout <= 0 {
			timeout = 5000
		}
		deadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
		attempts := 0
		for {
			attempts++
			root, err := d.Layout.Tree(ctx, dev)
			if err == nil {
				if _, ok := ui.Match(root, ui.Predicate{Text: args.Text}); ok {
					return jsonResult(map[string]any{"ok": true, "attempts": attempts})
				}
			}
			if time.Now().After(deadline) {
				return jsonResult(map[string]any{"ok": false, "attempts": attempts})
			}
			select {
			case <-ctx.Done():
				return errResult(ctx.Err())
			case <-time.After(250 * time.Millisecond):
			}
		}
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
		Description: "Pixel-level compare of two PNGs. Optionally writes a red-overlay diff image. Returns mismatch counts and percentage.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args diffArgs) (*mcp.CallToolResult, any, error) {
		res, err := ui.Diff(args.PathA, args.PathB, args.DiffOutput, args.Tolerance, args.ThresholdPct)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})
}
