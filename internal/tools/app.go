package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/apps"
)

// RegisterApp exposes test setup/teardown and verification verbs over apps:
// launch / terminate / clear-data, permission grant/revoke, intent dispatch
// (deep-link tests), package metadata inspection, and run-as data inspection.
//
// APK install / uninstall is intentionally absent — those are deployment
// concerns, not test code.
func RegisterApp(s *mcp.Server, d *Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_list",
		Description: "List installed apps with launcher activities (find the app under test).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		list, err := d.Apps.List(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(list)
	})

	type launchArgs struct {
		DeviceArg
		Package string `json:"package" jsonschema:"the package to launch"`
		Locale  string `json:"locale,omitempty" jsonschema:"BCP-47 locale tag to apply via cmd locale set-app-locales (e.g. ja-JP)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_launch",
		Description: "Launch the app's main launcher activity, optionally with a per-app locale override (test setup).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args launchArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Apps.Launch(ctx, dev, args.Package, args.Locale); err != nil {
			return errResult(err)
		}
		return textResult("launched " + args.Package)
	})

	type terminateArgs struct {
		DeviceArg
		Package string `json:"package" jsonschema:"the package to terminate"`
		Kind    string `json:"kind,omitempty" jsonschema:"how to stop the process. force_stop (default) = am force-stop: hard stop, package marked STOPPED, cancels alarms/jobs, drops queued broadcasts, prevents service auto-restart until next user launch. kill = am kill: soft kill of the process only (cached/background processes; no-op if foreground), leaves package state intact so START_STICKY services restart, alarms still fire, and broadcasts still deliver. Use kill to simulate the OS reclaiming memory; use force_stop for a clean test reset."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "app_terminate",
		Description: "Stop the app under test. Two modes with materially different semantics:\n\n" +
			"• `kind=force_stop` (default) — runs `am force-stop <pkg>`. Hard stop: kills every process, " +
			"cancels pending alarms and JobScheduler jobs, drops queued broadcasts, and marks the package " +
			"STOPPED so it receives no implicit broadcasts and its services do NOT auto-restart until the " +
			"user launches it again. Use this for a clean teardown between tests.\n\n" +
			"• `kind=kill` — runs `am kill <pkg>`. Soft kill: terminates the process only if it is " +
			"cached/background (no-op while in the foreground), and leaves package state untouched. " +
			"`START_STICKY` services will be restarted by the system, scheduled alarms still fire, and " +
			"broadcasts continue to deliver. Use this to simulate the OS reclaiming memory — e.g. to " +
			"verify `START_STICKY` recovery, `onTaskRemoved` re-creation, or that a JobService re-runs " +
			"after its process dies.\n\n" +
			"Pick `kill` when the test is *about* what survives process death; pick `force_stop` when you " +
			"just need the app gone.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args terminateArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		switch args.Kind {
		case "", "force_stop":
			if err := d.Apps.Terminate(ctx, dev, args.Package); err != nil {
				return errResult(err)
			}
			return textResult("force-stopped " + args.Package)
		case "kill":
			if err := d.Apps.Kill(ctx, dev, args.Package); err != nil {
				return errResult(err)
			}
			return textResult("killed " + args.Package)
		default:
			return errResult(fmt.Errorf("invalid kind %q (expected force_stop or kill)", args.Kind))
		}
	})

	type clearArgs struct {
		DeviceArg
		Package string `json:"package" jsonschema:"the package whose user data should be wiped"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_clear_data",
		Description: "Wipe an app's user data (pm clear) — standard test setup for a clean state.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptrTrue()},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args clearArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Apps.ClearData(ctx, dev, args.Package); err != nil {
			return errResult(err)
		}
		return textResult("cleared data for " + args.Package)
	})

	type infoArgs struct {
		DeviceArg
		Package string `json:"package" jsonschema:"the package to inspect"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_get_info",
		Description: "Return parsed `dumpsys package` info: version, target SDK, granted vs requested permissions, install timestamps. Useful for asserting the build under test.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args infoArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		info, err := d.Apps.Info(ctx, dev, args.Package)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(info)
	})

	type permArgs struct {
		DeviceArg
		Package    string `json:"package"`
		Permission string `json:"permission" jsonschema:"the runtime permission, e.g. android.permission.CAMERA"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "permission_grant",
		Description: "Grant a runtime permission to a package (test setup).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args permArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Apps.GrantPermission(ctx, dev, args.Package, args.Permission); err != nil {
			return errResult(err)
		}
		return textResult("granted " + args.Permission + " to " + args.Package)
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "permission_revoke",
		Description: "Revoke a runtime permission from a package (test setup).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args permArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Apps.RevokePermission(ctx, dev, args.Package, args.Permission); err != nil {
			return errResult(err)
		}
		return textResult("revoked " + args.Permission + " from " + args.Package)
	})

	type intentArgs struct {
		DeviceArg
		Mode     string            `json:"mode,omitempty" jsonschema:"start (default), broadcast, service, or foreground_service"`
		Action   string            `json:"action,omitempty" jsonschema:"intent action, e.g. android.intent.action.VIEW"`
		Category string            `json:"category,omitempty"`
		Data     string            `json:"data,omitempty" jsonschema:"URI passed via -d (deep link target)"`
		Mime     string            `json:"mime,omitempty"`
		Package  string            `json:"package,omitempty" jsonschema:"restrict to a specific package"`
		Class    string            `json:"class,omitempty" jsonschema:"explicit component, e.g. com.example/.Main"`
		Flags    []string          `json:"flags,omitempty"`
		StringEx map[string]string `json:"stringExtras,omitempty"`
		IntEx    map[string]string `json:"intExtras,omitempty"`
		BoolEx   map[string]string `json:"boolExtras,omitempty"`
		FloatEx  map[string]string `json:"floatExtras,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "intent_send",
		Description: "Dispatch an Android intent via `am start` / `am broadcast` / `am start-service` / `am start-foreground-service`. Use `mode=foreground_service` to start a service that calls `startForeground()` on Android 8+; `mode=service` for plain background services on pre-O or `bindService()`-equivalent flows.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args intentArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		err = d.Apps.SendIntent(ctx, dev, apps.Intent{
			Mode:     args.Mode,
			Action:   args.Action,
			Category: args.Category,
			Data:     args.Data,
			MimeType: args.Mime,
			Package:  args.Package,
			Class:    args.Class,
			Flags:    args.Flags,
			StringEx: args.StringEx,
			IntEx:    args.IntEx,
			BoolEx:   args.BoolEx,
			FloatEx:  args.FloatEx,
		})
		if err != nil {
			return errResult(err)
		}
		return textResult("intent dispatched")
	})

	type appDataListArgs struct {
		DeviceArg
		Package      string `json:"package"`
		RelativePath string `json:"relativePath,omitempty" jsonschema:"path inside the package data dir (default: top level)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_data_list",
		Description: "List files inside the app's private data dir using run-as. Requires a debuggable build. Useful for asserting cache/database state in tests.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args appDataListArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		out, err := d.Apps.ListAppData(ctx, dev, args.Package, args.RelativePath)
		if err != nil {
			return errResult(err)
		}
		return textResult(out)
	})

	type appDataReadArgs struct {
		DeviceArg
		Package      string `json:"package"`
		RelativePath string `json:"relativePath" jsonschema:"path inside the package data dir, e.g. shared_prefs/settings.xml"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_data_read",
		Description: "Read a file inside the app's private data dir using run-as. Requires a debuggable build.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args appDataReadArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		out, err := d.Apps.ReadAppData(ctx, dev, args.Package, args.RelativePath)
		if err != nil {
			return errResult(err)
		}
		return textResult(string(out))
	})

	type appOpsSetArgs struct {
		DeviceArg
		Package string `json:"package"`
		Op      string `json:"op" jsonschema:"the AppOps op name, e.g. android:mock_location"`
		Mode    string `json:"mode" jsonschema:"allow | deny | ignore | default"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "appops_set",
		Description: "Set an AppOps mode on a package via `appops set`. Distinct from runtime permissions — covers system-level grants like `android:mock_location` that `permission_grant` can't reach.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args appOpsSetArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Apps.SetAppOp(ctx, dev, args.Package, args.Op, apps.AppOpMode(args.Mode)); err != nil {
			return errResult(err)
		}
		return textResult("set " + args.Op + "=" + args.Mode + " on " + args.Package)
	})

	type appOpsGetArgs struct {
		DeviceArg
		Package string `json:"package"`
		Op      string `json:"op" jsonschema:"the AppOps op name, e.g. android:mock_location"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "appops_get",
		Description: "Read the current AppOps mode for an op on a package. Returns `default` when the op is not explicitly set.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args appOpsGetArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		mode, err := d.Apps.GetAppOp(ctx, dev, args.Package, args.Op)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]string{"mode": string(mode)})
	})
}
