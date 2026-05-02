package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/input"
	"github.com/randheer094/velocity-mcp-mobile/internal/system"
)

// RegisterInput registers tap/swipe/keys/clipboard tools.
func RegisterInput(s *mcp.Server, d *Deps) {
	type tapArgs struct {
		DeviceArg
		X int `json:"x" jsonschema:"x coordinate in pixels"`
		Y int `json:"y" jsonschema:"y coordinate in pixels"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "tap",
		Description: "Tap the screen at the given coordinates.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args tapArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.Tap(ctx, dev, args.X, args.Y); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("tapped (%d,%d)", args.X, args.Y))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "double_tap",
		Description: "Double-tap at the given coordinates.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args tapArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.DoubleTap(ctx, dev, args.X, args.Y); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("double-tapped (%d,%d)", args.X, args.Y))
	})

	type longPressArgs struct {
		tapArgs
		DurationMs int `json:"durationMs,omitempty" jsonschema:"hold duration in ms (1-10000, default 500)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "long_press",
		Description: "Long-press the screen at the given coordinates.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args longPressArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		dur := args.DurationMs
		if dur == 0 {
			dur = 500
		}
		if err := d.Input.LongPress(ctx, dev, args.X, args.Y, dur); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("long-pressed (%d,%d) for %dms", args.X, args.Y, dur))
	})

	type swipeArgs struct {
		DeviceArg
		Direction  string `json:"direction" jsonschema:"up | down | left | right"`
		AnchorX    int    `json:"anchorX,omitempty" jsonschema:"start x (default: screen center)"`
		AnchorY    int    `json:"anchorY,omitempty" jsonschema:"start y (default: screen center)"`
		Distance   int    `json:"distance,omitempty" jsonschema:"distance in px (default: 30% of relevant dim)"`
		DurationMs int    `json:"durationMs,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "swipe",
		Description: "Swipe across the screen in a direction. Defaults to centre-anchored, 30% of screen dim.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args swipeArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		size, err := screenSize(ctx, d.Screen, dev)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.Swipe(ctx, dev, input.Direction(args.Direction),
			size.Width, size.Height, args.AnchorX, args.AnchorY, args.Distance, args.DurationMs); err != nil {
			return errResult(err)
		}
		return textResult("swiped " + args.Direction)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "fling",
		Description: "Fast inertial swipe (~80ms). Defaults to centre-anchored.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args swipeArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		size, err := screenSize(ctx, d.Screen, dev)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.Fling(ctx, dev, input.Direction(args.Direction),
			size.Width, size.Height, args.AnchorX, args.AnchorY, args.Distance); err != nil {
			return errResult(err)
		}
		return textResult("flung " + args.Direction)
	})

	type dragArgs struct {
		DeviceArg
		FromX      int `json:"fromX"`
		FromY      int `json:"fromY"`
		ToX        int `json:"toX"`
		ToY        int `json:"toY"`
		DurationMs int `json:"durationMs,omitempty" jsonschema:"default 600"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "drag",
		Description: "Drag from (fromX,fromY) to (toX,toY) over durationMs.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args dragArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.Drag(ctx, dev, args.FromX, args.FromY, args.ToX, args.ToY, args.DurationMs); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("dragged (%d,%d)→(%d,%d)", args.FromX, args.FromY, args.ToX, args.ToY))
	})

	type typeArgs struct {
		DeviceArg
		Text   string `json:"text" jsonschema:"text to type into the focused field"`
		Submit bool   `json:"submit,omitempty" jsonschema:"if true, presses ENTER after typing"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "type_keys",
		Description: "Type text into the focused field. ASCII goes via `input text`; non-ASCII via the device clipboard then PASTE.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args typeArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.TypeKeys(ctx, dev, args.Text, args.Submit); err != nil {
			return errResult(err)
		}
		return textResult("typed " + fmt.Sprintf("%d chars", len(args.Text)))
	})

	type pressArgs struct {
		DeviceArg
		Button string `json:"button" jsonschema:"button name; e.g. BACK, HOME, ENTER, RECENTS, MENU, POWER, VOLUME_UP, VOLUME_DOWN"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "press_button",
		Description: "Press a hardware button by name.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pressArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.PressButton(ctx, dev, args.Button); err != nil {
			return errResult(err)
		}
		return textResult("pressed " + args.Button)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "clipboard_get",
		Description: "Read the device's primary clipboard (Android 10+).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		text, err := d.Input.GetClipboard(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]string{"text": text})
	})

	type clipSetArgs struct {
		DeviceArg
		Text string `json:"text"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "clipboard_set",
		Description: "Write text to the device's primary clipboard.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args clipSetArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.SetClipboard(ctx, dev, args.Text); err != nil {
			return errResult(err)
		}
		return textResult("clipboard updated")
	})
}

// screenSize is a small helper used by swipe/fling defaults.
func screenSize(ctx context.Context, sc *system.ScreenClient, dev string) (system.Size, error) {
	return sc.Get(ctx, dev)
}
