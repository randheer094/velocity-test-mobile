// Package matcher implements Espresso-/Compose-style selectors over the
// Android accessibility tree.
//
// A Matcher is a single JSON-friendly value the LLM agent passes to every
// testing tool (assert_visible, click, scroll_to, ...). Match(node, m)
// reports whether a single node satisfies the predicate; Find / FindAll
// walk a parsed UI tree.
//
// The vocabulary maps onto Espresso ViewMatchers and Compose
// SemanticsMatchers:
//
//	withText / hasText                  → Matcher.Text / TextContains / TextRegex
//	withContentDescription / hasCD      → Matcher.ContentDescription[Contains]
//	withResourceName / hasTestTag       → Matcher.ResourceID / TestTag
//	withClassName                       → Matcher.ClassName
//	withHint                            → Matcher.Hint
//	hasErrorText                        → Matcher.ErrorText
//	isClickable / hasClickAction        → Matcher.Clickable
//	isEnabled                           → Matcher.Enabled
//	isChecked                           → Matcher.Checked
//	isFocused / hasFocus                → Matcher.Focused
//	isSelected                          → Matcher.Selected
//	isDisplayed                         → Matcher.Displayed (bounds + visibleToUser)
//	hasDescendant / hasAncestor /
//	  hasSibling / withParent           → Matcher.HasDescendant / HasAncestor / HasSibling / HasParent
//	allOf / anyOf / not                 → Matcher.AllOf / AnyOf / Not
//
// Multiple matches are resolved by Matcher.Instance (0-indexed; default 0).
package matcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// Matcher is a JSON-friendly selector for elements in the UI tree.
//
// Every field is optional; a zero-value Matcher matches nothing (Find returns
// an error, Match returns false) so handlers can detect "no criteria
// supplied" upfront.
type Matcher struct {
	// Identity / properties
	Text                       string `json:"text,omitempty" jsonschema:"exact text match"`
	TextContains               string `json:"textContains,omitempty" jsonschema:"substring of text"`
	TextRegex                  string `json:"textRegex,omitempty" jsonschema:"Go regex over text"`
	ContentDescription         string `json:"contentDescription,omitempty" jsonschema:"exact content-desc"`
	ContentDescriptionContains string `json:"contentDescriptionContains,omitempty"`
	ResourceID                 string `json:"resourceId,omitempty" jsonschema:"matches resource-id; in Compose with testTagsAsResourceId, this is the testTag"`
	TestTag                    string `json:"testTag,omitempty" jsonschema:"convenience for resource-id suffix or full match"`
	ClassName                  string `json:"className,omitempty" jsonschema:"matches the node's class (substring)"`
	Hint                       string `json:"hint,omitempty"`
	Package                    string `json:"package,omitempty"`
	ErrorText                  string `json:"errorText,omitempty"`

	// State filters; nil pointer = "don't care", explicit true/false applies
	Clickable     *bool `json:"clickable,omitempty"`
	LongClickable *bool `json:"longClickable,omitempty"`
	Enabled       *bool `json:"enabled,omitempty"`
	Checkable     *bool `json:"checkable,omitempty"`
	Checked       *bool `json:"checked,omitempty"`
	Focused       *bool `json:"focused,omitempty"`
	Focusable     *bool `json:"focusable,omitempty"`
	Selected      *bool `json:"selected,omitempty"`
	Scrollable    *bool `json:"scrollable,omitempty"`
	Displayed     *bool `json:"displayed,omitempty" jsonschema:"requires non-zero bounds AND visibleToUser=true"`

	// Hierarchy combinators
	HasDescendant *Matcher `json:"hasDescendant,omitempty"`
	HasAncestor   *Matcher `json:"hasAncestor,omitempty"`
	HasParent     *Matcher `json:"hasParent,omitempty"`
	HasSibling    *Matcher `json:"hasSibling,omitempty"`

	// Logical combinators
	AllOf []Matcher `json:"allOf,omitempty"`
	AnyOf []Matcher `json:"anyOf,omitempty"`
	Not   *Matcher  `json:"not,omitempty"`

	// Disambiguation when multiple nodes match
	Instance int `json:"instance,omitempty" jsonschema:"0-indexed; pick the Nth match (default 0)"`
}

// IsEmpty reports whether the matcher specifies no criteria. A handler
// receiving an empty matcher should reject the call so the LLM doesn't
// accidentally select random nodes.
func (m *Matcher) IsEmpty() bool {
	if m == nil {
		return true
	}
	return m.Text == "" && m.TextContains == "" && m.TextRegex == "" &&
		m.ContentDescription == "" && m.ContentDescriptionContains == "" &&
		m.ResourceID == "" && m.TestTag == "" && m.ClassName == "" &&
		m.Hint == "" && m.Package == "" && m.ErrorText == "" &&
		m.Clickable == nil && m.LongClickable == nil && m.Enabled == nil &&
		m.Checkable == nil && m.Checked == nil && m.Focused == nil &&
		m.Focusable == nil && m.Selected == nil && m.Scrollable == nil &&
		m.Displayed == nil &&
		m.HasDescendant == nil && m.HasAncestor == nil &&
		m.HasParent == nil && m.HasSibling == nil &&
		len(m.AllOf) == 0 && len(m.AnyOf) == 0 && m.Not == nil
}

