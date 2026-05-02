package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/matcher"
	"github.com/randheer094/velocity-test-mobile/internal/system"
)

// RegisterSystem exposes the device-state knobs that real Espresso/Compose
// tests routinely require — animation control, orientation, plus the
// dumpsys-backed introspection (activity / service / location / notification)
// and the `shell_exec` backstop. All other system surface (network, doze,
// time, etc.) is intentionally absent from this testing-only server.
func RegisterSystem(s *mcp.Server, d *Deps) {
	registerAnimations(s, d)
	registerActivity(s, d)
	registerService(s, d)
	registerLocation(s, d)
	registerNotifications(s, d)
	registerShellExec(s, d)
}

func registerAnimations(s *mcp.Server, d *Deps) {
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

func registerActivity(s *mcp.Server, d *Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "activity_get_top",
		Description: "Return the currently-resumed activity from `dumpsys activity activities`. Returns null when no activity is resumed.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		t, err := d.Activity.GetTop(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(t)
	})

	type waitTopArgs struct {
		DeviceArg
		Package   string `json:"bundle_id" jsonschema:"the package whose resumed activity we want to see on top"`
		Activity  string `json:"activity,omitempty" jsonschema:"optional activity class — fully-qualified or relative (\".MainActivity\")"`
		TimeoutMs int    `json:"timeout_ms,omitempty" jsonschema:"poll deadline in ms (default 5000)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "activity_wait_for_top",
		Description: "Poll `dumpsys activity` until the resumed activity matches `bundle_id` (and optional `activity`), or the timeout elapses.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args waitTopArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		timeout := time.Duration(args.TimeoutMs) * time.Millisecond
		t, err := d.Activity.WaitForTop(ctx, dev, args.Package, args.Activity, timeout)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(t)
	})

	type startArgs struct {
		DeviceArg
		Package      string            `json:"bundle_id"`
		Activity     string            `json:"activity" jsonschema:"fully-qualified or relative activity class (\".MainActivity\")"`
		Action       string            `json:"action,omitempty"`
		Data         string            `json:"data,omitempty" jsonschema:"URI passed via -d"`
		Flags        []string          `json:"flags,omitempty"`
		StringExtras map[string]string `json:"stringExtras,omitempty"`
		IntExtras    map[string]string `json:"intExtras,omitempty"`
		BoolExtras   map[string]string `json:"boolExtras,omitempty"`
		FloatExtras  map[string]string `json:"floatExtras,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "activity_start",
		Description: "Launch an explicit activity component via `am start -n <pkg>/<activity>`. Use this when launcher resolution picks the wrong activity (e.g. LeakCanary's launcher outranks the app's real entry point).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args startArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Activity.Start(ctx, dev, system.StartArgs{
			Package:  args.Package,
			Activity: args.Activity,
			Action:   args.Action,
			Data:     args.Data,
			Flags:    args.Flags,
			StringEx: args.StringExtras,
			IntEx:    args.IntExtras,
			BoolEx:   args.BoolExtras,
			FloatEx:  args.FloatExtras,
		}); err != nil {
			return errResult(err)
		}
		return textResult("started " + args.Package + "/" + args.Activity)
	})
}

func registerService(s *mcp.Server, d *Deps) {
	type stateArgs struct {
		DeviceArg
		Package   string `json:"bundle_id"`
		Component string `json:"component,omitempty" jsonschema:"optional fully-qualified service class to disambiguate when multiple services exist"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "service_get_state",
		Description: "Return the running/foreground state of a service from `dumpsys activity services <bundle_id>`. Includes `isForeground=true` detection plus startId, fg notification id, and the last bound intent action.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args stateArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		st, err := d.Service.GetState(ctx, dev, args.Package, args.Component)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(st)
	})

	type waitStateArgs struct {
		DeviceArg
		Package    string `json:"bundle_id"`
		Component  string `json:"component,omitempty"`
		Running    *bool  `json:"running,omitempty" jsonschema:"if set, wait until ServiceState.running matches"`
		Foreground *bool  `json:"foreground,omitempty" jsonschema:"if set, wait until ServiceState.foreground matches"`
		TimeoutMs  int    `json:"timeout_ms,omitempty" jsonschema:"poll deadline in ms (default 5000)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "service_wait_for_state",
		Description: "Poll `dumpsys activity services` until every set field of the expectation matches, or the timeout elapses.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args waitStateArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		timeout := time.Duration(args.TimeoutMs) * time.Millisecond
		st, err := d.Service.WaitForState(ctx, dev, args.Package, args.Component, system.ServiceExpectation{
			Running:    args.Running,
			Foreground: args.Foreground,
		}, timeout)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(st)
	})
}

