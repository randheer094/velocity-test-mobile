package testing

import (
	"context"
	"errors"
	"testing"

	"github.com/randheer094/velocity-test-mobile/internal/matcher"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

func TestHashTree_Stable(t *testing.T) {
	a := ui.Element{Class: "FrameLayout", Bounds: ui.Bounds{Width: 100, Height: 100}, Enabled: true,
		Children: []ui.Element{
			{Class: "TextView", Text: "Hi", Bounds: ui.Bounds{X: 1, Y: 2, Width: 3, Height: 4}, Enabled: true},
		},
	}
	b := ui.Element{Class: "FrameLayout", Bounds: ui.Bounds{Width: 100, Height: 100}, Enabled: true,
		Children: []ui.Element{
			{Class: "TextView", Text: "Hi", Bounds: ui.Bounds{X: 1, Y: 2, Width: 3, Height: 4}, Enabled: true},
		},
	}
	if hashTree(a) != hashTree(b) {
		t.Fatalf("identical trees should hash equal")
	}
	c := ui.Element{Class: "FrameLayout", Bounds: ui.Bounds{Width: 100, Height: 100}, Enabled: true,
		Children: []ui.Element{
			{Class: "TextView", Text: "Hello", Bounds: ui.Bounds{X: 1, Y: 2, Width: 3, Height: 4}, Enabled: true},
		},
	}
	if hashTree(a) == hashTree(c) {
		t.Fatalf("different text should hash differently")
	}
}

var errFetch = errors.New("simulated fetch failure")

// TestWaitUntilVisible_PersistentFetchError_SurfacesRealError guards the
// Priority-0 bug fix: a wait tool used to swallow a persistent Layout.Tree
// failure and report a misleading "timed out" reason instead of the real
// cause. This must fail against the old code and pass against the fix.
func TestWaitUntilVisible_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	res, err := o.WaitUntilVisible(context.Background(), "", &matcher.Matcher{Text: "X"}, 30, 5)
	if err == nil {
		t.Fatal("expected the real fetch error to be returned, got nil")
	}
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
	if res.OK {
		t.Fatalf("res: %+v", res)
	}
}

// TestWaitUntilVisible_GenuineTimeout_KeepsDescriptiveReason confirms the
// fix didn't regress the legitimate "never became true" case: no fetch
// error, but the condition just never held — nil error, descriptive Reason.
func TestWaitUntilVisible_GenuineTimeout_KeepsDescriptiveReason(t *testing.T) {
	fl := &fakeLayout{tree: ui.Element{Class: "Root"}} // no matching child, no error
	o := New(fl, &fakeInput{})

	res, err := o.WaitUntilVisible(context.Background(), "", &matcher.Matcher{Text: "X"}, 30, 5)
	if err != nil {
		t.Fatalf("expected nil error for a genuine timeout, got %v", err)
	}
	if res.OK {
		t.Fatalf("res: %+v", res)
	}
	if res.Reason == "" {
		t.Fatal("expected a descriptive Reason")
	}
}

func visibleAfter(n int) func(call int) (ui.Element, error) {
	target := ui.Element{Class: "TextView", Text: "X", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{Width: 10, Height: 10}}
	empty := ui.Element{Class: "Root", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{Width: 100, Height: 100}}
	full := ui.Element{Class: "Root", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{Width: 100, Height: 100}, Children: []ui.Element{target}}
	return func(call int) (ui.Element, error) {
		if call >= n {
			return full, nil
		}
		return empty, nil
	}
}

func TestWaitUntilVisible_BecomesVisible(t *testing.T) {
	fl := &fakeLayout{treeFunc: visibleAfter(3)}
	o := New(fl, &fakeInput{})

	res, err := o.WaitUntilVisible(context.Background(), "", &matcher.Matcher{Text: "X"}, 2000, 5)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK || res.Element == nil {
		t.Fatalf("res: %+v", res)
	}
}

func TestWaitUntilNotVisible_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	_, err := o.WaitUntilNotVisible(context.Background(), "", &matcher.Matcher{Text: "X"}, 30, 5)
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
}

func TestWaitUntilText_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	_, err := o.WaitUntilText(context.Background(), "", &matcher.Matcher{ResourceID: "x"}, "hi", 30, 5)
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
}

func TestWaitUntilAtLeastOneExists_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	_, err := o.WaitUntilAtLeastOneExists(context.Background(), "", &matcher.Matcher{ResourceID: "x"}, 1, 30, 5)
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
}

func TestWaitUntilCount_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	_, err := o.WaitUntilCount(context.Background(), "", &matcher.Matcher{ResourceID: "x"}, 2, 30, 5)
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
}

func TestWaitUntilEnabled_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	_, err := o.WaitUntilEnabled(context.Background(), "", &matcher.Matcher{ResourceID: "x"}, 30, 5)
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
}

func TestWaitForIdle_PersistentFetchError_SurfacesRealError(t *testing.T) {
	fl := &fakeLayout{err: errFetch}
	o := New(fl, &fakeInput{})

	_, err := o.WaitForIdle(context.Background(), "", 30, 20)
	if !errors.Is(err, errFetch) {
		t.Fatalf("err = %v, want wrapping %v", err, errFetch)
	}
}

func TestWaitForIdle_StabilizesEventually(t *testing.T) {
	stable := ui.Element{Class: "Root", Bounds: ui.Bounds{Width: 10, Height: 10}}
	fl := &fakeLayout{tree: stable}
	o := New(fl, &fakeInput{})

	res, err := o.WaitForIdle(context.Background(), "", 2000, 30)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
}

func TestWaitForIdle_GenuineTimeout_NoErrorButNotOK(t *testing.T) {
	call := 0
	fl := &fakeLayout{treeFunc: func(c int) (ui.Element, error) {
		call++
		// A different tree every time (via a changing bounds field) never
		// stabilizes, and never errors.
		return ui.Element{Class: "Root", Bounds: ui.Bounds{Width: call, Height: 10}}, nil
	}}
	o := New(fl, &fakeInput{})

	res, err := o.WaitForIdle(context.Background(), "", 30, 10)
	if err != nil {
		t.Fatalf("expected nil error for a genuine non-stabilizing timeout, got %v", err)
	}
	if res.OK {
		t.Fatalf("res: %+v", res)
	}
	if res.Reason == "" {
		t.Fatal("expected a descriptive Reason")
	}
}