// Match reports whether a single element satisfies the matcher's local
// predicates. It does NOT consult ancestors/descendants/siblings — those
// require the surrounding tree, so Find / FindAll handle them.
func Match(e ui.Element, m *Matcher) (bool, error) {
	if m == nil || m.IsEmpty() {
		return false, nil
	}

	if m.Text != "" && e.Text != m.Text {
		return false, nil
	}
	if m.TextContains != "" && !strings.Contains(e.Text, m.TextContains) {
		return false, nil
	}
	if m.TextRegex != "" {
		re, err := regexp.Compile(m.TextRegex)
		if err != nil {
			return false, fmt.Errorf("invalid textRegex %q: %w", m.TextRegex, err)
		}
		if !re.MatchString(e.Text) {
			return false, nil
		}
	}
	if m.ContentDescription != "" && e.Label != m.ContentDescription {
		return false, nil
	}
	if m.ContentDescriptionContains != "" && !strings.Contains(e.Label, m.ContentDescriptionContains) {
		return false, nil
	}
	if m.ResourceID != "" && !resourceIDMatches(e.ResourceID, m.ResourceID) {
		return false, nil
	}
	if m.TestTag != "" && !testTagMatches(e.ResourceID, m.TestTag) {
		return false, nil
	}
	if m.ClassName != "" && !strings.Contains(e.Class, m.ClassName) {
		return false, nil
	}
	if m.Hint != "" && e.Hint != m.Hint {
		return false, nil
	}
	if m.Package != "" && e.Package != m.Package {
		return false, nil
	}
	if m.ErrorText != "" && e.ErrorText != m.ErrorText {
		return false, nil
	}

	if m.Clickable != nil && e.Clickable != *m.Clickable {
		return false, nil
	}
	if m.LongClickable != nil && e.LongClickable != *m.LongClickable {
		return false, nil
	}
	if m.Enabled != nil && e.Enabled != *m.Enabled {
		return false, nil
	}
	if m.Checkable != nil && e.Checkable != *m.Checkable {
		return false, nil
	}
	if m.Checked != nil && e.Checked != *m.Checked {
		return false, nil
	}
	if m.Focused != nil && e.Focused != *m.Focused {
		return false, nil
	}
	if m.Focusable != nil && e.Focusable != *m.Focusable {
		return false, nil
	}
	if m.Selected != nil && e.Selected != *m.Selected {
		return false, nil
	}
	if m.Scrollable != nil && e.Scrollable != *m.Scrollable {
		return false, nil
	}
	if m.Displayed != nil {
		want := *m.Displayed
		got := IsDisplayed(e)
		if got != want {
			return false, nil
		}
	}

	if m.Not != nil {
		ok, err := Match(e, m.Not)
		if err != nil {
			return false, err
		}
		if ok {
			return false, nil
		}
	}
	for i := range m.AllOf {
		ok, err := Match(e, &m.AllOf[i])
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	if len(m.AnyOf) > 0 {
		any := false
		for i := range m.AnyOf {
			ok, err := Match(e, &m.AnyOf[i])
			if err != nil {
				return false, err
			}
			if ok {
				any = true
				break
			}
		}
		if !any {
			return false, nil
		}
	}
	return true, nil
}

// IsDisplayed mirrors Espresso's isDisplayed / Compose's assertIsDisplayed:
// the element must have non-zero bounds AND not be marked invisible.
func IsDisplayed(e ui.Element) bool {
	if e.Bounds.Width <= 0 || e.Bounds.Height <= 0 {
		return false
	}
	// VisibleToUser is true by default for converters that don't set it.
	return e.VisibleToUser
}

// resourceIDMatches accepts both fully qualified IDs ("com.app:id/foo") and
// suffixes ("foo"). If the matcher contains a colon we require an exact
// match; otherwise we require the resource ID to end in "/<value>".
func resourceIDMatches(actual, want string) bool {
	if strings.Contains(want, ":") {
		return actual == want
	}
	if actual == want {
		return true
	}
	return strings.HasSuffix(actual, "/"+want)
}

// testTagMatches treats the matcher value as a Compose testTag — accepts
// either an exact resource-id (when an app sets `testTagsAsResourceId=true`)
// or the suffix after `:id/`.
func testTagMatches(actual, tag string) bool {
	if actual == tag {
		return true
	}
	if strings.HasSuffix(actual, "/"+tag) {
		return true
	}
	return false
}
