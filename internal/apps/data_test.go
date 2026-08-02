package apps

import "testing"

func TestListAppData_InvalidPackage(t *testing.T) {
	c := newTestClient()
	if _, err := c.ListAppData(nil, "dev", "com.example; reboot", ""); err == nil {
		t.Error("expected error for an invalid package name")
	}
}

func TestListAppData_InvalidPath(t *testing.T) {
	c := newTestClient()
	if _, err := c.ListAppData(nil, "dev", "com.example", "../etc/passwd"); err == nil {
		t.Error("expected error for a traversal path")
	}
}

func TestReadAppData_RequiresPath(t *testing.T) {
	c := newTestClient()
	if _, err := c.ReadAppData(nil, "dev", "com.example", ""); err == nil {
		t.Error("expected error for an empty relativePath")
	}
}

func TestReadAppData_InvalidPackage(t *testing.T) {
	c := newTestClient()
	if _, err := c.ReadAppData(nil, "dev", "com.example`whoami`", "file.txt"); err == nil {
		t.Error("expected error for an invalid package name")
	}
}
