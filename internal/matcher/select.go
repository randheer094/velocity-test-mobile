package matcher

import (
	"errors"
	"fmt"

	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// ErrEmptyMatcher is returned when a tool is called with no selection criteria.
var ErrEmptyMatcher = errors.New("matcher is empty: supply at least one of text, contentDescription, resourceId, testTag, className, hint, etc.")

// ErrNotFound is returned when no node satisfies the matcher.
var ErrNotFound = errors.New("no element matched the selector")

// FindAll returns every element in root that satisfies m, including
// hierarchy combinators (HasAncestor, HasDescendant, HasParent, HasSibling).
// Order is depth-first traversal of the original tree.
func FindAll(root ui.Element, m *Matcher) ([]ui.Element, error) {
	if m == nil || m.IsEmpty() {
		return nil, ErrEmptyMatcher
	}
	flat := flattenWithParents(root)
	var matches []ui.Element
	for i := range flat {
		ok, err := matchAtIndex(i, flat, m)
		if err != nil {
			return nil, err
		}
		if ok {
			matches = append(matches, flat[i].elem)
		}
	}
	return matches, nil
}

// Find returns the Nth matching element (m.Instance, default 0).
func Find(root ui.Element, m *Matcher) (ui.Element, error) {
	matches, err := FindAll(root, m)
	if err != nil {
		return ui.Element{}, err
	}
	if len(matches) == 0 {
		return ui.Element{}, ErrNotFound
	}
	idx := m.Instance
	if idx < 0 || idx >= len(matches) {
		return ui.Element{}, fmt.Errorf("%w: matched %d elements but instance %d requested", ErrNotFound, len(matches), idx)
	}
	return matches[idx], nil
}

// Count returns how many elements satisfy the matcher.
func Count(root ui.Element, m *Matcher) (int, error) {
	matches, err := FindAll(root, m)
	if err != nil {
		return 0, err
	}
	return len(matches), nil
}

// pathItem holds a node together with its position in the flattened tree.
// `parent` is the flat index of the immediate parent, -1 for the root.
type pathItem struct {
	elem   ui.Element
	parent int
	depth  int
}

func flattenWithParents(root ui.Element) []pathItem {
	out := []pathItem{}
	var walk func(e ui.Element, parent int, depth int)
	walk = func(e ui.Element, parent int, depth int) {
		idx := len(out)
		out = append(out, pathItem{elem: e, parent: parent, depth: depth})
		for _, c := range e.Children {
			walk(c, idx, depth+1)
		}
	}
	walk(root, -1, 0)
	return out
}

// matchAtIndex evaluates m against the node at flat[idx]. It is the
// canonical entry point for tree-aware predicates: every combinator that
// recurses (HasAncestor, HasDescendant, HasParent, HasSibling) ultimately
// calls back into matchAtIndex with the candidate's flat index.
//
// Carrying the index (rather than re-deriving it from the element's content)
// makes correctness independent of duplicate sibling content — two nodes
// with identical text/bounds/class are still distinct paths.
func matchAtIndex(idx int, flat []pathItem, m *Matcher) (bool, error) {
	item := flat[idx]

	// Local predicates first.
	ok, err := Match(item.elem, m)
	if err != nil || !ok {
		return ok, err
	}

	// Tree-position predicates (need the surrounding flat slice).
	if m.IsRoot != nil {
		isRoot := item.parent < 0
		if isRoot != *m.IsRoot {
			return false, nil
		}
	}
	if m.ParentIndex != nil {
		if item.parent < 0 {
			return false, nil
		}
		// Position among the parent's direct children.
		pos := -1
		count := 0
		for i, other := range flat {
			if other.parent != item.parent {
				continue
			}
			if i == idx {
				pos = count
				break
			}
			count++
		}
		if pos != *m.ParentIndex {
			return false, nil
		}
	}

	// Visibility refinements relative to the root viewport.
	if m.CompletelyDisplayed != nil || m.DisplayingAtLeastPercent > 0 {
		root := flat[0].elem
		visible := visibleArea(item.elem, root)
		total := area(item.elem.Bounds)
		if total == 0 {
			if m.CompletelyDisplayed != nil && *m.CompletelyDisplayed {
				return false, nil
			}
			if m.DisplayingAtLeastPercent > 0 {
				return false, nil
			}
		} else {
			if m.CompletelyDisplayed != nil {
				want := *m.CompletelyDisplayed
				got := visible == total
				if got != want {
					return false, nil
				}
			}
			if m.DisplayingAtLeastPercent > 0 {
				pct := 100 * visible / total
				if pct < m.DisplayingAtLeastPercent {
					return false, nil
				}
			}
		}
	}

	// Hierarchy combinators.
	if m.HasParent != nil {
		if item.parent < 0 {
			return false, nil
		}
		ok, err := matchAtIndex(item.parent, flat, m.HasParent)
		if err != nil || !ok {
			return ok, err
		}
	}

	if m.HasAncestor != nil {
		matched := false
		for p := item.parent; p >= 0; p = flat[p].parent {
			ok, err := matchAtIndex(p, flat, m.HasAncestor)
			if err != nil {
				return false, err
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}

	if m.HasDescendant != nil {
		matched := false
		for i := range flat {
			if i == idx || !isDescendantOf(flat, i, idx) {
				continue
			}
			ok, err := matchAtIndex(i, flat, m.HasDescendant)
			if err != nil {
				return false, err
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}

	if m.HasSibling != nil {
		if item.parent < 0 {
			return false, nil
		}
		matched := false
		for i, other := range flat {
			if i == idx || other.parent != item.parent {
				continue
			}
			ok, err := matchAtIndex(i, flat, m.HasSibling)
			if err != nil {
				return false, err
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}

func isDescendantOf(flat []pathItem, i, ancestor int) bool {
	for p := flat[i].parent; p >= 0; p = flat[p].parent {
		if p == ancestor {
			return true
		}
	}
	return false
}

// area returns the rectangular area of bounds.
func area(b ui.Bounds) int { return b.Width * b.Height }

// visibleArea returns the area of the intersection between e's bounds and
// root's bounds (used as the screen viewport).
func visibleArea(e, root ui.Element) int {
	x1 := max(e.Bounds.X, root.Bounds.X)
	y1 := max(e.Bounds.Y, root.Bounds.Y)
	x2 := min(e.Bounds.X+e.Bounds.Width, root.Bounds.X+root.Bounds.Width)
	y2 := min(e.Bounds.Y+e.Bounds.Height, root.Bounds.Y+root.Bounds.Height)
	if x2 <= x1 || y2 <= y1 {
		return 0
	}
	return (x2 - x1) * (y2 - y1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
