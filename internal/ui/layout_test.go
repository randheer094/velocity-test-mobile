package ui

import (
	"context"
	"errors"
	"testing"

	"github.com/randheer094/velocity-test-mobile/internal/androidcli"
)

func TestTree_RequiresAndroidCLI(t *testing.T) {
	l := NewLayoutClient(nil, nil)
	_, err := l.Tree(context.Background(), "")
	if !errors.Is(err, androidcli.ErrNotInstalled) {
		t.Errorf("Tree() with nil AndroidCLI = %v, want androidcli.ErrNotInstalled", err)
	}
}

func TestParseBoundsString(t *testing.T) {
	cases := map[string]Bounds{
		"[0,0][100,200]":   {X: 0, Y: 0, Width: 100, Height: 200},
		"[10,20][210,420]": {X: 10, Y: 20, Width: 200, Height: 400},
		"[-5,-5][5,5]":     {X: -5, Y: -5, Width: 10, Height: 10},
	}
	for in, want := range cases {
		got, ok := parseBoundsString(in)
		if !ok {
			t.Errorf("parseBoundsString(%q) failed", in)
			continue
		}
		if got != want {
			t.Errorf("parseBoundsString(%q) = %+v, want %+v", in, got, want)
		}
	}
	if _, ok := parseBoundsString("garbage"); ok {
		t.Error("expected failure for non-bounds string")
	}
}

// flatArrayLayoutSample is a redacted excerpt of real `android layout
// --pretty` output from android CLI v1.0.15985488, captured against an
// emulator's Settings app. This CLI version emits a flat JSON array of
// elements instead of a nested object tree: no `class`/`children`, `bounds`
// only on a few container elements, most leaves carry only `center`.
const flatArrayLayoutSample = `[
  {
    "interactions": ["scrollable"],
    "center": "[540,1168]",
    "bounds": "[0,63][1080,2274]",
    "resource-id": "settings_homepage_container",
    "key": 3506402
  },
  {
    "interactions": ["focusable", "scrollable"],
    "center": "[540,1489]",
    "bounds": "[0,705][1080,2274]",
    "resource-id": "main_content_scrollable_container",
    "key": 3506402
  },
  {
    "interactions": ["clickable", "focusable"],
    "off-screen": true,
    "key": 3506402
  },
  {
    "text": "Network & internet",
    "center": "[407,794]",
    "resource-id": "title",
    "key": 3506402
  },
  {
    "content-desc": "Profile picture, double tap to open Google Account",
    "interactions": ["clickable", "focusable"],
    "center": "[954,273]",
    "resource-id": "account_avatar",
    "key": 3506402
  }
]`

