package testing

import (
	"context"
	"sync"

	"github.com/randheer094/velocity-test-mobile/internal/input"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// fakeLayout is a test double for layoutSource. By default it returns tree
// on every call; set treeFunc for call-number-dependent behavior (e.g.
// simulating a tree that changes across polls, or an error that persists).
type fakeLayout struct {
	mu       sync.Mutex
	calls    int
	tree     ui.Element
	err      error
	treeFunc func(call int) (ui.Element, error)
}

func (f *fakeLayout) Tree(ctx context.Context, deviceID string) (ui.Element, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.mu.Unlock()
	if f.treeFunc != nil {
		return f.treeFunc(call)
	}
	return f.tree, f.err
}

func (f *fakeLayout) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// inputCall records one method invocation on fakeInput for assertions.
type inputCall struct {
	method string
	args   []any
}

// fakeInput is a test double for inputDispatcher. Configure per-method
// errors via errs (keyed by method name); every call is recorded in calls
// regardless of whether it returns an error.
type fakeInput struct {
	mu    sync.Mutex
	calls []inputCall
	errs  map[string]error
}

func (f *fakeInput) record(method string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, inputCall{method: method, args: args})
	if f.errs == nil {
		return nil
	}
	return f.errs[method]
}

func (f *fakeInput) callNames() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	names := make([]string, len(f.calls))
	for i, c := range f.calls {
		names[i] = c.method
	}
	return names
}

func (f *fakeInput) callCountOf(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c.method == method {
			n++
		}
	}
	return n
}

func (f *fakeInput) Tap(ctx context.Context, deviceID string, x, y int) error {
	return f.record("Tap", x, y)
}

func (f *fakeInput) DoubleTap(ctx context.Context, deviceID string, x, y int) error {
	return f.record("DoubleTap", x, y)
}

func (f *fakeInput) LongPress(ctx context.Context, deviceID string, x, y, durationMs int) error {
	return f.record("LongPress", x, y, durationMs)
}

func (f *fakeInput) Drag(ctx context.Context, deviceID string, fromX, fromY, toX, toY, durationMs int) error {
	return f.record("Drag", fromX, fromY, toX, toY, durationMs)
}

func (f *fakeInput) Swipe(ctx context.Context, deviceID string, dir input.Direction, screenW, screenH int, anchorX, anchorY, distance, durationMs int) error {
	return f.record("Swipe", dir, screenW, screenH, anchorX, anchorY, distance, durationMs)
}

func (f *fakeInput) PressButton(ctx context.Context, deviceID, name string) error {
	return f.record("PressButton", name)
}

func (f *fakeInput) PressButtonRepeat(ctx context.Context, deviceID, name string, count int) error {
	return f.record("PressButtonRepeat", name, count)
}

func (f *fakeInput) PressKeyCombination(ctx context.Context, deviceID string, names ...string) error {
	return f.record("PressKeyCombination", names)
}

func (f *fakeInput) TypeKeys(ctx context.Context, deviceID, text string, submit bool) error {
	return f.record("TypeKeys", text, submit)
}

func (f *fakeInput) TapAndPressButton(ctx context.Context, deviceID string, x, y, settleMs int, name string) error {
	return f.record("TapAndPressButton", x, y, settleMs, name)
}

func (f *fakeInput) TapAndType(ctx context.Context, deviceID string, x, y, settleMs int, text string, submit bool) error {
	return f.record("TapAndType", x, y, settleMs, text, submit)
}
