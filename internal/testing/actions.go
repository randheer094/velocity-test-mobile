package testing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/input"
	"github.com/randheer094/velocity-test-mobile/internal/matcher"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
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

// clearTextField focuses the matched field and wipes its text.
//
// Strategy:
//  1. If the field isn't focused, tap its centre and wait briefly.
//  2. Try CTRL+A (select-all) + DEL via `input keycombination` (API 31+).
//  3. Fall back to MOVE_END followed by repeated DEL — sized by the
//     current text length plus a margin.
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

	// Preferred path: CTRL+A then DEL — single keystroke clear, regardless of length.
	if err := o.Input.PressKeyCombination(ctx, deviceID, "CTRL_LEFT", "A"); err == nil {
		if err := o.Input.PressButton(ctx, deviceID, "DEL"); err == nil {
			return nil
		}
	}

	// Fallback: jump to end of line then issue DEL once per existing rune,
	// plus a small margin in case more text is in the field than was
	// captured by the accessibility snapshot.
	_ = o.Input.PressButton(ctx, deviceID, "MOVE_END")
	count := len([]rune(elem.Text)) + 4
	if count > 256 {
		count = 256 // cap to keep latency bounded on huge fields
	}
	for i := 0; i < count; i++ {
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
//
// The swipe is centred on the matched element and sized to one third of the
// node's relevant dimension (with a 50px floor). We pass 0 for screenW/H
// to Input.Swipe so the upper clamp is disabled — the real screen bounds
// will be enforced by Android's input system itself.
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
		0, 0, x, y, dist, durationMs); err != nil {
		return ActionResult{Element: &elem, X: x, Y: y, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// ScrollOptions controls the ScrollTo behaviour.
type ScrollOptions struct {
	MaxAttempts int              // total swipes to attempt (default 12)
	Direction   string           // "auto" | "up" | "down" | "left" | "right"
	Container   *matcher.Matcher // if non-empty, restrict the scroll to a specific scrollable
}

// ScrollTo — Espresso scrollTo() / Compose performScrollToNode.
//
// Locates a scrollable ancestor and swipes within it until either the
// target matcher is visible or MaxAttempts is exhausted.
//
// Container selection:
//   - If opts.Container is supplied and resolves to a scrollable node, use that.
//   - Otherwise pick the largest visible scrollable on screen.
//
// Direction:
//   - "auto" (default): swipe up for the first half of the attempts, then
//     swipe down for the remainder so we probe both directions.
//   - "up"/"down"/"left"/"right": swipe in that direction every attempt.
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
		scrollable, ok := chooseScrollable(root, opts.Container)
		if !ok {
			return ActionResult{Reason: "no scrollable container found"}, fmt.Errorf("no scrollable container on screen")
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

// chooseScrollable returns the requested container (if a non-empty matcher
// is supplied and resolves to a scrollable node) or the largest visible
// scrollable on screen.
func chooseScrollable(root ui.Element, container *matcher.Matcher) (ui.Element, bool) {
	if container != nil && !container.IsEmpty() {
		all, err := matcher.FindAll(root, container)
		if err == nil {
			for _, e := range all {
				if e.Scrollable && matcher.IsDisplayed(e) {
					return e, true
				}
			}
		}
		return ui.Element{}, false
	}
	return largestScrollable(root)
}

// swipeWithin dispatches a swipe centred on the container's bounds, sized
// to one third of the relevant dimension. screenW/H are passed as 0 so the
// upper clamp in Input.Swipe is disabled — the system's real screen bounds
// take over.
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
	return o.Input.Swipe(ctx, deviceID, input.Direction(direction), 0, 0, cx, cy, dist, 200)
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

// SlowSwipeNode — Espresso slowSwipeLeft / slow gesture variant. Same as
// SwipeNode but with a longer default duration.
func (o *Orchestrator) SlowSwipeNode(ctx context.Context, deviceID string, m *matcher.Matcher, direction string) (ActionResult, error) {
	return o.SwipeNode(ctx, deviceID, m, direction, 1500)
}

// DragNode drags the centre of the `from` matcher to the centre of the `to`
// matcher. Useful for reorderable lists, drag-and-drop, and slider thumbs
// where direction-based SwipeNode isn't enough. Implemented as a single
// `input swipe` from one centre to the other with a default 600ms duration.
func (o *Orchestrator) DragNode(ctx context.Context, deviceID string, from, to *matcher.Matcher, durationMs int) (ActionResult, error) {
	src, _, err := o.fetchAndFind(ctx, deviceID, from)
	if err != nil {
		return ActionResult{Reason: "from: " + err.Error()}, err
	}
	dst, _, err := o.fetchAndFind(ctx, deviceID, to)
	if err != nil {
		return ActionResult{Element: &src, Reason: "to: " + err.Error()}, err
	}
	if durationMs <= 0 {
		durationMs = 600
	}
	fx, fy := CenterOf(src)
	tx, ty := CenterOf(dst)
	if err := o.Input.Drag(ctx, deviceID, fx, fy, tx, ty, durationMs); err != nil {
		return ActionResult{Element: &src, X: tx, Y: ty, Reason: err.Error()}, err
	}
	return ActionResult{OK: true, Element: &src, X: tx, Y: ty}, nil
}

// ScrollToIndex — Compose performScrollToIndex(idx).
//
// LazyColumn/Row item indexing is opaque from outside the app, so this is
// an approximation: the matched scrollable container is swiped `index`
// times in `direction`. direction defaults to "up" (which scrolls the
// content upward, revealing later items in a vertical list); pass
// "down"/"left"/"right" to override.
func (o *Orchestrator) ScrollToIndex(ctx context.Context, deviceID string, container *matcher.Matcher, index int, direction string) (ActionResult, error) {
	if index < 0 {
		return ActionResult{Reason: "index must be non-negative"}, fmt.Errorf("index out of range")
	}
	if direction == "" {
		direction = "up"
	}
	elem, _, err := o.fetchAndFind(ctx, deviceID, container)
	if err != nil {
		return ActionResult{Reason: err.Error()}, err
	}
	if !elem.Scrollable {
		return ActionResult{Element: &elem, Reason: "matched element is not scrollable"}, fmt.Errorf("not scrollable")
	}
	for i := 0; i < index; i++ {
		if err := o.swipeWithin(ctx, deviceID, elem, direction); err != nil {
			return ActionResult{Element: &elem, Reason: err.Error()}, err
		}
		select {
		case <-ctx.Done():
			return ActionResult{}, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	x, y := CenterOf(elem)
	return ActionResult{OK: true, Element: &elem, X: x, Y: y}, nil
}

// PerformKeyPress — Compose performKeyPress(key, meta).
//
// Modifier keys (ctrl/shift/alt) are dispatched via `input keycombination`
// when the device supports it (Android 12+). On older devices we fall back
// to a plain `input keyevent` for the key alone and surface a Reason
// describing the missing modifier coverage.
func (o *Orchestrator) PerformKeyPress(ctx context.Context, deviceID string, m *matcher.Matcher, key string, ctrl, shift, alt bool) (ActionResult, error) {
	if m != nil && !m.IsEmpty() {
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
	}
	if !(ctrl || shift || alt) {
		if err := o.Input.PressButton(ctx, deviceID, key); err != nil {
			return ActionResult{Reason: err.Error()}, err
		}
		return ActionResult{OK: true}, nil
	}
	// Build the modifier chord.
	chord := []string{}
	if ctrl {
		chord = append(chord, "CTRL_LEFT")
	}
	if shift {
		chord = append(chord, "SHIFT_LEFT")
	}
	if alt {
		chord = append(chord, "ALT_LEFT")
	}
	chord = append(chord, key)
	if err := o.Input.PressKeyCombination(ctx, deviceID, chord...); err != nil {
		// Fallback: dispatch the key on its own and report the missing
		// modifier coverage so the agent can decide whether to retry.
		_ = o.Input.PressButton(ctx, deviceID, key)
		return ActionResult{
			OK:     true,
			Reason: fmt.Sprintf("modifier chord unsupported (%v); pressed %q without modifiers", err, key),
		}, nil
	}
	return ActionResult{OK: true}, nil
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