func TestParseAndroidCLILayout_FlatArray(t *testing.T) {
	root, err := parseAndroidCLILayout([]byte(flatArrayLayoutSample))
	if err != nil {
		t.Fatalf("parseAndroidCLILayout() error = %v", err)
	}
	if len(root.Children) != 5 {
		t.Fatalf("got %d children, want 5", len(root.Children))
	}

	// The synthetic root's Bounds must be the bounding box of the
	// real-bounds children (here, the outer scrollable container), not
	// zero-value. matcher.CompletelyDisplayed/DisplayingAtLeastPercent
	// measure candidates against their root as the viewport, so a
	// zero-area root would make those checks always fail.
	if want := (Bounds{X: 0, Y: 63, Width: 1080, Height: 2211}); root.Bounds != want {
		t.Errorf("root.Bounds = %+v, want %+v (union of real-bounds children)", root.Bounds, want)
	}

	container := root.Children[0]
	if !container.Scrollable {
		t.Error("scrollable container should have Scrollable=true")
	}
	if container.Bounds != (Bounds{X: 0, Y: 63, Width: 1080, Height: 2211}) {
		t.Errorf("container bounds = %+v, want real rect from bounds string", container.Bounds)
	}

	offScreen := root.Children[2]
	if offScreen.VisibleToUser {
		t.Error("off-screen element should have VisibleToUser=false")
	}
	if offScreen.Bounds != (Bounds{}) {
		t.Errorf("off-screen element with no center/bounds should have zero Bounds, got %+v", offScreen.Bounds)
	}

	title := root.Children[3]
	if title.Text != "Network & internet" {
		t.Errorf("title text = %q", title.Text)
	}
	if title.Bounds.Width <= 0 || title.Bounds.Height <= 0 {
		t.Errorf("leaf element with only center should get synthesized non-zero bounds, got %+v", title.Bounds)
	}
	cx, cy := title.Bounds.X+title.Bounds.Width/2, title.Bounds.Y+title.Bounds.Height/2
	if cx != 407 || cy != 794 {
		t.Errorf("synthesized bounds center = (%d,%d), want (407,794)", cx, cy)
	}

	avatar := root.Children[4]
	if avatar.Label != "Profile picture, double tap to open Google Account" {
		t.Errorf("avatar label = %q", avatar.Label)
	}
	if !avatar.Clickable {
		t.Error("avatar should be clickable")
	}

	flat := Flatten(root)
	if len(flat) != 4 {
		t.Errorf("Flatten() returned %d elements, want 4 (off-screen element dropped)", len(flat))
	}
}

func TestParseCenterString(t *testing.T) {
	x, y, ok := parseCenterString("[540,1168]")
	if !ok || x != 540 || y != 1168 {
		t.Errorf("parseCenterString(%q) = (%d, %d, %v)", "[540,1168]", x, y, ok)
	}
	if _, _, ok := parseCenterString(""); ok {
		t.Error("expected failure for empty string")
	}
}

func TestMatch(t *testing.T) {
	root := Element{
		Children: []Element{
			{Class: "android.widget.TextView", Text: "Hello world", ResourceID: "id/greeting", Bounds: Bounds{Width: 1, Height: 1}},
			{Class: "android.widget.Button", Label: "Login", Bounds: Bounds{Width: 1, Height: 1}},
		},
	}
	if got, ok := Match(root, Predicate{Text: "world"}); !ok || got.Text != "Hello world" {
		t.Errorf("substring match failed: %+v", got)
	}
	if got, ok := Match(root, Predicate{Text: "/^Hello\\s/"}); !ok || got.Text != "Hello world" {
		t.Errorf("regex match failed: %+v", got)
	}
	if got, ok := Match(root, Predicate{ContentDesc: "Login"}); !ok || got.Label != "Login" {
		t.Errorf("contentDesc match failed: %+v", got)
	}
	if _, ok := Match(root, Predicate{ResourceID: "id/missing"}); ok {
		t.Error("should not match missing resource id")
	}
	if _, ok := Match(root, Predicate{}); ok {
		t.Error("empty predicate should never match")
	}
}

func TestParseAndroidCLILayout(t *testing.T) {
	data := []byte(`{
		"class": "FrameLayout",
		"bounds": [0, 0, 1080, 2400],
		"children": [
			{"class": "TextView", "text": "Hi", "bounds": [10, 20, 110, 60], "clickable": true},
			{"class": "Button", "contentDesc": "Send", "bounds": {"left":0,"top":0,"right":50,"bottom":50}, "enabled": false}
		]
	}`)
	root, err := parseAndroidCLILayout(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if root.Class != "FrameLayout" {
		t.Errorf("class: %q", root.Class)
	}
	if len(root.Children) != 2 {
		t.Fatalf("children: %d", len(root.Children))
	}
	if root.Children[0].Bounds != (Bounds{X: 10, Y: 20, Width: 100, Height: 40}) {
		t.Errorf("text bounds: %+v", root.Children[0].Bounds)
	}
	if root.Children[1].Label != "Send" {
		t.Errorf("button label: %q", root.Children[1].Label)
	}
}
