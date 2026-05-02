package testing

import (
	"context"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/matcher"
	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// AssertResult is the structured outcome of every assertion verb.
type AssertResult struct {
	OK      bool        `json:"ok"`
	Reason  string      `json:"reason,omitempty"`
	Element *ui.Element `json:"element,omitempty"`
	Matched int         `json:"matched"`
}

// assertWith runs check on the matched element. When the matcher returns
// ErrNotFound, ok is automatically false with a helpful reason.
func (o *Orchestrator) assertWith(ctx context.Context, deviceID string, m *matcher.Matcher, check func(ui.Element) (bool, string)) (AssertResult, error) {
	elem, all, err := o.fetchAndFind(ctx, deviceID, m)
	if err != nil {
		if errIsNotFound(err) {
			return AssertResult{OK: false, Reason: "no element matched"}, nil
		}
		return AssertResult{}, err
	}
	ok, reason := check(elem)
	return AssertResult{OK: ok, Element: &elem, Matched: len(all), Reason: reason}, nil
}

func errIsNotFound(err error) bool {
	return err == matcher.ErrNotFound || (err != nil && err.Error() != "" &&
		(err == matcher.ErrNotFound || strings.HasPrefix(err.Error(), matcher.ErrNotFound.Error())))
}

// AssertVisible — Espresso isDisplayed / Compose assertIsDisplayed.
func (o *Orchestrator) AssertVisible(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if matcher.IsDisplayed(e) {
			return true, ""
		}
		return false, "element exists but is not displayed (zero bounds or visibleToUser=false)"
	})
}

// AssertNotVisible — Compose assertIsNotDisplayed.
func (o *Orchestrator) AssertNotVisible(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, m)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	for _, e := range all {
		if matcher.IsDisplayed(e) {
			return AssertResult{OK: false, Reason: "element is currently displayed", Matched: len(all), Element: &e}, nil
		}
	}
	return AssertResult{OK: true, Matched: len(all)}, nil
}

// AssertExists — Compose assertExists.
func (o *Orchestrator) AssertExists(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, m)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err == matcher.ErrNotFound || len(all) == 0 {
		return AssertResult{OK: false, Reason: "no element matched"}, nil
	}
	first := all[0]
	return AssertResult{OK: true, Element: &first, Matched: len(all)}, nil
}

// AssertDoesNotExist — Compose assertDoesNotExist / Espresso doesNotExist.
func (o *Orchestrator) AssertDoesNotExist(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, m)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == 0 {
		return AssertResult{OK: true}, nil
	}
	first := all[0]
	return AssertResult{OK: false, Reason: fmt.Sprintf("%d matching element(s) exist", len(all)), Element: &first, Matched: len(all)}, nil
}

// AssertClickable — isClickable / hasClickAction.
func (o *Orchestrator) AssertClickable(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Clickable {
			return true, ""
		}
		return false, "element is not clickable"
	})
}

// AssertEnabled — isEnabled / assertIsEnabled.
func (o *Orchestrator) AssertEnabled(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Enabled {
			return true, ""
		}
		return false, "element is not enabled"
	})
}

// AssertDisabled — isNotEnabled.
func (o *Orchestrator) AssertDisabled(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if !e.Enabled {
			return true, ""
		}
		return false, "element is enabled"
	})
}

// AssertFocused — isFocused / hasFocus.
func (o *Orchestrator) AssertFocused(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Focused {
			return true, ""
		}
		return false, "element is not focused"
	})
}

// AssertSelected — isSelected / assertIsSelected.
func (o *Orchestrator) AssertSelected(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Selected {
			return true, ""
		}
		return false, "element is not selected"
	})
}

// AssertChecked — isChecked.
func (o *Orchestrator) AssertChecked(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Checked {
			return true, ""
		}
		return false, "element is not checked"
	})
}

// AssertUnchecked — isNotChecked.
func (o *Orchestrator) AssertUnchecked(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if !e.Checked {
			return true, ""
		}
		return false, "element is checked"
	})
}

// AssertTextEquals — assertTextEquals.
func (o *Orchestrator) AssertTextEquals(ctx context.Context, deviceID string, m *matcher.Matcher, expected string) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Text == expected {
			return true, ""
		}
		return false, fmt.Sprintf("text=%q, expected %q", e.Text, expected)
	})
}

// AssertTextContains — assertTextContains.
func (o *Orchestrator) AssertTextContains(ctx context.Context, deviceID string, m *matcher.Matcher, substring string) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if strings.Contains(e.Text, substring) {
			return true, ""
		}
		return false, fmt.Sprintf("text=%q does not contain %q", e.Text, substring)
	})
}