func registerLocation(s *mcp.Server, d *Deps) {
	type locArgs struct {
		DeviceArg
		Provider string `json:"provider,omitempty" jsonschema:"gps | network | passive | fused — when omitted, the first reported provider wins"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "location_get_last_known",
		Description: "Return the most recent location reported to LocationManager (parsed from `dumpsys location`). Useful for verifying that a mock-location source actually reaches the framework, not just the UI.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args locArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		fix, err := d.Location.GetLastKnown(ctx, dev, args.Provider)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(fix)
	})
}

func registerNotifications(s *mcp.Server, d *Deps) {
	type listArgs struct {
		DeviceArg
		Package string `json:"bundle_id,omitempty"`
		Channel string `json:"channel_id,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notification_list",
		Description: "List currently-posted notifications, optionally filtered by package and/or channel id (parsed from `dumpsys notification --noredact`).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		out, err := d.Notifications.List(ctx, dev, system.ListFilter{Package: args.Package, Channel: args.Channel})
		if err != nil {
			return errResult(err)
		}
		if out == nil {
			out = []system.Notification{}
		}
		return jsonResult(map[string]any{"items": out})
	})

	type shadeArgs struct {
		DeviceArg
		State string `json:"state" jsonschema:"expanded | collapsed | quick_settings"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notification_shade_set",
		Description: "Open or close the system notification shade via `cmd statusbar` (expand-notifications | collapse | expand-settings).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args shadeArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Notifications.SetShade(ctx, dev, system.ShadeState(args.State)); err != nil {
			return errResult(err)
		}
		return textResult("shade set to " + args.State)
	})

	type tapArgs struct {
		DeviceArg
		Package    string `json:"bundle_id"`
		Channel    string `json:"channel_id,omitempty"`
		TitleMatch string `json:"title_match,omitempty" jsonschema:"optional substring match against the notification title to disambiguate when several match"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notification_tap",
		Description: "Tap an active notification by `(bundle_id, channel_id?)` — opens the shade, finds the matching notification's title in the layout, clicks it, and collapses the shade. Higher-level than `notification_shade_set + find_node + click`.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args tapArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		notes, err := d.Notifications.List(ctx, dev, system.ListFilter{Package: args.Package, Channel: args.Channel})
		if err != nil {
			return errResult(err)
		}
		var picked *system.Notification
		for i := range notes {
			n := notes[i]
			if args.TitleMatch != "" &&
				!strings.Contains(n.Title, args.TitleMatch) &&
				!strings.Contains(n.Text, args.TitleMatch) {
				continue
			}
			picked = &n
			break
		}
		if picked == nil {
			return errResult(fmt.Errorf("no notification matched bundle_id=%q channel_id=%q title_match=%q", args.Package, args.Channel, args.TitleMatch))
		}
		if picked.Title == "" && picked.Text == "" {
			return errResult(fmt.Errorf("notification for %s has no title/text to click on", args.Package))
		}
		needle := picked.Title
		if needle == "" {
			needle = picked.Text
		}
		if err := d.Notifications.SetShade(ctx, dev, system.ShadeExpanded); err != nil {
			return errResult(err)
		}
		// Re-use the testing surface: a substring match by visible text is the
		// same primitive `find_node + click` in the runbooks.
		res, err := d.Tester.Click(ctx, dev, &matcher.Matcher{TextContains: needle})
		if err != nil {
			_ = d.Notifications.SetShade(ctx, dev, system.ShadeCollapsed)
			return errResult(err)
		}
		_ = d.Notifications.SetShade(ctx, dev, system.ShadeCollapsed)
		return jsonResult(res)
	})
}

func registerShellExec(s *mcp.Server, d *Deps) {
	type shellArgs struct {
		DeviceArg
		Command   string `json:"command" jsonschema:"the full shell command, forwarded as-is to adb shell"`
		TimeoutMs int    `json:"timeout_ms,omitempty" jsonschema:"per-call timeout in ms (default 10000)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "shell_exec",
		Description: "Backstop for arbitrary `adb shell <command>`. Prefer the typed wrappers (service_get_state, notification_list, location_get_last_known, etc.); use this only for one-off introspection of dumpsys services or debug-only setprop calls. Returns stdout/stderr and exit_code regardless of exit status.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args shellArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		timeout := 10 * time.Second
		if args.TimeoutMs > 0 {
			timeout = time.Duration(args.TimeoutMs) * time.Millisecond
		}
		res, err := d.Shell.Exec(ctx, dev, args.Command, timeout)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})
}
