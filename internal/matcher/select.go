package matcher

import (
	"errors"
	"fmt"

	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
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
	for _, item := range flat {
		ok, err := matchAtPath(item, flat, m)
		if err != nil {
			return nil, err
		}
		if ok {
			matches = append(matches, item.elem)
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

// pathItem holds a node together with its ancestry chain so combinators
// can be evaluated cheaply.
type pathItem struct {
	elem       ui.Element
	parent     int // index in flat slice; -1 for root
	depth      int
	siblingsOf int // index of nearest ancestor that has multiple direct children including this branch
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

func matchAtPath(item pathItem, flat []pathItem, m *Matcher) (bool, error) {
	// Local predicates first.
	ok, err := Match(item.elem, m)
	if err != nil || !ok {
		return ok, err
	}

	// Hierarchy combinators.
	if m.HasParent != nil {
		if item.parent < 0 {
			return false, nil
		}
		ok, err := matchAtPath(flat[item.parent], flat, m.HasParent)
		if err != nil || !ok {
			return ok, err
		}
	}

	if m.HasAncestor != nil {
		matched := false
		for p := item.parent; p >= 0; p = flat[p].parent {
			ok, err := matchAtPath(flat[p], flat, m.HasAncestor)
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
			if i == indexOf(item, flat) {
				continue
			}
			if !isDescendantOf(flat, i, indexOf(item, flat)) {
				continue
			}
			ok, err := matchAtPath(flat[i], flat, m.HasDescendant)
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
		myIdx := indexOf(item, flat)
		for i, other := range flat {
			if i == myIdx || other.parent != item.parent {
				continue
			}
			ok, err := matchAtPath(other, flat, m.HasSibling)
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

func indexOf(item pathItem, flat []pathItem) int {
	// Stable: the slice is built once per Find call; pointer-style identity
	// isn't available so we fall back to "the first item with this depth and
	// parent and matching elem header". Good enough — flatten preserves order.
	for i, it := range flat {
		if it.depth == item.depth && it.parent == item.parent &&
			it.elem.Bounds == item.elem.Bounds &&
			it.elem.Class == item.elem.Class &&
			it.elem.Text == item.elem.Text {
			return i
		}
	}
	return -1
}

func isDescendantOf(flat []pathItem, i, ancestor int) bool {
	for p := flat[i].parent; p >= 0; p = flat[p].parent {
		if p == ancestor {
			return true
		}
	}
	return false
}
