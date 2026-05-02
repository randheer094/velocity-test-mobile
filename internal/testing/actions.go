package testing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/randheer094/velocity-mcp-mobile/internal/input"
	"github.com/randheer094/velocity-mcp-mobile/internal/matcher"
	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// ActionResult is returned by every action verb so the LLM agent sees what
// was matched and where the input was actually dispatched.
type ActionResult struct {
	OK      bool        `json:"ok"`
	Element *ui.Element `json:"element,omitempty"`
	X       int         `json:"x"`
	Y       int         `json:"y"`
	Reason  string      `json:"reason,omitempty"`
}

// Click — Espresso click() / Compose performClick().
func (o *Orchestrator) Click(ctx context.Context, deviceID string, m *matcher.Matcher) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	x, y := CenterOf(elem)
	if err := o.Input.Tap(ctx, deviceID, x, y); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// DoubleClick — Espresso doubleClick().
func (o *Orchestrator) DoubleClick(ctx context.Context, deviceID string, m *matcher.Matcher) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	x, y := CenterOf(elem)
	if err := o.Input.DoubleTap(ctx, deviceID, x, y); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// LongClick — Espresso longClick().
func (o *Orchestrator) LongClick(ctx context.Context, deviceID string, m *matcher.Matcher, durationMs int) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	if durationMs <= 0 {
		durationMs = 800
	}
	x, y := CenterOf(elem)
	if err := o.Input.LongPress(ctx, deviceID, x, y, durationMs); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// TypeText — Espresso typeText() / Compose performTextInput().
