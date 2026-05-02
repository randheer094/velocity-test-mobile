package system

import "testing"

func TestParseWMSize(t *testing.T) {
	cases := []struct {
		in   string
		w, h int
	}{
		{"Physical size: 1080x2400\n", 1080, 2400},
		{"Physical size: 1080x2400\nOverride size: 720x1600\n", 720, 1600},
		{"\n", 0, 0},
	}
	for _, c := range cases {
		w, h := parseWMSize(c.in)
		if w != c.w || h != c.h {
			t.Errorf("parseWMSize(%q) = %dx%d, want %dx%d", c.in, w, h, c.w, c.h)
		}
	}
}

func TestParseWMDensity(t *testing.T) {
	out := "Physical density: 420\nOverride density: 480\n"
	if d := parseWMDensity(out); d != 480 {
		t.Errorf("density = %d, want 480", d)
	}
}
