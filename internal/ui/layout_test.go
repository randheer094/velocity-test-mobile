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