// AssertContentDescriptionEquals.
func (o *Orchestrator) AssertContentDescriptionEquals(ctx context.Context, deviceID string, m *matcher.Matcher, expected string) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Label == expected {
			return true, ""
		}
		return false, fmt.Sprintf("contentDescription=%q, expected %q", e.Label, expected)
	})
}

// AssertCountEquals — assertCountEquals (Compose).
func (o *Orchestrator) AssertCountEquals(ctx context.Context, deviceID string, m *matcher.Matcher, expected int) (AssertResult, error) {
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, m)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == expected {
		return AssertResult{OK: true, Matched: len(all)}, nil
	}
	return AssertResult{OK: false, Matched: len(all), Reason: fmt.Sprintf("matched %d, expected %d", len(all), expected)}, nil
}

// AssertWidthDp — Compose assertWidthIsEqualTo(dp). The pixel width is
// converted to dp via the supplied density (1.0 if zero); a small tolerance
// is allowed to absorb sub-pixel rounding.
func (o *Orchestrator) AssertWidthDp(ctx context.Context, deviceID string, m *matcher.Matcher, expectedDp int, density float64) (AssertResult, error) {
	if density <= 0 {
		density = 1.0
	}
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		gotDp := int(float64(e.Bounds.Width)/density + 0.5)
		if absInt(gotDp-expectedDp) <= 1 {
			return true, ""
		}
		return false, fmt.Sprintf("widthDp=%d, expected %d (density=%v)", gotDp, expectedDp, density)
	})
}

// AssertHeightDp — Compose assertHeightIsEqualTo(dp).
func (o *Orchestrator) AssertHeightDp(ctx context.Context, deviceID string, m *matcher.Matcher, expectedDp int, density float64) (AssertResult, error) {
	if density <= 0 {
		density = 1.0
	}
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		gotDp := int(float64(e.Bounds.Height)/density + 0.5)
		if absInt(gotDp-expectedDp) <= 1 {
			return true, ""
		}
		return false, fmt.Sprintf("heightDp=%d, expected %d (density=%v)", gotDp, expectedDp, density)
	})
}

// AssertWidthAtLeastDp — Compose assertWidthIsAtLeast.
func (o *Orchestrator) AssertWidthAtLeastDp(ctx context.Context, deviceID string, m *matcher.Matcher, minDp int, density float64) (AssertResult, error) {
	if density <= 0 {
		density = 1.0
	}
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		gotDp := int(float64(e.Bounds.Width) / density)
		if gotDp >= minDp {
			return true, ""
		}
		return false, fmt.Sprintf("widthDp=%d, expected >= %d (density=%v)", gotDp, minDp, density)
	})
}

// AssertHeightAtLeastDp — Compose assertHeightIsAtLeast.
func (o *Orchestrator) AssertHeightAtLeastDp(ctx context.Context, deviceID string, m *matcher.Matcher, minDp int, density float64) (AssertResult, error) {
	if density <= 0 {
		density = 1.0
	}
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		gotDp := int(float64(e.Bounds.Height) / density)
		if gotDp >= minDp {
			return true, ""
		}
		return false, fmt.Sprintf("heightDp=%d, expected >= %d (density=%v)", gotDp, minDp, density)
	})
}

// AssertPositionInRoot — Compose assertPositionInRootIsEqualTo. Pixel-based
// since dp would require density; LLM agent can compare to a captured
// expected value or use density for conversion.
func (o *Orchestrator) AssertPositionInRoot(ctx context.Context, deviceID string, m *matcher.Matcher, expectedX, expectedY int, tolerancePx int) (AssertResult, error) {
	if tolerancePx < 0 {
		tolerancePx = 0
	}
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		dx := absInt(e.Bounds.X - expectedX)
		dy := absInt(e.Bounds.Y - expectedY)
		if dx <= tolerancePx && dy <= tolerancePx {
			return true, ""
		}
		return false, fmt.Sprintf("position=(%d,%d), expected (%d,%d) ±%d", e.Bounds.X, e.Bounds.Y, expectedX, expectedY, tolerancePx)
	})
}

