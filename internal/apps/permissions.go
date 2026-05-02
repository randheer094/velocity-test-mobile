package apps

import (
	"context"
	"fmt"
	"regexp"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

var permRE = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)

// GrantPermission grants a runtime permission to a package.
func (c *Client) GrantPermission(ctx context.Context, deviceID, pkg, perm string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	if !permRE.MatchString(perm) {
		return fmt.Errorf("invalid permission name %q", perm)
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "pm", "grant", pkg, perm)
	return err
}

// RevokePermission revokes a runtime permission from a package.
func (c *Client) RevokePermission(ctx context.Context, deviceID, pkg, perm string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	if !permRE.MatchString(perm) {
		return fmt.Errorf("invalid permission name %q", perm)
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "pm", "revoke", pkg, perm)
	return err
}
