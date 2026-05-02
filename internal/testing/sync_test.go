package testing

import (
	"testing"

	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
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
