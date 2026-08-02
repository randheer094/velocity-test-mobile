package system

import "testing"

func TestShellExec_RejectsEmptyCommand(t *testing.T) {
	c := NewShellClient(nil)
	if _, err := c.Exec(nil, "dev", "", 0); err == nil {
		t.Error("expected error for an empty command")
	}
}
