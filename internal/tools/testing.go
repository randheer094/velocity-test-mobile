package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/matcher"
	"github.com/randheer094/velocity-mcp-mobile/internal/testing"
	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// matcherEnvelope holds the device + matcher + arbitrary extras parsed
// from a CallToolRequest. We unmarshal into the envelope first to pull
// out device/match, then unmarshal again into a per-handler extras struct.
type matcherEnvelope struct {
	Device string          `json:"device"`
	Match  matcher.Matcher `json:"match"`
}

// RegisterTesting wires the Espresso/Compose-style testing surface.
//
// All matcher-bearing tools are registered via Server.AddTool with hand-built
// JSON schemas (the SDK's Go-type schema generator can't follow the cyclic
// references inside Matcher). Schemas use `$ref` for recursion so an LLM
// client still gets full discovery of the selector vocabulary.
func RegisterTesting(s *mcp.Server, d *Deps) {
	o := d.Tester
	intents := d.Intents

	// addMatcherTool registers a tool that takes {device, match, ...extras}
	// where the matcher subschema is the recursive `$defs/matcher`.
	addMatcherTool := func(
		name, desc string,
		readOnly bool,
		extraProps map[string]any,
		extraRequired []string,
		fn func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error),
	) {
		schema := schemaWithMatcher(extraProps, extraRequired)
		annot := &mcp.ToolAnnotations{ReadOnlyHint: readOnly}
		s.AddTool(&mcp.Tool{
			Name:        name,
			Description: desc,
			Annotations: annot,
			InputSchema: schema,
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var env matcherEnvelope
			if err := json.Unmarshal(req.Params.Arguments, &env); err != nil {
				return errResultDirect(fmt.Errorf("invalid arguments: %w", err)), nil
			}
			dev, err := d.resolveDevice(ctx, env.Device)
			if err != nil {
				return errResultDirect(err), nil
			}
			result, err := fn(ctx, dev, &env.Match, req.Params.Arguments)
			if err != nil {
				return errResultDirect(err), nil
			}
			return jsonResultDirect(result), nil
		})
	}

	// addDeviceTool registers a tool that takes {device, ...extras} (no matcher).
	addDeviceTool := func(
		name, desc string,
		readOnly bool,
		extraProps map[string]any,
		extraRequired []string,
		fn func(ctx context.Context, dev string, raw json.RawMessage) (any, error),
	) {
		schema := schemaDeviceOnly(extraProps, extraRequired)
		annot := &mcp.ToolAnnotations{ReadOnlyHint: readOnly}
		s.AddTool(&mcp.Tool{
			Name:        name,
			Description: desc,
			Annotations: annot,
			InputSchema: schema,
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var env struct {
				Device string `json:"device"`
			}
			_ = json.Unmarshal(req.Params.Arguments, &env)
			dev, err := d.resolveDevice(ctx, env.Device)
			if err != nil {
				return errResultDirect(err), nil
			}
			result, err := fn(ctx, dev, req.Params.Arguments)
			if err != nil {
				return errResultDirect(err), nil
			}
			return jsonResultDirect(result), nil
		})
	}

	// ── Finders ─────────────────────────────────────────────────────────

	addMatcherTool("find_node", "Find a single element matching the selector. Returns the matched node or 'not found'.", true, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			root, err := o.SnapshotTree(ctx, dev)
			if err != nil {
				return nil, err
			}
			elem, err := matcher.Find(root, m)
			if err != nil {
				return map[string]any{"found": false, "error": err.Error()}, nil
			}
			return map[string]any{"found": true, "element": elem}, nil
		})

	addMatcherTool("find_all_nodes", "Find every element matching the selector.", true, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			root, err := o.SnapshotTree(ctx, dev)
			if err != nil {
				return nil, err
			}
			all, err := matcher.FindAll(root, m)
			if err != nil {
				return nil, err
			}
			return map[string]any{"count": len(all), "elements": all}, nil
		})

	addMatcherTool("count_nodes", "Count how many elements match the selector.", true, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			root, err := o.SnapshotTree(ctx, dev)
			if err != nil {
				return nil, err
			}
			c, err := matcher.Count(root, m)
			if err != nil {
				return nil, err
			}
			return map[string]int{"count": c}, nil
		})

	// print_tree: optional matcher; if absent, print the whole tree.
	s.AddTool(&mcp.Tool{
		Name:        "print_tree",
		Description: "Pretty-print the on-screen UI hierarchy (Compose printToLog / Espresso debug helper). If `match` is given, prints the matched subtree; otherwise the full tree.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		InputSchema: map[string]any{
			"type": "object",
			"$defs": map[string]any{
				"matcher": matcherSchemaDef,
			},
			"properties": map[string]any{
				"device":   deviceProp,
				"match":    map[string]any{"$ref": "#/$defs/matcher"},
				"maxDepth": map[string]any{"type": "integer", "minimum": 0, "description": "0 = unlimited"},
			},
			"additionalProperties": false,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Device   string           `json:"device"`
			Match    *matcher.Matcher `json:"match,omitempty"`
			MaxDepth int              `json:"maxDepth,omitempty"`
		}
		_ = json.Unmarshal(req.Params.Arguments, &args)
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResultDirect(err), nil
		}
		root, err := o.SnapshotTree(ctx, dev)
		if err != nil {
			return errResultDirect(err), nil
		}
		var subject ui.Element
		if args.Match != nil && !args.Match.IsEmpty() {
			subject, err = matcher.Find(root, args.Match)
			if err != nil {
				return errResultDirect(err), nil
			}
		} else {
			subject = root
		}
		return textResultDirect(printTree(subject, 0, args.MaxDepth)), nil
	})

	// ── Assertions ──────────────────────────────────────────────────────

	registerSimpleAssert := func(name, desc string, fn func(context.Context, string, *matcher.Matcher) (testing.AssertResult, error)) {
		addMatcherTool(name, desc, true, nil, nil,
			func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
				return fn(ctx, dev, m)
			})
	}

	registerSimpleAssert("assert_visible", "Assert the matched element is displayed (Espresso isDisplayed / Compose assertIsDisplayed).", o.AssertVisible)
	registerSimpleAssert("assert_not_visible", "Assert no matching element is currently displayed.", o.AssertNotVisible)
	registerSimpleAssert("assert_exists", "Assert at least one element matches (Compose assertExists).", o.AssertExists)
	registerSimpleAssert("assert_does_not_exist", "Assert no element matches (Compose assertDoesNotExist / Espresso doesNotExist).", o.AssertDoesNotExist)
	registerSimpleAssert("assert_clickable", "Assert the matched element is clickable.", o.AssertClickable)
	registerSimpleAssert("assert_enabled", "Assert the matched element is enabled.", o.AssertEnabled)
	registerSimpleAssert("assert_disabled", "Assert the matched element is disabled.", o.AssertDisabled)
	registerSimpleAssert("assert_focused", "Assert the matched element has focus.", o.AssertFocused)
	registerSimpleAssert("assert_selected", "Assert the matched element is selected.", o.AssertSelected)
	registerSimpleAssert("assert_checked", "Assert the matched element is checked.", o.AssertChecked)
	registerSimpleAssert("assert_unchecked", "Assert the matched element is not checked.", o.AssertUnchecked)

	addMatcherTool("assert_text_equals",
		"Assert the matched element's text equals the expected string (Compose assertTextEquals).",
		true,
		map[string]any{"expected": map[string]any{"type": "string"}},
		[]string{"expected"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Expected string `json:"expected"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.AssertTextEquals(ctx, dev, m, x.Expected)
		})

	addMatcherTool("assert_text_contains",
		"Assert the matched element's text contains the substring (Compose assertTextContains).",
		true,
		map[string]any{"substring": map[string]any{"type": "string"}},
		[]string{"substring"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Substring string `json:"substring"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.AssertTextContains(ctx, dev, m, x.Substring)
		})

	addMatcherTool("assert_content_description_equals",
		"Assert the matched element's content description equals expected (Compose assertContentDescriptionEquals).",
		true,
		map[string]any{"expected": map[string]any{"type": "string"}},
		[]string{"expected"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Expected string `json:"expected"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.AssertContentDescriptionEquals(ctx, dev, m, x.Expected)
		})

	addMatcherTool("assert_count_equals",
		"Assert the number of elements matching equals expected (Compose assertCountEquals).",
		true,
		map[string]any{"expected": map[string]any{"type": "integer", "minimum": 0}},
		[]string{"expected"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Expected int `json:"expected"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.AssertCountEquals(ctx, dev, m, x.Expected)
		})

	// assert_has_descendant takes two matchers; route through a custom schema.
	s.AddTool(&mcp.Tool{
		Name:        "assert_has_descendant",
		Description: "Assert the matched element has a descendant satisfying the descendant selector (Espresso hasDescendant).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		InputSchema: map[string]any{
			"type": "object",
			"$defs": map[string]any{
				"matcher": matcherSchemaDef,
			},
			"properties": map[string]any{
				"device":     deviceProp,
				"match":      map[string]any{"$ref": "#/$defs/matcher"},
				"descendant": map[string]any{"$ref": "#/$defs/matcher"},
			},
			"required":             []string{"match", "descendant"},
			"additionalProperties": false,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Device     string          `json:"device"`
			Match      matcher.Matcher `json:"match"`
			Descendant matcher.Matcher `json:"descendant"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return errResultDirect(err), nil
		}
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResultDirect(err), nil
		}
		res, err := o.AssertHasDescendant(ctx, dev, &args.Match, &args.Descendant)
		if err != nil {
			return errResultDirect(err), nil
		}
		return jsonResultDirect(res), nil
	})

	// ── Actions ─────────────────────────────────────────────────────────

	registerSimpleAction := func(name, desc string, fn func(context.Context, string, *matcher.Matcher) (testing.ActionResult, error)) {
		addMatcherTool(name, desc, false, nil, nil,
			func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
				res, _ := fn(ctx, dev, m)
				return res, nil
			})
	}
	registerSimpleAction("click", "Click the matched element (Espresso click() / Compose performClick()).", o.Click)
	registerSimpleAction("double_click", "Double-click the matched element.", o.DoubleClick)

	addMatcherTool("long_click", "Long-click the matched element.", false,
		map[string]any{"durationMs": map[string]any{"type": "integer", "minimum": 1, "maximum": 10000, "description": "default 800"}},
		nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				DurationMs int `json:"durationMs"`
			}
			_ = json.Unmarshal(raw, &x)
			res, _ := o.LongClick(ctx, dev, m, x.DurationMs)
			return res, nil
		})

	textTypeProps := map[string]any{
		"text":   map[string]any{"type": "string"},
		"submit": map[string]any{"type": "boolean", "description": "press ENTER after typing"},
	}
	addMatcherTool("type_text",
		"Click the matched element to focus it, then type text (Espresso typeText / Compose performTextInput).",
		false, textTypeProps, []string{"text"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Text   string `json:"text"`
				Submit bool   `json:"submit"`
			}
			_ = json.Unmarshal(raw, &x)
			res, _ := o.TypeText(ctx, dev, m, x.Text, x.Submit)
			return res, nil
		})
	addMatcherTool("replace_text",
		"Clear the matched field's existing text, then type new text (Espresso replaceText / Compose performTextReplacement).",
		false, textTypeProps, []string{"text"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Text   string `json:"text"`
				Submit bool   `json:"submit"`
			}
			_ = json.Unmarshal(raw, &x)
			res, _ := o.ReplaceText(ctx, dev, m, x.Text, x.Submit)
			return res, nil
		})
	addMatcherTool("clear_text",
		"Clear the matched field's text (Espresso clearText / Compose performTextClearance).",
		false, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			res, _ := o.ClearText(ctx, dev, m)
			return res, nil
		})
	addMatcherTool("submit_text",
		"Press the IME action button on the matched (or focused) field — convenience for ENTER.",
		false, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			res, _ := o.Submit(ctx, dev, m)
			return res, nil
		})

	addMatcherTool("swipe_node",
		"Swipe within the matched element's bounds (Espresso swipeUp/Down/Left/Right scoped to a view).",
		false,
		map[string]any{
			"direction":  map[string]any{"type": "string", "enum": []string{"up", "down", "left", "right"}},
			"durationMs": map[string]any{"type": "integer", "minimum": 1, "maximum": 10000},
		},
		[]string{"direction"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Direction  string `json:"direction"`
				DurationMs int    `json:"durationMs"`
			}
			_ = json.Unmarshal(raw, &x)
			res, _ := o.SwipeNode(ctx, dev, m, x.Direction, x.DurationMs)
			return res, nil
		})

	addMatcherTool("scroll_to",
		"Scroll a scrollable ancestor until the matched element is visible (Espresso scrollTo / Compose performScrollToNode).",
		false,
		map[string]any{
			"maxAttempts": map[string]any{"type": "integer", "minimum": 1, "description": "total swipes to attempt (default 12)"},
			"direction":   map[string]any{"type": "string", "enum": []string{"auto", "up", "down", "left", "right"}, "description": "default auto"},
		},
		nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				MaxAttempts int    `json:"maxAttempts"`
				Direction   string `json:"direction"`
			}
			_ = json.Unmarshal(raw, &x)
			res, _ := o.ScrollTo(ctx, dev, m, testing.ScrollOptions{MaxAttempts: x.MaxAttempts, Direction: x.Direction})
			return res, nil
		})

	addMatcherTool("perform_ime_action",
		"Press ENTER on the matched (or currently focused) field (Espresso pressImeActionButton).",
		false, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			res, _ := o.PerformIMEAction(ctx, dev, m)
			return res, nil
		})

	addMatcherTool("assert_clickable_and_click",
		"Convenience: assert the matched element is clickable, then click it.",
		false, nil, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, _ json.RawMessage) (any, error) {
			check, err := o.AssertClickable(ctx, dev, m)
			if err != nil {
				return nil, err
			}
			if !check.OK {
				return map[string]any{"clicked": false, "assert": check}, nil
			}
			click, _ := o.Click(ctx, dev, m)
			return map[string]any{"clicked": click.OK, "assert": check, "click": click}, nil
		})

	// ── Synchronization ─────────────────────────────────────────────────

	waitProps := map[string]any{
		"timeoutMs":  map[string]any{"type": "integer", "minimum": 1, "description": "max wait (default depends on tool)"},
		"intervalMs": map[string]any{"type": "integer", "minimum": 50, "description": "poll interval (default 250)"},
	}
	addMatcherTool("wait_until_visible",
		"Poll until a matching element is displayed (Compose waitUntilExists).",
		true, waitProps, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				TimeoutMs  int `json:"timeoutMs"`
				IntervalMs int `json:"intervalMs"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.WaitUntilVisible(ctx, dev, m, x.TimeoutMs, x.IntervalMs)
		})
	addMatcherTool("wait_until_not_visible",
		"Poll until no matching element is displayed (Compose waitUntilDoesNotExist).",
		true, waitProps, nil,
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				TimeoutMs  int `json:"timeoutMs"`
				IntervalMs int `json:"intervalMs"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.WaitUntilNotVisible(ctx, dev, m, x.TimeoutMs, x.IntervalMs)
		})

	addMatcherTool("wait_until_text",
		"Poll until a node matching the selector contains the expected text.",
		true,
		map[string]any{
			"expected":   map[string]any{"type": "string"},
			"timeoutMs":  waitProps["timeoutMs"],
			"intervalMs": waitProps["intervalMs"],
		},
		[]string{"expected"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Expected   string `json:"expected"`
				TimeoutMs  int    `json:"timeoutMs"`
				IntervalMs int    `json:"intervalMs"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.WaitUntilText(ctx, dev, m, x.Expected, x.TimeoutMs, x.IntervalMs)
		})

	addMatcherTool("wait_until_count",
		"Poll until exactly `count` elements match the selector.",
		true,
		map[string]any{
			"count":      map[string]any{"type": "integer", "minimum": 0},
			"timeoutMs":  waitProps["timeoutMs"],
			"intervalMs": waitProps["intervalMs"],
		},
		[]string{"count"},
		func(ctx context.Context, dev string, m *matcher.Matcher, raw json.RawMessage) (any, error) {
			var x struct {
				Count      int `json:"count"`
				TimeoutMs  int `json:"timeoutMs"`
				IntervalMs int `json:"intervalMs"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.WaitUntilCount(ctx, dev, m, x.Count, x.TimeoutMs, x.IntervalMs)
		})

	addDeviceTool("wait_for_idle",
		"Approximate Espresso onIdle / Compose waitForIdle: poll the tree until it stops changing for idleWindowMs. Heuristic — there is no real IdlingResource hook from outside the app.",
		true,
		map[string]any{
			"timeoutMs":    map[string]any{"type": "integer", "minimum": 100, "description": "default 8000"},
			"idleWindowMs": map[string]any{"type": "integer", "minimum": 100, "description": "tree must be unchanged for this long; default 500"},
		},
		nil,
		func(ctx context.Context, dev string, raw json.RawMessage) (any, error) {
			var x struct {
				TimeoutMs    int `json:"timeoutMs"`
				IdleWindowMs int `json:"idleWindowMs"`
			}
			_ = json.Unmarshal(raw, &x)
			return o.WaitForIdle(ctx, dev, x.TimeoutMs, x.IdleWindowMs)
		})

	// ── Espresso top-level conveniences ─────────────────────────────────

	addDeviceTool("espresso_press_back", "Press the system Back button (Espresso pressBack).", false, nil, nil,
		func(ctx context.Context, dev string, _ json.RawMessage) (any, error) {
			if err := d.Input.PressButton(ctx, dev, "BACK"); err != nil {
				return nil, err
			}
			return map[string]string{"pressed": "BACK"}, nil
		})
	addDeviceTool("close_soft_keyboard", "Best-effort dismiss the soft keyboard (Espresso closeSoftKeyboard); presses BACK which collapses most IMEs.", false, nil, nil,
		func(ctx context.Context, dev string, _ json.RawMessage) (any, error) {
			if err := d.Input.PressButton(ctx, dev, "BACK"); err != nil {
				return nil, err
			}
			return map[string]string{"keyboard": "closed"}, nil
		})
	addDeviceTool("open_overflow_menu", "Open the action-bar overflow / options menu (Espresso openActionBarOverflowOrOptionsMenu).", false, nil, nil,
		func(ctx context.Context, dev string, _ json.RawMessage) (any, error) {
			if err := d.Input.PressButton(ctx, dev, "MENU"); err != nil {
				return nil, err
			}
			return map[string]string{"menu": "opened"}, nil
		})

	// ── Espresso-Intents (recording-only) ───────────────────────────────

	addDeviceTool("intent_monitor_start",
		"Start capturing dispatched intents via logcat scrape. Stubbing (Espresso intending()) is NOT supported externally.",
		false,
		map[string]any{"package": map[string]any{"type": "string", "description": "only capture intents whose pkg matches"}},
		nil,
		func(ctx context.Context, dev string, raw json.RawMessage) (any, error) {
			var x struct {
				Package string `json:"package"`
			}
			_ = json.Unmarshal(raw, &x)
			if err := intents.Start(ctx, dev, x.Package); err != nil {
				return nil, err
			}
			return map[string]string{"status": "started"}, nil
		})

	addDeviceTool("intent_list_captured",
		"Return every captured intent in the active monitoring window.",
		true, nil, nil,
		func(ctx context.Context, dev string, _ json.RawMessage) (any, error) {
			return intents.List(ctx, dev)
		})

	addDeviceTool("assert_intent_sent",
		"Assert at least one captured intent satisfies the matcher (Espresso intended() — read-only).",
		true,
		map[string]any{
			"action":       map[string]any{"type": "string"},
			"data":         map[string]any{"type": "string", "description": "exact match"},
			"dataContains": map[string]any{"type": "string"},
			"package":      map[string]any{"type": "string"},
			"category":     map[string]any{"type": "string"},
		},
		nil,
		func(ctx context.Context, dev string, raw json.RawMessage) (any, error) {
			var im testing.IntentMatcher
			_ = json.Unmarshal(raw, &im)
			return intents.AssertSent(ctx, dev, im)
		})
}

// printTree renders an indented summary of an Element subtree.
func printTree(e ui.Element, depth, maxDepth int) string {
	if maxDepth > 0 && depth > maxDepth {
		return ""
	}
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	line := fmt.Sprintf("%s- %s", indent, e.Class)
	if e.Text != "" {
		line += fmt.Sprintf(" text=%q", e.Text)
	}
	if e.Label != "" {
		line += fmt.Sprintf(" desc=%q", e.Label)
	}
	if e.ResourceID != "" {
		line += fmt.Sprintf(" id=%q", e.ResourceID)
	}
	if e.Bounds.Width > 0 || e.Bounds.Height > 0 {
		line += fmt.Sprintf(" bounds=[%d,%d %dx%d]",
			e.Bounds.X, e.Bounds.Y, e.Bounds.Width, e.Bounds.Height)
	}
	flags := ""
	if e.Clickable {
		flags += "C"
	}
	if e.Focused {
		flags += "F"
	}
	if e.Checked {
		flags += "✓"
	}
	if e.Scrollable {
		flags += "S"
	}
	if !e.Enabled {
		flags += "-"
	}
	if flags != "" {
		line += " [" + flags + "]"
	}
	out := line + "\n"
	for _, c := range e.Children {
		out += printTree(c, depth+1, maxDepth)
	}
	return out
}