// First clicks the element to give it focus, then dispatches text. If the
// element is already focused, the click is a no-op as far as content goes.
func (o *Orchestrator) TypeText(ctx context.Context, deviceID string, m *matcher.Matcher, text string, submit bool) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	x, y := CenterOf(elem)
	if !elem.Focused {
		if err := o.Input.Tap(ctx, deviceID, x, y); err != nil {
			return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
		}
		// Give focus a moment to settle.
		select {
		case <-ctx.Done():
			return ActionResult{}, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	if err := o.Input.TypeKeys(ctx, deviceID, text, submit); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// ReplaceText — Espresso replaceText() / Compose performTextReplacement.
// Implementation: click → select-all (Ctrl+A via key event) → delete → type.
func (o *Orchestrator) ReplaceText(ctx context.Context, deviceID string, m *matcher.Matcher, text string, submit bool) (ActionResult, error) {
	if err := o.clearTextField(ctx, deviceID, m); err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	return o.TypeText(ctx, deviceID, m, text, submit)
}

// ClearText — Espresso clearText() / Compose performTextClearance.
func (o *Orchestrator) ClearText(ctx context.Context, deviceID string, m *matcher.Matcher) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	if err := o.clearTextField(ctx, deviceID, m); err != nil {
		return ActionResult{Element: &elem, Reason: err.Error()}, err
	}
	x, y := CenterOf(elem)
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// clearTextField focuses the element then issues KEYCODE_MOVE_END (123)
// to put the cursor at the end, then KEYCODE_DEL (67) repeatedly to wipe.
// We use Ctrl+A + DEL via shell `input keycombination` if supported, else
// fall back to repeated DEL using the existing text length.
func (o *Orchestrator) clearTextField(ctx context.Context, deviceID string, m *matcher.Matcher) error {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return err
	}
	x, y := CenterOf(elem)
	if !elem.Focused {
		if err := o.Input.Tap(ctx, deviceID, x, y); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	// Move cursor to end then delete `len(text)` characters. This is the
	// most portable approach across keyboards / IMEs.
	current := elem.Text
	if err := o.Input.PressButton(ctx, deviceID, "DPAD_RIGHT"); err != nil {
		// non-fatal
		_ = err
	}
	for i := 0; i < len([]rune(current))+1; i++ {
		if err := o.Input.PressButton(ctx, deviceID, "DEL"); err != nil {
			return err
		}
	}
	return nil
}

// Submit — Espresso pressImeActionButton; convenience for "type then ENTER".
func (o *Orchestrator) Submit(ctx context.Context, deviceID string, m *matcher.Matcher) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	x, y := CenterOf(elem)
	if !elem.Focused {
		if err := o.Input.Tap(ctx, deviceID, x, y); err != nil {
			return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
		}
		select {
		case <-ctx.Done():
			return ActionResult{}, ctx.Err()
		case <-time.After(120 * time.Millisecond):
		}
	}
	if err := o.Input.PressButton(ctx, deviceID, "ENTER"); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// SwipeNode — Espresso swipeUp/Down/Left/Right when bound to a specific view.
func (o *Orchestrator) SwipeNode(ctx context.Context, deviceID string, m *matcher.Matcher, direction string, durationMs int) (ActionResult, error) {
	elem, _, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	x, y := CenterOf(elem)
	if durationMs <= 0 {
		durationMs = 250
	}
	dist := elem.Bounds.Width / 3
	if direction == "up" || direction == "down" {
		dist = elem.Bounds.Height / 3
	}
	if dist < 50 {
		dist = 50
	}
	if err := o.Input.Swipe(ctx, deviceID, input.Direction(direction),
		elem.Bounds.X+elem.Bounds.Width, elem.Bounds.Y+elem.Bounds.Height, x, y, dist, durationMs); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// ScrollOptions controls the ScrollTo behaviour.
type ScrollOptions struct {
	MaxAttempts int    // total swipes to attempt (default 12)
	Direction   string // "auto" | "up" | "down" | "left" | "right"
}

// ScrollTo — Espresso scrollTo() / Compose performScrollToNode.
// Locates a scrollable ancestor and swipes within it until either the target
// matcher is visible or MaxAttempts is exhausted. When Direction is "auto"
// we swipe up first (move content up to reveal items below); after half the
// attempts we switch to swipe-down to also probe upward.
func (o *Orchestrator) ScrollTo(ctx context.Context, deviceID string, m *matcher.Matcher, opts ScrollOptions) (ActionResult, error) {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 12
	}
	if opts.Direction == "" {
		opts.Direction = "auto"
	}
	for attempt := 0; attempt <= opts.MaxAttempts; attempt++ {
		root, err := o.Layout.Tree(ctx, deviceID)
		if err != nil {
			return ActionResult{Reason: err.Error()}, err
		}
		matches, err := matcher.FindAll(root, m)
		if err == matcher.ErrEmptyMatcher {
			return ActionResult{}, err
		}
		if err == nil && len(matches) > 0 {
			elem := matches[0]
			if matcher.IsDisplayed(elem) {
				x, y := CenterOf(elem)
				return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
			}
		}
		if attempt == opts.MaxAttempts {
			break
		}
		// Find the scrollable container, default to the largest visible scrollable.
		scrollable, ok := largestScrollable(root)
		if !ok {
			return ActionResult{Reason: "no scrollable ancestor found"}, fmt.Errorf("no scrollable container on screen")
		}
		dir := opts.Direction
		if dir == "auto" {
			if attempt < opts.MaxAttempts/2 {
				dir = "up"
			} else {
				dir = "down"
			}
		}
		if err := o.swipeWithin(ctx, deviceID, scrollable, dir); err != nil {
			return ActionResult{Reason: err.Error()}, err
		}
		// Brief settle.
		select {
		case <-ctx.Done():
			return ActionResult{}, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return ActionResult{Reason: "target never appeared on screen"}, errors.New("scroll_to: target not reachable within attempt budget")
}

func (o *Orchestrator) swipeWithin(ctx context.Context, deviceID string, container ui.Element, direction string) error {
	w := container.Bounds.Width
	h := container.Bounds.Height
	cx := container.Bounds.X + w/2
	cy := container.Bounds.Y + h/2
	dist := h / 3
	if direction == "left" || direction == "right" {
		dist = w / 3
	}
	if dist < 50 {
		dist = 50
	}
	return o.Input.Swipe(ctx, deviceID, input.Direction(direction), w, h, cx, cy, dist, 200)
}

func largestScrollable(root ui.Element) (ui.Element, bool) {
	var best ui.Element
	bestArea := 0
	var walk func(ui.Element)
	walk = func(e ui.Element) {
		if e.Scrollable && matcher.IsDisplayed(e) {
			area := e.Bounds.Width * e.Bounds.Height
			if area > bestArea {
				best = e
				bestArea = area
			}
		}
		for _, c := range e.Children {
			walk(c)
		}
	}
	walk(root)
	return best, bestArea > 0
}

// PerformIMEAction — Espresso pressImeActionButton (ENTER on a focused field).
func (o *Orchestrator) PerformIMEAction(ctx context.Context, deviceID string, m *matcher.Matcher) (ActionResult, error) {
	if m != nil && !m.IsEmpty() {
		// Focus the matched element first.
		elem, _, err := o.fetchAndFind(ctx, deviceID, m)
		if err != nil {
			return ActionResult{Reason: err.Error()}, err
		}
		x, y := CenterOf(elem)
		if !elem.Focused {
			if err := o.Input.Tap(ctx, deviceID, x, y); err != nil {
				return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
			}
		}
	}
	if err := o.Input.PressButton(ctx, deviceID, "ENTER"); err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	return ActionResult{OK: true}, nil
}
