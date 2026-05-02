package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterSystem registers wake/lock, animations, doze, time, network, location.
func RegisterSystem(s *mcp.Server, d *Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_wake",
		Description: "Wake the device and best-effort dismiss simple lock screens.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Screen.Wake(ctx, dev); err != nil {
			return errResult(err)
		}
		return textResult("woken")
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_lock",
		Description: "Turn the screen off / lock the device (KEYCODE_SLEEP).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Screen.Lock(ctx, dev); err != nil {
			return errResult(err)
		}
		return textResult("locked")
	})

	type animArgs struct {
		DeviceArg
		Scale float64 `json:"scale" jsonschema:"animation scale (0 disables; 1 is default; max 10)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "animations_set",
		Description: "Set window/transition/animator scales together. 0 disables animations entirely (recommended for stable UI tests).",
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
		Description: "Read the three global animation scales.",
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

	type dozeArgs struct {
		DeviceArg
		State string `json:"state" jsonschema:"idle (force deep idle) or active (restore)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "doze_simulate",
		Description: "Simulate Doze (battery-saver) state. `idle` forces deep idle; `active` restores normal operation.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args dozeArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Doze.SetState(ctx, dev, args.State); err != nil {
			return errResult(err)
		}
		return textResult("doze state: " + args.State)
	})

	type tzArgs struct {
		DeviceArg
		Timezone string `json:"timezone" jsonschema:"Olson identifier, e.g. Asia/Tokyo"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "time_set_timezone",
		Description: "Set the device timezone via setprop and broadcast TIMEZONE_CHANGED.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args tzArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Time.SetTimezone(ctx, dev, args.Timezone); err != nil {
			return errResult(err)
		}
		return textResult("timezone set to " + args.Timezone)
	})

	type boolArgs struct {
		DeviceArg
		Enabled bool `json:"enabled"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "network_set_airplane",
		Description: "Toggle airplane mode.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args boolArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Network.SetAirplaneMode(ctx, dev, args.Enabled); err != nil {
			return errResult(err)
		}
		return textResult("airplane mode set")
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "network_set_wifi",
		Description: "Enable/disable Wi-Fi.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args boolArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Network.SetWiFi(ctx, dev, args.Enabled); err != nil {
			return errResult(err)
		}
		return textResult("wifi set")
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "network_set_mobile_data",
		Description: "Enable/disable mobile data.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args boolArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Network.SetMobileData(ctx, dev, args.Enabled); err != nil {
			return errResult(err)
		}
		return textResult("mobile data set")
	})

	type locArgs struct {
		DeviceArg
		Lat      float64  `json:"lat"`
		Lon      float64  `json:"lon"`
		Altitude *float64 `json:"altitude,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "location_set",
		Description: "Inject a GPS coordinate. Works directly on emulators; on physical devices a mock-location provider app must be installed.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args locArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		res, err := d.Location.Set(ctx, dev, args.Lat, args.Lon, args.Altitude)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})
}
