package diagnostics

import (
	"reflect"
	"testing"
)

func TestSplitLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"one line\n", []string{"one line"}},
		{"a\nb\nc\n", []string{"a", "b", "c"}},
		{"a\nb", []string{"a", "b"}}, // no trailing newline
	}
	for _, c := range cases {
		got := splitLines([]byte(c.in))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitLines(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
