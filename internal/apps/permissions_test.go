package apps

import "testing"

func TestGrantPermission_InvalidPackage(t *testing.T) {
	c := newTestClient()
	if err := c.GrantPermission(nil, "dev", "com.example; reboot", "android.permission.CAMERA"); err == nil {
		t.Error("expected error for an invalid package name")
	}
}

func TestGrantPermission_InvalidPermission(t *testing.T) {
	c := newTestClient()
	if err := c.GrantPermission(nil, "dev", "com.example", "android.permission.CAMERA; reboot"); err == nil {
		t.Error("expected error for an invalid permission name")
	}
}

func TestRevokePermission_InvalidPackage(t *testing.T) {
	c := newTestClient()
	if err := c.RevokePermission(nil, "dev", "../etc/passwd", "android.permission.CAMERA"); err == nil {
		t.Error("expected error for an invalid package name")
	}
}

func TestRevokePermission_InvalidPermission(t *testing.T) {
	c := newTestClient()
	if err := c.RevokePermission(nil, "dev", "com.example", "not a permission"); err == nil {
		t.Error("expected error for an invalid permission name")
	}
}
