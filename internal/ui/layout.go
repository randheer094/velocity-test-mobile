// Package ui covers screen capture, UI layout extraction, image diffing, and
// screen-recording lifecycle.
package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
	"github.com/randheer094/velocity-test-mobile/internal/androidcli"
)

// Bounds is the rectangle occupied by an Element on screen.
type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Element is a single node in the UI hierarchy. Field names align with
// UIAutomator XML attributes and Compose semantics where applicable.
type Element struct {
	Class         string    `json:"class,omitempty"`
	Text          string    `json:"text,omitempty"`
	Label         string    `json:"label,omitempty"` // content-desc
	Hint          string    `json:"hint,omitempty"`
	ResourceID    string    `json:"resourceId,omitempty"`
	Package       string    `json:"package,omitempty"`
	ErrorText     string    `json:"errorText,omitempty"`
	Focused       bool      `json:"focused,omitempty"`
	Focusable     bool      `json:"focusable,omitempty"`
	Checkable     bool      `json:"checkable,omitempty"`
	Checked       bool      `json:"checked,omitempty"`
	Clickable     bool      `json:"clickable,omitempty"`
	LongClickable bool      `json:"longClickable,omitempty"`
	Scrollable    bool      `json:"scrollable,omitempty"`
	Selected      bool      `json:"selected,omitempty"`
	Enabled       bool      `json:"enabled,omitempty"`
	VisibleToUser bool      `json:"visibleToUser,omitempty"`
	Bounds        Bounds    `json:"bounds"`
	Children      []Element `json:"children,omitempty"`
}

// Predicate is a matcher used by wait_for_element.
type Predicate struct {
	Text        string `json:"text,omitempty"`
	ContentDesc string `json:"contentDesc,omitempty"`
	ResourceID  string `json:"resourceId,omitempty"`
	Class       string `json:"class,omitempty"`
}

// LayoutClient extracts UI hierarchies from a connected device.
type LayoutClient struct {
	Adb        *adb.Client
	AndroidCLI *androidcli.Client
}

// NewLayoutClient builds a LayoutClient.
func NewLayoutClient(a *adb.Client, c *androidcli.Client) *LayoutClient {
	return &LayoutClient{Adb: a, AndroidCLI: c}
}

// Tree returns a hierarchical layout via `android layout --pretty` JSON. The
// `android` agent CLI is required — returns androidcli.ErrNotInstalled if
// it isn't on PATH.
func (l *LayoutClient) Tree(ctx context.Context, deviceID string) (Element, error) {
	if l.AndroidCLI == nil || !l.AndroidCLI.Available() {
		return Element{}, androidcli.ErrNotInstalled
	}
	return l.fromAndroidCLI(ctx, deviceID)
}

func (l *LayoutClient) fromAndroidCLI(ctx context.Context, deviceID string) (Element, error) {
	args := []string{"layout", "--pretty"}
	if deviceID != "" {
		args = append(args, "--device", deviceID)
	}
	res, err := l.AndroidCLI.Run(ctx, args...)
	if err != nil {
		return Element{}, err
	}
	return parseAndroidCLILayout(res.Stdout)
}

// genericNode mirrors the JSON shape produced by `android layout --pretty`.
// The exact schema isn't formally documented; we accept several aliases.
type genericNode struct {
	Class         string          `json:"class"`
	Text          string          `json:"text"`
	ContentDesc   string          `json:"contentDesc"`
	Description   string          `json:"description"`
	Hint          string          `json:"hint"`
	ResourceID    string          `json:"resourceId"`
	Package       string          `json:"package"`
	ErrorText     string          `json:"errorText"`
	Focused       bool            `json:"focused"`
	Focusable     bool            `json:"focusable"`
	Checkable     bool            `json:"checkable"`
	Checked       bool            `json:"checked"`
	Clickable     bool            `json:"clickable"`
	LongClickable bool            `json:"longClickable"`
	Scrollable    bool            `json:"scrollable"`
	Selected      bool            `json:"selected"`
	Enabled       *bool           `json:"enabled"`
	VisibleToUser *bool           `json:"visibleToUser"`
	Bounds        json.RawMessage `json:"bounds"`
	Children      []genericNode   `json:"children"`
}

func parseAndroidCLILayout(data []byte) (Element, error) {
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '[' {
		return parseFlatArrayLayout(data)
	}
	var n genericNode
	if err := json.Unmarshal(data, &n); err != nil {
		return Element{}, fmt.Errorf("parsing android layout JSON: %w", err)
	}
	return convertNode(n), nil
}

