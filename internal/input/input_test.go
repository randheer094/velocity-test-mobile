package input

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestIsASCII(t *testing.T) {
	cases := map[string]bool{
		"":         true,
		"hello":    true,
		"hi 123!?": true,
		"héllo":    false,
		"こんにちは":    false,
		"emoji 🚀":  false,
	}
	for in, want := range cases {
		if got := isASCII(in); got != want {
			t.Errorf("isASCII(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestClamp(t *testing.T) {
	if clamp(50, 0, 100) != 50 {
		t.Error("identity")
	}
	if clamp(-5, 0, 100) != 0 {
		t.Error("low clamp")
	}
	if clamp(500, 0, 100) != 100 {
		t.Error("high clamp")
	}
	if clamp(50, 0, 0) != 50 {
		t.Error("zero hi disables high clamp")
	}
}

func TestBuildTapAndPressButtonCmd(t *testing.T) {
	got := buildTapAndPressButtonCmd(100, 200, 120, 66)
	want := "input tap 100 200; sleep 0.120; input keyevent 66"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildTapAndPressButtonCmd_NoSettle(t *testing.T) {
	got := buildTapAndPressButtonCmd(100, 200, 0, 66)
	want := "input tap 100 200; input keyevent 66"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildTapAndTypeASCIICmd(t *testing.T) {
	cases := []struct {
		name           string
		x, y, settleMs int
		text           string
		submit         bool
		want           string
	}{
		{"tap+settle+text+submit", 10, 20, 150, "hello world", true,
			"input tap 10 20; sleep 0.150; input text 'hello%sworld'; input keyevent 66"},
		{"no settle", 10, 20, 0, "hi", false,
			"input tap 10 20; input text 'hi'"},
		{"empty text, submit only", 10, 20, 150, "", true,
			"input tap 10 20; sleep 0.150; input keyevent 66"},
		{"text with single quote is escaped", 10, 20, 0, "it's", false,
			`input tap 10 20; input text 'it'\''s'`},
		{"text with shell metacharacters is quoted, not injected", 10, 20, 0, "a; rm -rf /", false,
			"input tap 10 20; input text 'a;%srm%s-rf%s/'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildTapAndTypeASCIICmd(c.x, c.y, c.settleMs, c.text, c.submit)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestBuildTapAndSettleCmd(t *testing.T) {
	if got, want := buildTapAndSettleCmd(1, 2, 150), "input tap 1 2; sleep 0.150"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := buildTapAndSettleCmd(1, 2, 0), "input tap 1 2"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSettleSeconds(t *testing.T) {
	cases := map[int]string{150: "0.150", 120: "0.120", 1000: "1.000", 5: "0.005"}
	for ms, want := range cases {
		if got := settleSeconds(ms); got != want {
			t.Errorf("settleSeconds(%d) = %q, want %q", ms, got, want)
		}
	}
}

func TestBuildDoubleTapCmd(t *testing.T) {
	got := buildDoubleTapCmd(10, 20)
	want := "input tap 10 20; input tap 10 20"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildPasteUnicodeCmd(t *testing.T) {
	cmd := buildPasteUnicodeCmd("héllo")
	if !strings.Contains(cmd, "&& input keyevent 279") {
		t.Errorf("expected `&&`-chained paste keyevent, got %q", cmd)
	}
	if !strings.Contains(cmd, "cmd clipboard set-primary --user 0") {
		t.Errorf("expected clipboard set-primary, got %q", cmd)
	}
	// The base64 payload must decode back to the original text.
	encoded := base64.StdEncoding.EncodeToString([]byte("héllo"))
	if !strings.Contains(cmd, encoded) {
		t.Errorf("expected base64 payload %q in command %q", encoded, cmd)
	}
}

func TestBatchKeyeventUnsupported(t *testing.T) {
	cases := []struct {
		name           string
		stdout, stderr string
		want           bool
	}{
		{"supported, empty output", "", "", false},
		{"supported, whitespace only", "\n", "", false},
		{"unknown command in stdout", "Error: Unknown command: keyevent 67 67 67", "", true},
		{"invalid error in stderr", "", "error: invalid keyevent code", true},
		{"mixed case unknown command", "UNKNOWN COMMAND", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := batchKeyeventUnsupported(c.stdout, c.stderr); got != c.want {
				t.Errorf("batchKeyeventUnsupported(%q, %q) = %v, want %v", c.stdout, c.stderr, got, c.want)
			}
		})
	}
}
