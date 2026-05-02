package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/diagnostics"
)

// RegisterDiagnostics exposes only the test-debug helpers: logcat tail
// and clear. Performance dumpsys / atrace / perfetto live in a separate,
// non-testing surface and are not registered here.
func RegisterDiagnostics(s *mcp.Server, d *Deps) {
	type logArgs struct {
		DeviceArg
		Package  string `json:"package,omitempty" jsonschema:"limit logs to this package's PID"`
		Tag      string `json:"tag,omitempty"`
		Priority string `json:"priority,omitempty" jsonschema:"V|D|I|W|E|F|S"`
		MaxLines int    `json:"maxLines,omitempty" jsonschema:"default 1000"`
		Since    string `json:"since,omitempty" jsonschema:"-T value, e.g. 'MM-DD HH:MM:SS.SSS'"`
		Regex    string `json:"regex,omitempty" jsonschema:"post-filter regex applied to each line"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "logcat_tail",
		Description: "Return recent logcat lines (test debug aid). Filter by package PID, tag, priority, regex.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args logArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		lines, err := d.Logs.Tail(ctx, dev, diagnostics.LogOptions{
			Package:  args.Package,
			Tag:      args.Tag,
			Priority: args.Priority,
			MaxLines: args.MaxLines,
			Since:    args.Since,
			Regex:    args.Regex,
		})
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"count": len(lines), "lines": lines})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "logcat_clear",
		Description: "Clear logcat buffers (`logcat -c`) — useful between tests.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptrTrue()},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Logs.Clear(ctx, dev); err != nil {
			return errResult(err)
		}
		return textResult("logcat cleared")
	})
}
