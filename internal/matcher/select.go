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
	flat, childrenOf := flattenWithParents(root)
	var matches []ui.Element
	for i := range flat {
		ok, err := matchAtIndex(i, flat, childrenOf, m)
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
// `siblingIndex` and `subtreeEnd` are precomputed once per FindAll call so
// the tree-position combinators below (ParentIndex, HasDescendant,
// HasSibling) don't need to rescan the whole flattened tree per candidate.
type pathItem struct {
	elem         ui.Element
	parent       int
	depth        int
	siblingIndex int // position among the parent's direct children (0-based)
	subtreeEnd   int // largest flat index within this node's own subtree (inclusive)
}

// flattenWithParents does a pre-order DFS flatten. Pre-order means every
// node's descendants occupy a contiguous index range immediately after it
// ([idx+1, subtreeEnd]), which is what lets HasDescendant below iterate just
// the candidate's subtree instead of the entire tree.
//
// childrenOf[i] lists the flat indices of node i's direct children, in
// order — used by HasSibling to iterate just the candidate's siblings.
func flattenWithParents(root ui.Element) (flat []pathItem, childrenOf [][]int) {
	var walk func(e ui.Element, parent, depth, siblingIndex int) int
	walk = func(e ui.Element, parent, depth, siblingIndex int) int {
		idx := len(flat)
		flat = append(flat, pathItem{elem: e, parent: parent, depth: depth, siblingIndex: siblingIndex})
		childrenOf = append(childrenOf, nil)
		end := idx
		for i, c := range e.Children {
			childIdx := len(flat)
			childrenOf[idx] = append(childrenOf[idx], childIdx)
			end = walk(c, idx, depth+1, i)
		}
		flat[idx].subtreeEnd = end
		return end
	}
	walk(root, -1, 0, 0)
	return flat, childrenOf
}

// matchAtIndex evaluates m against the node at flat[idx]. It is the
// canonical entry point for tree-aware predicates: every combinator that
// recurses (HasAncestor, HasDescendant, HasParent, HasSibling) ultimately
// calls back into matchAtIndex with the candidate's flat index.
//
// Carrying the index (rather than re-deriving it from the element's content)
// makes correctness independent of duplicate sibling content — two nodes
// with identical text/bounds/class are still distinct paths.
func matchAtIndex(idx int, flat []pathItem, childrenOf [][]int, m *Matcher) (bool, error) {
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
		if item.siblingIndex != *m.ParentIndex {
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
		ok, err := matchAtIndex(item.parent, flat, childrenOf, m.HasParent)
		if err != nil || !ok {
			return ok, err
		}
	}

	if m.HasAncestor != nil {
		matched := false
		for p := item.parent; p >= 0; p = flat[p].parent {
			ok, err := matchAtIndex(p, flat, childrenOf, m.HasAncestor)
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
		// Pre-order DFS means idx's descendants occupy the contiguous range
		// (idx, subtreeEnd] — no need to rescan the whole tree.
		for i := idx + 1; i <= item.subtreeEnd; i++ {
			ok, err := matchAtIndex(i, flat, childrenOf, m.HasDescendant)
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
		for _, i := range childrenOf[item.parent] {
			if i == idx {
				continue
			}
			ok, err := matchAtIndex(i, flat, childrenOf, m.HasSibling)
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
