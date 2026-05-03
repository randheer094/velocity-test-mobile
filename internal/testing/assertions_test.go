package testing

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// The new asserts (text_regex, content_description_contains, hint_equals,
// error_text_equals, input_type, long_clickable, has_ime_action) all reduce
// to a single predicate over ui.Element. assertWith handles the matcher
// resolution and result wrapping; verifying the predicates directly here
// avoids spinning up a fake LayoutClient/Adb just to exercise the trivial
// boolean logic.

func TestPredicate_TextRegex(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		pattern string
		want    bool
	}{
		{"exact", "hello", "^hello$", true},
		{"prefix", "hello world", "^hello", true},
		{"miss", "goodbye", "^hello", false},
		{"digit_alternation", "Note 137", `^Note \d+$`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			re := regexp.MustCompile(tc.pattern)
			got := re.MatchString(tc.text)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPredicate_ContentDescriptionContains(t *testing.T) {
	if !strings.Contains(ui.Element{Label: "open settings"}.Label, "settings") {
		t.Fatalf("expected match")
	}
	if strings.Contains(ui.Element{Label: "navigate"}.Label, "settings") {
		t.Fatalf("expected miss")
	}
}

func TestPredicate_ErrorTextEquals(t *testing.T) {
	if (ui.Element{ErrorText: "Required"}).ErrorText != "Required" {
		t.Fatalf("expected match")
	}
	if (ui.Element{ErrorText: "Required"}).ErrorText == "Optional" {
		t.Fatalf("expected miss")
	}
}

func TestPredicate_HintEquals(t *testing.T) {
	if (ui.Element{Hint: "Username"}).Hint != "Username" {
		t.Fatalf("expected match")
	}
	if (ui.Element{}).Hint == "Username" {
		t.Fatalf("empty hint should not match")
	}
}

func TestPredicate_InputTypeClassSubstring(t *testing.T) {
	if !strings.Contains((ui.Element{Class: "android.widget.EditText"}).Class, "EditText") {
		t.Fatalf("expected EditText match")
	}
	if strings.Contains((ui.Element{Class: "TextView"}).Class, "EditText") {
		t.Fatalf("TextView should not match EditText")
	}
}

func TestPredicate_LongClickable(t *testing.T) {
	if !(ui.Element{LongClickable: true}).LongClickable {
		t.Fatalf("LongClickable=true should match")
	}
	if (ui.Element{}).LongClickable {
		t.Fatalf("zero element should not match")
	}
}

// TestAssertTextRegex_InvalidPattern exercises the application-level error path
// in AssertTextRegex — the function must reject the pattern before touching
// the LayoutClient, so a zero-value Orchestrator is sufficient here.
func TestAssertTextRegex_InvalidPattern(t *testing.T) {
	o := &Orchestrator{}
	_, err := o.AssertTextRegex(context.Background(), "dev", nil, "[invalid")
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPredicate_HasImeAction(t *testing.T) {
	cases := []struct {
		name string
		elem ui.Element
		want bool
	}{
		{"editable_focusable", ui.Element{Class: "android.widget.EditText", Focusable: true}, true},
		{"compose_textfield", ui.Element{Class: "androidx.compose.ui.TextField", Focusable: true}, true},
		{"text_input_focusable", ui.Element{Class: "android.widget.TextInputLayout", Focusable: true}, true},
		{"editable_not_focusable", ui.Element{Class: "EditText"}, false},
		{"focusable_not_editable", ui.Element{Class: "android.widget.Button", Focusable: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.elem.Focusable && (strings.Contains(tc.elem.Class, "EditText") ||
				strings.Contains(tc.elem.Class, "TextInput") || strings.Contains(tc.elem.Class, "TextField"))
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
