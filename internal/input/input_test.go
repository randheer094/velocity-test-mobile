package input

import "testing"

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
