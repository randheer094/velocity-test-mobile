// Package files wraps `adb push` / `adb pull`.
package files

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// Client groups push/pull helpers.
type Client struct {
	Adb *adb.Client
}

// New constructs a Client.
func New(a *adb.Client) *Client { return &Client{Adb: a} }

// Push uploads a local file to the device.
func (c *Client) Push(ctx context.Context, deviceID, local, remote string) error {
	if local == "" || remote == "" {
		return errors.New("both local and remote paths are required")
	}
	if !strings.HasPrefix(remote, "/") {
		return fmt.Errorf("remote path %q must be absolute", remote)
	}
	_, err := c.Adb.Run(ctx, deviceID, "push", local, remote)
	return err
}

// Pull downloads a device file to the host.
func (c *Client) Pull(ctx context.Context, deviceID, remote, local string) error {
	if local == "" || remote == "" {
		return errors.New("both local and remote paths are required")
	}
	if !strings.HasPrefix(remote, "/") {
		return fmt.Errorf("remote path %q must be absolute", remote)
	}
	_, err := c.Adb.Run(ctx, deviceID, "pull", remote, local)
	return err
}
