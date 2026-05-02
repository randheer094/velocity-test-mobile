package adb

import (
	"strings"
	"testing"
)

func TestBaseArgs(t *testing.T) {
	c := &Client{}
	got := c.baseArgs("emulator-5554", "shell", "wm", "size")
	want := []string{"-s", "emulator-5554", "shell", "wm", "size"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %v, want %v", got, want)
	}
	got2 := c.baseArgs("", "devices", "-l")
	if strings.Join(got2, " ") != "devices -l" {
		t.Fatalf("got %v", got2)
	}
}

func TestQuoteForShell(t *testing.T) {
	cases := map[string]string{
		"":             "''",
		"hello":        "'hello'",
		"a b c":        "'a b c'",
		"it's":         `'it'\''s'`,
		"$(rm -rf /)":  `'$(rm -rf /)'`,
		"line1\nline2": "'line1\nline2'",
	}
	for in, want := range cases {
		if got := QuoteForShell(in); got != want {
			t.Errorf("QuoteForShell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMustQuotePackage(t *testing.T) {
	good := []string{"com.example.app", "com.foo_bar.baz", "x"}
	for _, p := range good {
		if _, err := MustQuotePackage(p); err != nil {
			t.Errorf("unexpected error for %q: %v", p, err)
		}
	}
	bad := []string{"", "com.example app", "com.example;ls", "$pkg"}
	for _, p := range bad {
		if _, err := MustQuotePackage(p); err == nil {
			t.Errorf("expected error for %q", p)
		}
	}
}

func TestKeycode(t *testing.T) {
	cases := map[string]int{
		"BACK":          4,
		"home":          3,
		"KEYCODE_ENTER": 66,
		"  power  ":     26,
		"A":             29,
		"z":             54,
		"0":             7,
		"9":             16,
		"MOVE_END":      123,
		"CTRL_LEFT":     113,
	}
	for in, want := range cases {
		got, err := Keycode(in)
		if err != nil {
			t.Errorf("Keycode(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Keycode(%q) = %d, want %d", in, got, want)
		}
	}
	if _, err := Keycode("BANANA"); err == nil {
		t.Error("expected error for unknown key")
	}
	if _, err := Keycode("AB"); err == nil {
		t.Error("expected error for two-letter sequence")
	}
}
