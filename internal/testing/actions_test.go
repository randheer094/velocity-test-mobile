package testing

import (
	"context"
	"errors"
	"testing"

	"github.com/randheer094/velocity-test-mobile/internal/matcher"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

func fieldElem(text string, focused bool) ui.Element {
	return ui.Element{
		Class: "EditText", Text: text, ResourceID: "field", Focused: focused,
		Enabled: true, VisibleToUser: true, Clickable: true,
		Bounds: ui.Bounds{X: 100, Y: 200, Width: 100, Height: 50}, // center (150,225)
	}
}

func fieldTree(text string, focused bool) ui.Element {
	return ui.Element{
		Class: "FrameLayout", Enabled: true, VisibleToUser: true,
		Bounds:   ui.Bounds{Width: 1000, Height: 1000},
		Children: []ui.Element{fieldElem(text, focused)},
	}
}

func fieldMatcher() *matcher.Matcher { return &matcher.Matcher{ResourceID: "field"} }

func TestClick(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.Click(context.Background(), "", fieldMatcher())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK || res.X != 150 || res.Y != 225 {
		t.Fatalf("res: %+v", res)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "Tap" {
		t.Fatalf("calls: %v", got)
	}
}

func TestClick_NotFound(t *testing.T) {
	fl := &fakeLayout{tree: ui.Element{Class: "FrameLayout"}}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.Click(context.Background(), "", fieldMatcher())
	if err == nil {
		t.Fatal("expected error")
	}
	if res.OK {
		t.Fatalf("res: %+v", res)
	}
	if len(fi.callNames()) != 0 {
		t.Fatalf("input should not be dispatched: %v", fi.callNames())
	}
}

func TestDoubleClick(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	if _, err := o.DoubleClick(context.Background(), "", fieldMatcher()); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "DoubleTap" {
		t.Fatalf("calls: %v", got)
	}
}

func TestLongClick(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	if _, err := o.LongClick(context.Background(), "", fieldMatcher(), 0); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := fi.calls; len(got) != 1 || got[0].method != "LongPress" || got[0].args[2] != 800 {
		t.Fatalf("calls: %+v", got)
	}
}

func TestTypeText_NotFocused_UsesTapAndType(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.TypeText(context.Background(), "", fieldMatcher(), "hello", true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "TapAndType" {
		t.Fatalf("calls: %v", got)
	}
}

func TestTypeText_Focused_UsesTypeKeysOnly(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", true)}
	fi := &fakeInput{}
	o := New(fl, fi)

	if _, err := o.TypeText(context.Background(), "", fieldMatcher(), "hello", false); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "TypeKeys" {
		t.Fatalf("calls: %v", got)
	}
}

func TestClearTextFieldElem_CtrlA_Succeeds(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("hello", true)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.ClearText(context.Background(), "", fieldMatcher())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	got := fi.callNames()
	want := []string{"PressKeyCombination", "PressButton"}
	if len(got) != len(want) {
		t.Fatalf("calls: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("calls: %v, want %v", got, want)
		}
	}
}

func TestClearTextFieldElem_CtrlAUnsupported_FallsBackToMoveEndDel(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("hello", true)}
	fi := &fakeInput{errs: map[string]error{"PressKeyCombination": errors.New("unsupported")}}
	o := New(fl, fi)

	res, err := o.ClearText(context.Background(), "", fieldMatcher())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	got := fi.callNames()
	want := []string{"PressKeyCombination", "PressButton", "PressButtonRepeat"}
	if len(got) != len(want) {
		t.Fatalf("calls: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("calls: %v, want %v", got, want)
		}
	}
	// count = len("hello") + 4 = 9
	if n := fi.calls[2].args[1]; n != 9 {
		t.Fatalf("DEL count = %v, want 9", n)
	}
}

func TestClearTextFieldElem_NotFocused_TapsFirst(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("hi", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	if _, err := o.ClearText(context.Background(), "", fieldMatcher()); err != nil {
		t.Fatalf("err: %v", err)
	}
	got := fi.callNames()
	if len(got) < 1 || got[0] != "Tap" {
		t.Fatalf("expected Tap first: %v", got)
	}
}

func TestReplaceText_FusesClearAndType(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("old", true)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.ReplaceText(context.Background(), "", fieldMatcher(), "new", false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	// clear (CTRL+A, DEL) then type (TypeKeys, since clearTextFieldElem's
	// success implies focus, so typeTextElem takes the focused path).
	got := fi.callNames()
	want := []string{"PressKeyCombination", "PressButton", "TypeKeys"}
	if len(got) != len(want) {
		t.Fatalf("calls: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("calls: %v, want %v", got, want)
		}
	}
	if fl.callCount() != 1 {
		t.Fatalf("Layout.Tree called %d times, want 1 (no redundant re-fetch)", fl.callCount())
	}
}

func TestSubmit_NotFocused_UsesTapAndPressButton(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	if _, err := o.Submit(context.Background(), "", fieldMatcher()); err != nil {
		t.Fatalf("err: %v", err)
	}
	got := fi.calls
	if len(got) != 1 || got[0].method != "TapAndPressButton" || got[0].args[3] != "ENTER" {
		t.Fatalf("calls: %+v", got)
	}
}

func TestSubmit_Focused_PressesButtonOnly(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", true)}
	fi := &fakeInput{}
	o := New(fl, fi)

	if _, err := o.Submit(context.Background(), "", fieldMatcher()); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "PressButton" {
		t.Fatalf("calls: %v", got)
	}
}

func scrollableTree(targetVisible bool) ui.Element {
	target := ui.Element{Class: "Item", Text: "Target", Enabled: true, VisibleToUser: targetVisible,
		Bounds: ui.Bounds{X: 10, Y: 10, Width: 50, Height: 50}}
	if !targetVisible {
		target = ui.Element{} // absent
	}
	container := ui.Element{
		Class: "ScrollView", Scrollable: true, Enabled: true, VisibleToUser: true,
		Bounds: ui.Bounds{Width: 500, Height: 500},
	}
	if targetVisible {
		container.Children = []ui.Element{target}
	}
	return ui.Element{Class: "Root", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{Width: 500, Height: 500}, Children: []ui.Element{container}}
}

func TestScrollTo_FindsImmediately(t *testing.T) {
	fl := &fakeLayout{tree: scrollableTree(true)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.ScrollTo(context.Background(), "", &matcher.Matcher{Text: "Target"}, ScrollOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	if len(fi.callNames()) != 0 {
		t.Fatalf("expected no swipes when already visible: %v", fi.callNames())
	}
}

func TestScrollTo_ExhaustsAttempts(t *testing.T) {
	fl := &fakeLayout{tree: scrollableTree(false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	_, err := o.ScrollTo(context.Background(), "", &matcher.Matcher{Text: "Target"}, ScrollOptions{MaxAttempts: 2})
	if err == nil {
		t.Fatal("expected error when target never appears")
	}
	if n := fi.callCountOf("Swipe"); n != 2 {
		t.Fatalf("Swipe called %d times, want 2 (MaxAttempts)", n)
	}
}

func dragTree() ui.Element {
	return ui.Element{
		Class: "Root", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{Width: 1000, Height: 1000},
		Children: []ui.Element{
			{Class: "Item", Text: "Src", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{X: 0, Y: 0, Width: 100, Height: 100}},
			{Class: "Item", Text: "Dst", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{X: 800, Y: 800, Width: 100, Height: 100}},
		},
	}
}

func TestDragNode(t *testing.T) {
	fl := &fakeLayout{tree: dragTree()}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.DragNode(context.Background(), "", &matcher.Matcher{Text: "Src"}, &matcher.Matcher{Text: "Dst"}, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK || res.X != 850 || res.Y != 850 {
		t.Fatalf("res: %+v", res)
	}
	if fl.callCount() != 1 {
		t.Fatalf("Layout.Tree called %d times, want 1 (one fetch reused for both matchers)", fl.callCount())
	}
	if got := fi.calls; len(got) != 1 || got[0].method != "Drag" || got[0].args[4] != 600 {
		t.Fatalf("calls: %+v", got)
	}
}

func TestDragNode_ToNotFound(t *testing.T) {
	fl := &fakeLayout{tree: dragTree()}
	fi := &fakeInput{}
	o := New(fl, fi)

	_, err := o.DragNode(context.Background(), "", &matcher.Matcher{Text: "Src"}, &matcher.Matcher{Text: "Missing"}, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(fi.callNames()) != 0 {
		t.Fatalf("Drag should not dispatch when `to` is missing: %v", fi.callNames())
	}
}

func scrollIndexTree() ui.Element {
	elem := ui.Element{
		Class: "LazyColumn", ResourceID: "list", Scrollable: true, Enabled: true, VisibleToUser: true,
		Bounds: ui.Bounds{Width: 500, Height: 500},
	}
	return ui.Element{Class: "Root", Enabled: true, VisibleToUser: true, Bounds: ui.Bounds{Width: 500, Height: 500}, Children: []ui.Element{elem}}
}

func TestScrollToIndex(t *testing.T) {
	fl := &fakeLayout{tree: scrollIndexTree()}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.ScrollToIndex(context.Background(), "", &matcher.Matcher{ResourceID: "list"}, 3, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	if n := fi.callCountOf("Swipe"); n != 3 {
		t.Fatalf("Swipe called %d times, want 3", n)
	}
}

func TestScrollToIndex_NotScrollable(t *testing.T) {
	tree := scrollIndexTree()
	tree.Children[0].Scrollable = false
	fl := &fakeLayout{tree: tree}
	fi := &fakeInput{}
	o := New(fl, fi)

	_, err := o.ScrollToIndex(context.Background(), "", &matcher.Matcher{ResourceID: "list"}, 1, "")
	if err == nil {
		t.Fatal("expected error for non-scrollable element")
	}
	if len(fi.callNames()) != 0 {
		t.Fatalf("no swipe expected: %v", fi.callNames())
	}
}

func TestPerformKeyPress_NoModifier_NotFocused_UsesTapAndPressButton(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.PerformKeyPress(context.Background(), "", fieldMatcher(), "A", false, false, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "TapAndPressButton" {
		t.Fatalf("calls: %v", got)
	}
}

func TestPerformKeyPress_WithModifier_UsesKeyCombination(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.PerformKeyPress(context.Background(), "", fieldMatcher(), "A", true, false, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	got := fi.callNames()
	want := []string{"Tap", "PressKeyCombination"}
	if len(got) != len(want) {
		t.Fatalf("calls: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("calls: %v, want %v", got, want)
		}
	}
}

func TestPerformKeyPress_ModifierUnsupported_FallsBack(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", true)} // focused: skip tap
	fi := &fakeInput{errs: map[string]error{"PressKeyCombination": errors.New("unsupported")}}
	o := New(fl, fi)

	res, err := o.PerformKeyPress(context.Background(), "", fieldMatcher(), "A", true, false, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK || res.Reason == "" {
		t.Fatalf("res: %+v", res)
	}
	got := fi.callNames()
	want := []string{"PressKeyCombination", "PressButton"}
	if len(got) != len(want) {
		t.Fatalf("calls: %v, want %v", got, want)
	}
}

func TestPerformIMEAction_NotFocused(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.PerformIMEAction(context.Background(), "", fieldMatcher())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	got := fi.calls
	if len(got) != 1 || got[0].method != "TapAndPressButton" || got[0].args[2] != 0 {
		t.Fatalf("calls: %+v (want single TapAndPressButton with settleMs=0)", got)
	}
}

func TestPerformIMEAction_NoMatcher(t *testing.T) {
	fl := &fakeLayout{tree: fieldTree("", false)}
	fi := &fakeInput{}
	o := New(fl, fi)

	res, err := o.PerformIMEAction(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.OK {
		t.Fatalf("res: %+v", res)
	}
	if got := fi.callNames(); len(got) != 1 || got[0] != "PressButton" {
		t.Fatalf("calls: %v", got)
	}
}
