package tools

import (
	"context"

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
		Package string `json:"package" jsonschema:"the package to force-stop"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "app_terminate",
		Description: "Force-stop the app under test (test teardown).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args terminateArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Apps.Terminate(ctx, dev, args.Package); err != nil {
			return errResult(err)
		}
		return textResult("terminated " + args.Package)
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
		Mode     string            `json:"mode,omitempty" jsonschema:"start (default) or broadcast"`
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
		Description: "Dispatch an Android intent via `am start` / `am broadcast` (deep-link & broadcast tests).",
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
}
