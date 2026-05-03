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

	type recordStartArgs struct {
		DeviceArg
		LocalFile    string `json:"local_file" jsonschema:"host path the recording is pulled to on stop (e.g. /tmp/test-run.mp4)"`
		MaxDurationS int    `json:"max_duration_s,omitempty" jsonschema:"device-side time limit in seconds (Android default is 180; 1800+ on Android 11+)"`
		SizeWidth    int    `json:"size_width,omitempty" jsonschema:"optional output width in px (paired with size_height to downscale)"`
		SizeHeight   int    `json:"size_height,omitempty"`
		BitRate      int    `json:"bit_rate,omitempty" jsonschema:"bits per second; 0 uses the device default (~4 Mbps at 720p)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_record_start",
		Description: "Start `adb shell screenrecord` in the background. One active recording per device. Pair with `screen_record_stop` to flush the MP4 container, pull the file to `local_file`, and remove the device-side temp.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args recordStartArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		remote, err := d.Record.Start(ctx, dev, diagnostics.RecordOptions{
			LocalFile:    args.LocalFile,
			MaxDurationS: args.MaxDurationS,
			SizeWidth:    args.SizeWidth,
			SizeHeight:   args.SizeHeight,
			BitRate:      args.BitRate,
		})
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "remote_file": remote, "local_file": args.LocalFile})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_record_stop",
		Description: "Stop the active screen recording on this device, pull the MP4 to the path passed to `screen_record_start`, and remove the device-side temp.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		res, err := d.Record.Stop(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	type pullArgs struct {
		DeviceArg
		Remote string `json:"remote" jsonschema:"absolute device path (e.g. /sdcard/Pictures/foo.png, /data/local/tmp/dump.txt)"`
		Local  string `json:"local" jsonschema:"host path to write to"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "pull_file",
		Description: "Copy a file off the device via `adb pull`. Use for artefacts in /sdcard/ or /data/local/tmp/. For debuggable-app private storage, prefer `app_data_read` (uses run-as).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pullArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Record.PullFile(ctx, dev, args.Remote, args.Local); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "remote": args.Remote, "local": args.Local})
	})
}
