package apps

import "testing"

func TestParseAppOpMode(t *testing.T) {
	cases := []struct {
		in   string
		want AppOpMode
	}{
		{"android:mock_location: allow\n", AppOpAllow},
		{"MOCK_LOCATION: deny; time=+1h2m3s; duration=+0s\n", AppOpDeny},
		{"  android:fine_location: ignore", AppOpIgnore},
		{"No operations.", AppOpDefault},
		{"", AppOpDefault},
	}
	for _, c := range cases {
		got, err := parseAppOpMode(c.in)
		if err != nil {
			t.Errorf("parseAppOpMode(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseAppOpMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
