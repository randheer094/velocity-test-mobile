package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/diagnostics"
)

// RegisterDiagnostics registers logcat, dumpsys, and trace tools.
func RegisterDiagnostics(s *mcp.Server, d *Deps) {
	type logArgs struct {
		DeviceArg
		Package  string `json:"package,omitempty" jsonschema:"limit logs to this package's PID"`
		Tag      string `json:"tag,omitempty"`
		Priority string `json:"priority,omitempty" jsonschema:"V|D|I|W|E|F|S"`
		MaxLines int    `json:"maxLines,omitempty" jsonschema:"default 1000"`
		Since    string `json:"since,omitempty" jsonschema:"-T value, e.g. 'MM-DD HH:MM:SS.SSS'"`
		Regex    string `json:"regex,omitempty" jsonschema:"post-filter regex, applied to each line"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "logcat_tail",
		Description: "Return recent logcat lines, optionally filtered by package, tag, priority, regex.",
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
		Description: "Clear logcat buffers (`logcat -c`).",
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

	type pkgArgs struct {
		DeviceArg
		Package string `json:"package"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "dumpsys_meminfo",
		Description: "Parsed `dumpsys meminfo <package>`: TotalPSS, native/dalvik heap, code, stack, graphics.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pkgArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		m, err := d.Dumpsys.MemInfo(ctx, dev, args.Package)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(m)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dumpsys_gfxinfo",
		Description: "Parsed `dumpsys gfxinfo <package>`: total/janky frames, jank %, latency percentiles.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pkgArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		g, err := d.Dumpsys.GfxInfo(ctx, dev, args.Package)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(g)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dumpsys_battery",
		Description: "Parsed `dumpsys battery`: level, voltage, temperature, technology, etc.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		b, err := d.Dumpsys.BatteryInfo(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(b)
	})

	type activityArgs struct {
		DeviceArg
		Package string `json:"package,omitempty" jsonschema:"if set, the recents list is filtered to this package"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "dumpsys_activity",
		Description: "Summary of `dumpsys activity activities`: focused activity, top resumed, recents stack.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args activityArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		a, err := d.Dumpsys.Activity(ctx, dev, args.Package)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(a)
	})

	type atraceArgs struct {
		DeviceArg
		DurationSec int      `json:"durationSec,omitempty" jsonschema:"capture duration in seconds (default 5, max 300)"`
		Categories  []string `json:"categories,omitempty" jsonschema:"atrace categories (default: gfx,view,input,wm,am)"`
		Output      string   `json:"output,omitempty" jsonschema:"host file path (.trace); auto-generated if empty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "atrace_capture",
		Description: "Run `atrace` for the given duration and pull the binary trace to host.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args atraceArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		path, err := d.Trace.AtraceCapture(ctx, dev, args.DurationSec, args.Categories, args.Output)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]string{"path": path})
	})

	type perfettoArgs struct {
		DeviceArg
		DurationSec int    `json:"durationSec,omitempty" jsonschema:"default 10, max 300"`
		ConfigText  string `json:"configText,omitempty" jsonschema:"perfetto text config (.pbtxt); a default is used if empty"`
		Output      string `json:"output,omitempty" jsonschema:"host file path (.pftrace)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "perfetto_capture",
		Description: "Run `perfetto` for the given duration and pull the .pftrace to host.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args perfettoArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		path, err := d.Trace.PerfettoCapture(ctx, dev, args.DurationSec, args.ConfigText, args.Output)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]string{"path": path})
	})
}