// AssertAny — Compose assertAny: at least one of the matched collection
// also satisfies sub-matcher.
func (o *Orchestrator) AssertAny(ctx context.Context, deviceID string, m *matcher.Matcher, sub *matcher.Matcher) (AssertResult, error) {
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, m)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	for _, e := range all {
		ok, perr := matcher.Match(e, sub)
		if perr != nil {
			return AssertResult{}, perr
		}
		if ok {
			ec := e
			return AssertResult{OK: true, Element: &ec, Matched: len(all)}, nil
		}
	}
	return AssertResult{OK: false, Matched: len(all), Reason: "no matched element satisfied the sub-matcher"}, nil
}

// AssertAll — Compose assertAll: every matched element must also satisfy
// the sub-matcher.
func (o *Orchestrator) AssertAll(ctx context.Context, deviceID string, m *matcher.Matcher, sub *matcher.Matcher) (AssertResult, error) {
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, m)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == 0 {
		return AssertResult{OK: false, Reason: "no elements matched the outer selector"}, nil
	}
	for _, e := range all {
		ok, perr := matcher.Match(e, sub)
		if perr != nil {
			return AssertResult{}, perr
		}
		if !ok {
			ec := e
			return AssertResult{OK: false, Matched: len(all), Element: &ec, Reason: "at least one element failed the sub-matcher"}, nil
		}
	}
	return AssertResult{OK: true, Matched: len(all)}, nil
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// assertCheckable — Compose assertIsToggleable; checks the Checkable flag.
func (o *Orchestrator) AssertCheckable(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	return o.assertWith(ctx, deviceID, m, func(e ui.Element) (bool, string) {
		if e.Checkable {
			return true, ""
		}
		return false, "element is not toggleable (checkable=false)"
	})
}

// assertCompletelyDisplayed — Espresso isCompletelyDisplayed via the
// matcher-level CompletelyDisplayed flag.
func (o *Orchestrator) AssertCompletelyDisplayed(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	combined := *m
	yes := true
	combined.CompletelyDisplayed = &yes
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, &combined)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == 0 {
		return AssertResult{OK: false, Reason: "no element matched and was completely displayed"}, nil
	}
	first := all[0]
	return AssertResult{OK: true, Element: &first, Matched: len(all)}, nil
}

// assertDisplayingAtLeast — Espresso isDisplayingAtLeast(percent).
func (o *Orchestrator) AssertDisplayingAtLeast(ctx context.Context, deviceID string, m *matcher.Matcher, percent int) (AssertResult, error) {
	if percent < 1 {
		percent = 1
	}
	if percent > 100 {
		percent = 100
	}
	combined := *m
	combined.DisplayingAtLeastPercent = percent
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, &combined)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == 0 {
		return AssertResult{OK: false, Reason: fmt.Sprintf("no element matched while displaying at least %d%%", percent)}, nil
	}
	first := all[0]
	return AssertResult{OK: true, Element: &first, Matched: len(all)}, nil
}

// assertIsRoot — Espresso isRoot.
func (o *Orchestrator) AssertIsRoot(ctx context.Context, deviceID string, m *matcher.Matcher) (AssertResult, error) {
	combined := *m
	yes := true
	combined.IsRoot = &yes
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, &combined)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == 0 {
		return AssertResult{OK: false, Reason: "matched element is not the root"}, nil
	}
	first := all[0]
	return AssertResult{OK: true, Element: &first, Matched: len(all)}, nil
}

// assertChildCount — Espresso hasChildCount / hasMinimumChildCount.
func (o *Orchestrator) AssertChildCount(ctx context.Context, deviceID string, m *matcher.Matcher, count int, atLeast bool) (AssertResult, error) {
	combined := *m
	if atLeast {
		combined.MinChildCount = &count
	} else {
		combined.ChildCount = &count
	}
	root, err := o.Layout.Tree(ctx, deviceID)
	if err != nil {
		return AssertResult{}, err
	}
	all, err := matcher.FindAll(root, &combined)
	if err == matcher.ErrEmptyMatcher {
		return AssertResult{}, err
	}
	if err != nil && err != matcher.ErrNotFound {
		return AssertResult{}, err
	}
	if len(all) == 0 {
		return AssertResult{OK: false, Reason: fmt.Sprintf("no element matched with the required child count")}, nil
	}
	first := all[0]
	return AssertResult{OK: true, Element: &first, Matched: len(all)}, nil
}

// AssertHasDescendant — Espresso hasDescendant.
func (o *Orchestrator) AssertHasDescendant(ctx context.Context, deviceID string, m *matcher.Matcher, descendant *matcher.Matcher) (AssertResult, error) {
	combined := *m
	combined.HasDescendant = descendant
	return o.AssertExists(ctx, deviceID, &combined)
}
