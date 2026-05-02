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
