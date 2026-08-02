// Package testing implements Espresso- and Compose-style verbs over the
// device's accessibility tree. Every operation re-fetches the tree, applies
// a Matcher, and dispatches an action via the existing input client.
//
// Synchronization is approximated by polling: there is no IdlingResource
// hook from outside the app process, so wait_for_idle hashes the tree at
// intervals and returns when consecutive snapshots match.
package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/input"
	"github.com/randheer094/velocity-test-mobile/internal/matcher"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// layoutSource is the subset of *ui.LayoutClient this package depends on.
// Defined here (consumer side) so tests can substitute a fake without a
// real device — *ui.LayoutClient already satisfies this implicitly.
type layoutSource interface {
	Tree(ctx context.Context, deviceID string) (ui.Element, error)
}

// inputDispatcher is the subset of *input.Client this package depends on.
// Same rationale as layoutSource — *input.Client already satisfies this.
type inputDispatcher interface {
	Tap(ctx context.Context, deviceID string, x, y int) error
	DoubleTap(ctx context.Context, deviceID string, x, y int) error
	LongPress(ctx context.Context, deviceID string, x, y, durationMs int) error
	Drag(ctx context.Context, deviceID string, fromX, fromY, toX, toY, durationMs int) error
	Swipe(ctx context.Context, deviceID string, dir input.Direction, screenW, screenH int, anchorX, anchorY, distance, durationMs int) error
	PressButton(ctx context.Context, deviceID, name string) error
	PressButtonRepeat(ctx context.Context, deviceID, name string, count int) error
	PressKeyCombination(ctx context.Context, deviceID string, names ...string) error
	TypeKeys(ctx context.Context, deviceID, text string, submit bool) error
	TapAndPressButton(ctx context.Context, deviceID string, x, y, settleMs int, name string) error
	TapAndType(ctx context.Context, deviceID string, x, y, settleMs int, text string, submit bool) error
}

// Orchestrator wires the layout snapshot source and the input dispatcher
// together. It is the single dependency of every Tester verb.
type Orchestrator struct {
	Layout layoutSource
	Input  inputDispatcher
}

// New builds an Orchestrator. Callers pass the concrete *ui.LayoutClient and
// *input.Client; they satisfy layoutSource/inputDispatcher implicitly.
func New(layout layoutSource, in inputDispatcher) *Orchestrator {
	return &Orchestrator{Layout: layout, Input: in}
}

// MatchResult is included in every action / assertion response so the LLM
// agent sees what was actually matched.
type MatchResult struct {
	Found    bool        `json:"found"`
	Element  *ui.Element `json:"element,omitempty"`
	Count    int         `json:"count"`
	Reason   string      `json:"reason,omitempty"`
	Attempts int         `json:"attempts,omitempty"`
}

// fetchAndFind snapshots the device tree once and returns either the matched
// node or a structured failure with diagnostic info.
func (o *Orchestrator) fetchAndFind(ctx context.Context, deviceID string, m *matcher.Matcher) (ui.Element, []ui.Element, error) {
	if m == nil || m.IsEmpty() {
		return ui.Element{}, nil, matcher.ErrEmptyMatcher
	}
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return ui.Element{}, nil, fmt.Errorf("snapshotting UI: %w", err)
	}
	return findInRoot(root, m)
}

// findInRoot applies m against an already-fetched snapshot — no subprocess
// call. Use this instead of fetchAndFind when a caller needs to match more
// than one selector (or re-verify one) against the same on-screen state,
// to avoid re-fetching the tree for state that hasn't changed.
func findInRoot(root ui.Element, m *matcher.Matcher) (ui.Element, []ui.Element, error) {
	if m == nil || m.IsEmpty() {
		return ui.Element{}, nil, matcher.ErrEmptyMatcher
	}
	all, err := matcher.FindAll(root, m)
	if err != nil {
		return ui.Element{}, nil, err
	}
	if len(all) == 0 {
		return ui.Element{}, nil, matcher.ErrNotFound
	}
	idx := m.Instance
	if idx < 0 || idx >= len(all) {
		return ui.Element{}, all, fmt.Errorf("%w: matched %d, instance %d requested", matcher.ErrNotFound, len(all), idx)
	}
	return all[idx], all, nil
}

// CenterOf returns the centre point of an element's bounds.
func CenterOf(e ui.Element) (int, int) {
	return e.Bounds.X + e.Bounds.Width/2, e.Bounds.Y + e.Bounds.Height/2
}

// SnapshotTree exposes the layout source so tools that want a one-shot tree
// (e.g. find_node, print_tree) can call it without re-creating an Orchestrator.
func (o *Orchestrator) SnapshotTree(ctx context.Context, deviceID string) (ui.Element, error) {
	return o.Layout.Tree(ctx, deviceID)
}

// pollUntil repeatedly invokes check until it returns ok=true or the
// deadline elapses. Returns the number of attempts consumed.
func pollUntil(ctx context.Context, timeoutMs, intervalMs int, check func(context.Context) (bool, error)) (int, bool, error) {
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if intervalMs <= 0 {
		intervalMs = 250
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	attempts := 0
	for {
		attempts++
		ok, err := check(ctx)
		if err == nil && ok {
			return attempts, true, nil
		}
		if time.Now().After(deadline) {
			return attempts, false, err
		}
		select {
		case <-ctx.Done():
			return attempts, false, ctx.Err()
		case <-time.After(time.Duration(intervalMs) * time.Millisecond):
		}
	}
}
