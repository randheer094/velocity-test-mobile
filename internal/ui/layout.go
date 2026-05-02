// Package ui covers screen capture, UI layout extraction, image diffing, and
// screen-recording lifecycle.
package ui

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// Tree returns a hierarchical layout. Prefers `android layout --pretty` JSON
// when the agent CLI is installed; falls back to UIAutomator XML.
func (l *LayoutClient) Tree(ctx context.Context, deviceID string) (Element, error) {
	if l.AndroidCLI != nil && l.AndroidCLI.Available() {
		if tree, err := l.fromAndroidCLI(ctx, deviceID); err == nil {
			return tree, nil
		}
		// Soft-fall through on CLI error to give the UIAutomator path a chance.
	}
	return l.fromUIAutomator(ctx, deviceID)
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
	var n genericNode
	if err := json.Unmarshal(data, &n); err != nil {
		return Element{}, fmt.Errorf("parsing android layout JSON: %w", err)
	}
	return convertNode(n), nil
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

// uiautomator XML path -----------------------------------------------------

type xmlNode struct {
	XMLName       xml.Name  `xml:"node"`
	Class         string    `xml:"class,attr"`
	Text          string    `xml:"text,attr"`
	ContentDesc   string    `xml:"content-desc,attr"`
	Hint          string    `xml:"hint,attr"`
	ResourceID    string    `xml:"resource-id,attr"`
	Package       string    `xml:"package,attr"`
	Bounds        string    `xml:"bounds,attr"`
	Focused       string    `xml:"focused,attr"`
	Focusable     string    `xml:"focusable,attr"`
	Checkable     string    `xml:"checkable,attr"`
	Checked       string    `xml:"checked,attr"`
	Clickable     string    `xml:"clickable,attr"`
	LongClickable string    `xml:"long-clickable,attr"`
	Scrollable    string    `xml:"scrollable,attr"`
	Selected      string    `xml:"selected,attr"`
	Enabled       string    `xml:"enabled,attr"`
	VisibleToUser string    `xml:"visible-to-user,attr"`
	Children      []xmlNode `xml:"node"`
}

type xmlHierarchy struct {
	XMLName xml.Name  `xml:"hierarchy"`
	Nodes   []xmlNode `xml:"node"`
}

func (l *LayoutClient) fromUIAutomator(ctx context.Context, deviceID string) (Element, error) {
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		raw, err := l.Adb.ExecOut(ctx, deviceID, "uiautomator", "dump", "/dev/tty")
		if err == nil {
			cleaned := stripDumpTrailer(raw)
			if h, perr := parseUIAutomatorXML(cleaned); perr == nil && len(h.Nodes) > 0 {
				return convertXMLRoot(h), nil
			} else if perr != nil {
				lastErr = perr
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return Element{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	if lastErr == nil {
		lastErr = errors.New("uiautomator returned an empty hierarchy")
	}
	return Element{}, lastErr
}

func stripDumpTrailer(b []byte) []byte {
	s := string(b)
	if i := strings.LastIndex(s, "UI hierarchy dumped to:"); i > 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	return []byte(s)
}

func parseUIAutomatorXML(data []byte) (xmlHierarchy, error) {
	var h xmlHierarchy
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = false
	err := dec.Decode(&h)
	return h, err
}

func convertXMLRoot(h xmlHierarchy) Element {
	root := Element{Class: "hierarchy"}
	for _, n := range h.Nodes {
		root.Children = append(root.Children, convertXMLNode(n))
	}
	return root
}

func convertXMLNode(n xmlNode) Element {
	b, _ := parseBoundsString(n.Bounds)
	e := Element{
		Class:         n.Class,
		Text:          n.Text,
		Label:         n.ContentDesc,
		Hint:          n.Hint,
		ResourceID:    n.ResourceID,
		Package:       n.Package,
		Focused:       n.Focused == "true",
		Focusable:     n.Focusable == "true",
		Checkable:     n.Checkable == "true",
		Checked:       n.Checked == "true",
		Clickable:     n.Clickable == "true",
		LongClickable: n.LongClickable == "true",
		Scrollable:    n.Scrollable == "true",
		Selected:      n.Selected == "true",
		Enabled:       n.Enabled == "true" || n.Enabled == "",
		VisibleToUser: n.VisibleToUser == "true" || n.VisibleToUser == "",
		Bounds:        b,
	}
	for _, c := range n.Children {
		e.Children = append(e.Children, convertXMLNode(c))
	}
	return e
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