// flatNode mirrors the flat, non-hierarchical JSON array produced by some
// `android` CLI versions (observed as of v1.0.15985488): a single flat list
// of elements with no `children`/`class`/object `bounds`. `bounds` (a rect
// string) is only present on a minority of elements, mostly scrollable
// containers; most leaf elements only carry `center`.
type flatNode struct {
	Text         string   `json:"text"`
	ContentDesc  string   `json:"content-desc"`
	ResourceID   string   `json:"resource-id"`
	Bounds       string   `json:"bounds"`
	Center       string   `json:"center"`
	Interactions []string `json:"interactions"`
	OffScreen    bool     `json:"off-screen"`
}

// parseFlatArrayLayout builds a synthetic root over the flat list. This
// schema has no real hierarchy, so the root exists only as a container:
// matchers relying on HasParent/HasAncestor/HasDescendant/HasSibling
// against this tree won't find meaningful results, since every element is
// a direct child of the synthetic root.
//
// The root's own Bounds is set to the bounding box of its children's real
// (CLI-reported) rects, falling back to the union of synthesized rects if
// no element carried a real one. Without this, matcher.CompletelyDisplayed
// and DisplayingAtLeastPercent (which measure a candidate against its
// root's viewport) would always fail against a zero-area root.
func parseFlatArrayLayout(data []byte) (Element, error) {
	var nodes []flatNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return Element{}, fmt.Errorf("parsing android layout JSON: %w", err)
	}
	root := Element{}
	var realUnion, anyUnion Bounds
	haveReal, haveAny := false, false
	for _, n := range nodes {
		e, hadRealBounds := convertFlatNode(n)
		root.Children = append(root.Children, e)
		if e.Bounds.Width <= 0 || e.Bounds.Height <= 0 {
			continue
		}
		anyUnion = unionBounds(anyUnion, e.Bounds, haveAny)
		haveAny = true
		if hadRealBounds {
			realUnion = unionBounds(realUnion, e.Bounds, haveReal)
			haveReal = true
		}
	}
	switch {
	case haveReal:
		root.Bounds = realUnion
	case haveAny:
		root.Bounds = anyUnion
	}
	return root, nil
}

func unionBounds(acc, b Bounds, haveAcc bool) Bounds {
	if !haveAcc {
		return b
	}
	x1 := min(acc.X, b.X)
	y1 := min(acc.Y, b.Y)
	x2 := max(acc.X+acc.Width, b.X+b.Width)
	y2 := max(acc.Y+acc.Height, b.Y+b.Height)
	return Bounds{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// convertFlatNode returns the parsed Element and whether its Bounds came
// from the CLI's own `bounds` string (true) or was synthesized from
// `center` (false, or zero if neither was present).
func convertFlatNode(n flatNode) (Element, bool) {
	e := Element{
		Text:          n.Text,
		Label:         n.ContentDesc,
		ResourceID:    n.ResourceID,
		Enabled:       true,
		VisibleToUser: !n.OffScreen,
	}
	// "clickable"/"long-clickable"/"focusable"/"scrollable" were directly
	// observed on a live device (android CLI v1.0.15985488). The other
	// aliases ("tap", "long_press", "scroll", "checkable") are speculative
	// — kept defensively for other CLI versions/builds but unverified.
	for _, in := range n.Interactions {
		switch in {
		case "clickable", "tap":
			e.Clickable = true
		case "long-clickable", "long_press", "long-press":
			e.LongClickable = true
		case "focusable":
			e.Focusable = true
		case "scrollable", "scroll":
			e.Scrollable = true
		case "checkable":
			e.Checkable = true
		}
	}
	if b, ok := parseBoundsString(n.Bounds); ok {
		e.Bounds = b
		return e, true
	}
	if cx, cy, ok := parseCenterString(n.Center); ok {
		// No real rectangle available for this element, only a tap point.
		// Synthesize a tiny non-zero rect around it so CenterOf() still
		// resolves correctly and isInteresting()/IsDisplayed() (which key
		// off Width/Height > 0) don't drop it.
		const half = 2
		e.Bounds = Bounds{X: cx - half, Y: cy - half, Width: 2 * half, Height: 2 * half}
	}
	return e, false
}

func convertNode(n genericNode) Element {
	enabled := true
	if n.Enabled != nil {
		enabled = *n.Enabled
	}
	visible := true
	if n.VisibleToUser != nil {
		visible = *n.VisibleToUser
	}
	label := n.ContentDesc
	if label == "" {
		label = n.Description
	}
	e := Element{
		Class:         n.Class,
		Text:          n.Text,
		Label:         label,
		Hint:          n.Hint,
		ResourceID:    n.ResourceID,
		Package:       n.Package,
		ErrorText:     n.ErrorText,
		Focused:       n.Focused,
		Focusable:     n.Focusable,
		Checkable:     n.Checkable,
		Checked:       n.Checked,
		Clickable:     n.Clickable,
		LongClickable: n.LongClickable,
		Scrollable:    n.Scrollable,
		Selected:      n.Selected,
		Enabled:       enabled,
		VisibleToUser: visible,
		Bounds:        parseBoundsJSON(n.Bounds),
	}
	for _, c := range n.Children {
		e.Children = append(e.Children, convertNode(c))
	}
	return e
}

func parseBoundsJSON(raw json.RawMessage) Bounds {
	if len(raw) == 0 {
		return Bounds{}
	}
	// Try array form: [x1,y1,x2,y2]
	var arr []int
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) == 4 {
		return Bounds{X: arr[0], Y: arr[1], Width: arr[2] - arr[0], Height: arr[3] - arr[1]}
	}
	// Try object {left,top,right,bottom} or {x,y,width,height}
	var obj struct {
		Left, Top, Right, Bottom int
		X, Y, Width, Height      int
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.Width != 0 || obj.Height != 0 {
			return Bounds{X: obj.X, Y: obj.Y, Width: obj.Width, Height: obj.Height}
		}
		return Bounds{X: obj.Left, Y: obj.Top, Width: obj.Right - obj.Left, Height: obj.Bottom - obj.Top}
	}
	// Try string "[x1,y1][x2,y2]"
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if b, ok := parseBoundsString(s); ok {
			return b
		}
	}
	return Bounds{}
}

