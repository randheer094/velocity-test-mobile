// Package input wraps `adb shell input` for taps, swipes, drags, key events,
// text entry, and clipboard control.
package input

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// Direction is one of "up", "down", "left", "right".
type Direction string

// Client groups input-related adb shell calls.
type Client struct {
	Adb *adb.Client
}

// New returns a Client.
func New(a *adb.Client) *Client { return &Client{Adb: a} }

// Tap sends a tap event.
func (c *Client) Tap(ctx context.Context, deviceID string, x, y int) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("coordinates must be non-negative")
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "tap", itoa(x), itoa(y))
	return err
}

// DoubleTap sends two taps in rapid succession.
func (c *Client) DoubleTap(ctx context.Context, deviceID string, x, y int) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("coordinates must be non-negative")
	}
	if err := c.tapOnce(ctx, deviceID, x, y); err != nil {
		return err
	}
	return c.tapOnce(ctx, deviceID, x, y)
}

func (c *Client) tapOnce(ctx context.Context, deviceID string, x, y int) error {
	_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "tap", itoa(x), itoa(y))
	return err
}

// LongPress simulates a press at (x,y) for durationMs milliseconds (1..10000).
func (c *Client) LongPress(ctx context.Context, deviceID string, x, y, durationMs int) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("coordinates must be non-negative")
	}
	if durationMs < 1 || durationMs > 10000 {
		return fmt.Errorf("durationMs must be between 1 and 10000")
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "swipe",
		itoa(x), itoa(y), itoa(x), itoa(y), itoa(durationMs))
	return err
}

// Drag swipes from (fromX,fromY) to (toX,toY) over durationMs.
func (c *Client) Drag(ctx context.Context, deviceID string, fromX, fromY, toX, toY, durationMs int) error {
	if fromX < 0 || fromY < 0 || toX < 0 || toY < 0 {
		return fmt.Errorf("coordinates must be non-negative")
	}
	if durationMs <= 0 {
		durationMs = 600
	}
	if durationMs > 10000 {
		return fmt.Errorf("durationMs must be <= 10000")
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "swipe",
		itoa(fromX), itoa(fromY), itoa(toX), itoa(toY), itoa(durationMs))
	return err
}

// Swipe scrolls the screen in a given direction. anchorX/anchorY default to
// the screen centre. distance is in pixels (default 30% of the relevant dim).
func (c *Client) Swipe(ctx context.Context, deviceID string, dir Direction, screenW, screenH int, anchorX, anchorY, distance, durationMs int) error {
	if anchorX <= 0 {
		anchorX = screenW / 2
	}
	if anchorY <= 0 {
		anchorY = screenH / 2
	}
	if distance <= 0 {
		switch dir {
		case "up", "down":
			distance = int(float64(screenH) * 0.30)
		case "left", "right":
			distance = int(float64(screenW) * 0.30)
		}
	}
	if durationMs <= 0 {
		durationMs = 200
	}
	endX, endY := anchorX, anchorY
	switch dir {
	case "up":
		endY = anchorY - distance
	case "down":
		endY = anchorY + distance
	case "left":
		endX = anchorX - distance
	case "right":
		endX = anchorX + distance
	default:
		return fmt.Errorf("invalid direction %q", dir)
	}
	endX = clamp(endX, 0, screenW)
	endY = clamp(endY, 0, screenH)
	_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "swipe",
		itoa(anchorX), itoa(anchorY), itoa(endX), itoa(endY), itoa(durationMs))
	return err
}

// Fling is a fast swipe used for inertial scrolling.
func (c *Client) Fling(ctx context.Context, deviceID string, dir Direction, screenW, screenH, anchorX, anchorY, distance int) error {
	return c.Swipe(ctx, deviceID, dir, screenW, screenH, anchorX, anchorY, distance, 80)
}

// PressButton sends a keyevent for the named button.
func (c *Client) PressButton(ctx context.Context, deviceID, name string) error {
	code, err := adb.Keycode(name)
	if err != nil {
		return err
	}
	_, err = c.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", itoa(code))
	return err
}

// TypeKeys enters text into the focused field. ASCII bytes go through
// `input text`; non-ASCII text is base64-encoded into the device clipboard
// and pasted via KEYCODE_PASTE.
func (c *Client) TypeKeys(ctx context.Context, deviceID, text string, submit bool) error {
	if text == "" {
		if submit {
			_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", "66")
			return err
		}
		return nil
	}
	if isASCII(text) {
		// Use the simple input-text path. Spaces require %s and shell escaping.
		converted := strings.ReplaceAll(text, " ", "%s")
		_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "text", converted)
		if err != nil {
			return err
		}
	} else {
		if err := c.pasteUnicode(ctx, deviceID, text); err != nil {
			return err
		}
	}
	if submit {
		if _, err := c.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", "66"); err != nil {
			return err
		}
	}
	return nil
}

// pasteUnicode writes the text into the primary clipboard via the cmd
// service and presses PASTE. This avoids `input text`'s ASCII-only limit.
func (c *Client) pasteUnicode(ctx context.Context, deviceID, text string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	// Try API-29+ path first.
	cmd := fmt.Sprintf("echo -n %s | base64 -d | cmd clipboard set-primary --user 0", adb.QuoteForShell(encoded))
	if _, err := c.Adb.Shell(ctx, deviceID, cmd); err == nil {
		_, err := c.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", "279")
		return err
	}
	return fmt.Errorf("non-ASCII typing is not supported on this device (cmd clipboard set-primary failed)")
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7f {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if hi > 0 && v > hi {
		return hi
	}
	return v
}
