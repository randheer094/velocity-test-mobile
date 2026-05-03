package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// RegisterInput exposes only the test-supporting input verbs that don't
// require a matcher: clipboard read/write and a generic key press.
//
// All other input — taps, swipes, text entry, etc. — is reachable via the
// semantic testing tools (click, type_text, swipe_node, scroll_to, etc.).
func RegisterInput(s *mcp.Server, d *Deps) {
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

	type pressKeyArgs struct {
		DeviceArg
		Key string `json:"key" jsonschema:"key name; e.g. BACK, HOME, ENTER, RECENTS, MENU, VOLUME_UP, VOLUME_DOWN, DEL, TAB, ESCAPE, DPAD_UP/DOWN/LEFT/RIGHT/CENTER, MEDIA_PLAY_PAUSE"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "press_key",
		Description: "Press a hardware/system key by name (Espresso pressKey / Compose performKeyPress without modifiers).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pressKeyArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.PressButton(ctx, dev, args.Key); err != nil {
			return errResult(err)
		}
		return textResult("pressed " + args.Key)
	})

	type tapAtArgs struct {
		DeviceArg
		X int `json:"x" jsonschema:"x coordinate in pixels (0-based, top-left origin)"`
		Y int `json:"y" jsonschema:"y coordinate in pixels"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "tap_at_coordinates",
		Description: "Send a tap at raw screen coordinates. Use as a fallback when no matcher resolves the target — e.g. fully custom Canvas surfaces, unlabelled Compose drawing, or in-app web content. Prefer the semantic `click` (matcher-based) for ordinary UI.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args tapAtArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.Tap(ctx, dev, args.X, args.Y); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "x": args.X, "y": args.Y})
	})

	type longPressAtArgs struct {
		DeviceArg
		X          int `json:"x"`
		Y          int `json:"y"`
		DurationMs int `json:"durationMs,omitempty" jsonschema:"hold time in ms (default 800)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "long_press_at_coordinates",
		Description: "Long-press at raw screen coordinates. Same fallback role as tap_at_coordinates; default duration 800ms.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args longPressAtArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		dur := args.DurationMs
		if dur <= 0 {
			dur = 800
		}
		if err := d.Input.LongPress(ctx, dev, args.X, args.Y, dur); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "x": args.X, "y": args.Y, "durationMs": dur})
	})

	type swipeScreenArgs struct {
		DeviceArg
		FromX      int `json:"fromX"`
		FromY      int `json:"fromY"`
		ToX        int `json:"toX"`
		ToY        int `json:"toY"`
		DurationMs int `json:"durationMs,omitempty" jsonschema:"swipe duration in ms (default 200 — fast swipe). Longer values register as drag in some apps; see drag_screen."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "swipe_screen",
		Description: "Swipe from (fromX,fromY) to (toX,toY) at the screen level. Default duration 200ms. Use for full-screen gestures (edge swipes, pull-to-refresh) where a node-scoped swipe_node isn't appropriate.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args swipeScreenArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		dur := args.DurationMs
		if dur <= 0 {
			dur = 200
		}
		if err := d.Input.Drag(ctx, dev, args.FromX, args.FromY, args.ToX, args.ToY, dur); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "fromX": args.FromX, "fromY": args.FromY, "toX": args.ToX, "toY": args.ToY, "durationMs": dur})
	})

	type dragScreenArgs struct {
		DeviceArg
		FromX      int `json:"fromX"`
		FromY      int `json:"fromY"`
		ToX        int `json:"toX"`
		ToY        int `json:"toY"`
		DurationMs int `json:"durationMs,omitempty" jsonschema:"drag duration in ms (default 800 — slow enough to register as a drag rather than swipe in most apps)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "drag_screen",
		Description: "Drag from (fromX,fromY) to (toX,toY) at the screen level. Same dispatch as swipe_screen but with a longer default (800ms) so apps that distinguish drag-and-drop from swipes register the gesture correctly.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args dragScreenArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		dur := args.DurationMs
		if dur <= 0 {
			dur = 800
		}
		if err := d.Input.Drag(ctx, dev, args.FromX, args.FromY, args.ToX, args.ToY, dur); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "fromX": args.FromX, "fromY": args.FromY, "toX": args.ToX, "toY": args.ToY, "durationMs": dur})
	})

	type typeFocusedArgs struct {
		DeviceArg
		Text   string `json:"text"`
		Submit bool   `json:"submit,omitempty" jsonschema:"press ENTER after typing"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "type_into_focused",
		Description: "Type text into the currently-focused field without first selecting it (Espresso typeTextIntoFocusedView). Useful when focus has already been set by a prior click.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args typeFocusedArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Input.TypeKeys(ctx, dev, args.Text, args.Submit); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("typed %d chars into focused view", len(args.Text)))
	})
}

var _ = adb.QuoteForShell // keep adb import used elsewhere if pruned