var boundsRE = regexp.MustCompile(`\[(-?\d+),(-?\d+)\]\[(-?\d+),(-?\d+)\]`)

func parseBoundsString(s string) (Bounds, bool) {
	m := boundsRE.FindStringSubmatch(s)
	if m == nil {
		return Bounds{}, false
	}
	atoi := func(x string) int { n, _ := strconv.Atoi(x); return n }
	x1, y1, x2, y2 := atoi(m[1]), atoi(m[2]), atoi(m[3]), atoi(m[4])
	return Bounds{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}, true
}

var centerRE = regexp.MustCompile(`\[(-?\d+),(-?\d+)\]`)

func parseCenterString(s string) (x, y int, ok bool) {
	m := centerRE.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	atoi := func(v string) int { n, _ := strconv.Atoi(v); return n }
	return atoi(m[1]), atoi(m[2]), true
}

// Flatten returns interactive/significant elements ordered by depth-first
// traversal, dropping zero-area and purely structural nodes.
func Flatten(root Element) []Element {
	var out []Element
	var walk func(Element)
	walk = func(e Element) {
		if isInteresting(e) {
			leaf := e
			leaf.Children = nil
			out = append(out, leaf)
		}
		for _, c := range e.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

func isInteresting(e Element) bool {
	if e.Bounds.Width <= 0 || e.Bounds.Height <= 0 {
		return false
	}
	return e.Text != "" ||
		e.Label != "" ||
		e.Hint != "" ||
		e.ResourceID != "" ||
		e.Checkable ||
		e.Clickable
}

// Match reports whether any descendant of root satisfies p.
func Match(root Element, p Predicate) (Element, bool) {
	var walk func(Element) (Element, bool)
	walk = func(e Element) (Element, bool) {
		if matchOne(e, p) {
			return e, true
		}
		for _, c := range e.Children {
			if got, ok := walk(c); ok {
				return got, true
			}
		}
		return Element{}, false
	}
	return walk(root)
}

func matchOne(e Element, p Predicate) bool {
	if p.Text == "" && p.ContentDesc == "" && p.ResourceID == "" && p.Class == "" {
		return false
	}
	return matches(e.Text, p.Text) &&
		matches(e.Label, p.ContentDesc) &&
		matches(e.ResourceID, p.ResourceID) &&
		matches(e.Class, p.Class)
}

func matches(value, pattern string) bool {
	if pattern == "" {
		return true
	}
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) >= 2 {
		re, err := regexp.Compile(pattern[1 : len(pattern)-1])
		if err != nil {
			return false
		}
		return re.MatchString(value)
	}
	return strings.Contains(value, pattern)
}

// WriteTempScreenshotForResolve writes raw PNG bytes to a temp file and
// returns the path. This is used by `screen_resolve` which needs a host
// path to feed the android CLI.
func WriteTempScreenshotForResolve(prefix string, data []byte) (string, error) {
	f, err := os.CreateTemp("", prefix+"-*.png")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}
