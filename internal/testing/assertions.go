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

// AssertHasDescendant — Espresso hasDescendant.
func (o *Orchestrator) AssertHasDescendant(ctx context.Context, deviceID string, m *matcher.Matcher, descendant *matcher.Matcher) (AssertResult, error) {
	combined := *m
	combined.HasDescendant = descendant
	return o.AssertExists(ctx, deviceID, &combined)
}
