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
