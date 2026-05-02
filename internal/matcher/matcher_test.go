package matcher

import (
	"testing"

	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

func boolPtr(b bool) *bool { return &b }

func tree() ui.Element {
	mkBox := func(text, label, rid, class string, clickable, enabled, displayed bool) ui.Element {
		return ui.Element{
			Class:         class,
			Text:          text,
			Label:         label,
			ResourceID:    rid,
			Clickable:     clickable,
			Enabled:       enabled,
			VisibleToUser: displayed,
			Bounds:        ui.Bounds{X: 10, Y: 10, Width: 100, Height: 50},
		}
	}
	root := ui.Element{
		Class:         "FrameLayout",
		Bounds:        ui.Bounds{Width: 1080, Height: 2400},
		Enabled:       true,
		VisibleToUser: true,
		Children: []ui.Element{
			mkBox("Hello", "", "com.example:id/title", "android.widget.TextView", false, true, true),
			mkBox("Login", "Login", "com.example:id/loginBtn", "android.widget.Button", true, true, true),
			mkBox("Submit", "Send the form", "com.example:id/submitBtn", "android.widget.Button", true, false, true),
			{
				Class:         "android.widget.ScrollView",
				Scrollable:    true,
				Enabled:       true,
				VisibleToUser: true,
				Bounds:        ui.Bounds{Width: 1080, Height: 1000},
				Children: []ui.Element{
					mkBox("Item 1", "", "com.example:id/item", "android.widget.TextView", false, true, true),
					mkBox("Item 2", "", "com.example:id/item", "android.widget.TextView", false, true, true),
				},
			},
			// Hidden node
			{
				Class:         "android.widget.View",
				Text:          "Hidden",
				Enabled:       true,
				Bounds:        ui.Bounds{},
				VisibleToUser: false,
			},
		},
	}
	return root
}

func TestMatch_TextAndProperties(t *testing.T) {
	root := tree()
	cases := []struct {
		name string
		m    Matcher
		want int
	}{
		{"by exact text", Matcher{Text: "Login"}, 1},
		{"by substring", Matcher{TextContains: "ogi"}, 1},
		{"by regex", Matcher{TextRegex: "^Item"}, 2},
		{"by content desc", Matcher{ContentDescription: "Send the form"}, 1},
		{"by resource id full", Matcher{ResourceID: "com.example:id/loginBtn"}, 1},
		{"by resource id suffix", Matcher{ResourceID: "loginBtn"}, 1},
		{"by testTag", Matcher{TestTag: "submitBtn"}, 1},
		{"by class substring", Matcher{ClassName: "Button"}, 2},
		{"clickable filter", Matcher{Clickable: boolPtr(true)}, 2},
		{"enabled false", Matcher{Enabled: boolPtr(false)}, 1},
		{"displayed true", Matcher{Displayed: boolPtr(true), Text: "Login"}, 1},
		{"displayed false", Matcher{Displayed: boolPtr(false), Text: "Hidden"}, 1},
		{"AllOf", Matcher{AllOf: []Matcher{{ClassName: "Button"}, {Clickable: boolPtr(true)}, {Enabled: boolPtr(true)}}}, 1},
		{"AnyOf", Matcher{AnyOf: []Matcher{{Text: "Login"}, {Text: "Submit"}}}, 2},
		{"Not", Matcher{ClassName: "Button", Not: &Matcher{Enabled: boolPtr(false)}}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FindAll(root, &tc.m)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if len(got) != tc.want {
				t.Fatalf("got %d, want %d (matches: %+v)", len(got), tc.want, got)
			}
		})
	}
}

func TestMatch_Hierarchy(t *testing.T) {
	root := tree()
	// Items inside the ScrollView
	got, err := FindAll(root, &Matcher{
		Text:        "Item 1",
		HasAncestor: &Matcher{Scrollable: boolPtr(true)},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 match, got %d", len(got))
	}

	// Sibling matcher: an Item with a sibling that's also an Item
	got, err = FindAll(root, &Matcher{
		Text:       "Item 1",
		HasSibling: &Matcher{Text: "Item 2"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("sibling match: got %d, want 1", len(got))
	}

	// hasDescendant
	got, err = FindAll(root, &Matcher{
		ClassName:     "ScrollView",
		HasDescendant: &Matcher{Text: "Item 2"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("descendant match: got %d, want 1", len(got))
	}
}

func TestEmptyMatcher(t *testing.T) {
	if _, err := FindAll(tree(), &Matcher{}); err == nil {
		t.Fatal("expected error for empty matcher")
	}
}

func TestInstance(t *testing.T) {
	root := tree()
	first, err := Find(root, &Matcher{ResourceID: "item"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != "Item 1" {
		t.Fatalf("first: %q", first.Text)
	}
	second, err := Find(root, &Matcher{ResourceID: "item", Instance: 1})
	if err != nil {
		t.Fatal(err)
	}
	if second.Text != "Item 2" {
		t.Fatalf("second: %q", second.Text)
	}
	if _, err := Find(root, &Matcher{ResourceID: "item", Instance: 5}); err == nil {
		t.Fatal("expected out-of-range error")
	}
}
