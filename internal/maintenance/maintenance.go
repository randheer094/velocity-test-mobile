// Package maintenance covers reboot and wireless-ADB pairing.
package maintenance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// Client groups maintenance commands.
type Client struct {
	Adb *adb.Client
}

// New constructs a Client.
func New(a *adb.Client) *Client { return &Client{Adb: a} }

// allowed reboot modes.
var rebootModes = map[string]struct{}{
	"":           {},
	"bootloader": {},
	"recovery":   {},
	"sideload":   {},
	"fastboot":   {},
}

// Reboot reboots the device. The action is destructive, so callers must
// pass confirm=true; otherwise the call is rejected.
func (c *Client) Reboot(ctx context.Context, deviceID, mode string, confirm bool) error {
	if !confirm {
		return errors.New("reboot is destructive; pass confirm=true to proceed")
	}
	if _, ok := rebootModes[mode]; !ok {
		return fmt.Errorf("invalid reboot mode %q", mode)
	}
	args := []string{"reboot"}
	if mode != "" {
		args = append(args, mode)
	}
	_, err := c.Adb.Run(ctx, deviceID, args...)
	return err
}

// EnableWirelessADB switches the (USB-attached) device to TCP/IP mode on
// the supplied port. After this the caller can use ConnectWireless using
// the device's IP.
func (c *Client) EnableWirelessADB(ctx context.Context, deviceID string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of range", port)
	}
	_, err := c.Adb.Run(ctx, deviceID, "tcpip", fmt.Sprintf("%d", port))
	return err
}

// ConnectWireless attaches to a wireless device.
func (c *Client) ConnectWireless(ctx context.Context, hostPort string) (string, error) {
	if !strings.Contains(hostPort, ":") {
		hostPort = hostPort + ":5555"
	}
	if _, _, err := net.SplitHostPort(hostPort); err != nil {
		return "", fmt.Errorf("invalid host:port %q: %w", hostPort, err)
	}
	res, err := c.Adb.Run(ctx, "", "connect", hostPort)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// PairWireless performs `adb pair host:port` (Android 11+) using the supplied
// 6-digit code. The code is fed to adb on stdin.
func (c *Client) PairWireless(ctx context.Context, hostPort, code string) (string, error) {
	if _, _, err := net.SplitHostPort(hostPort); err != nil {
		return "", fmt.Errorf("invalid host:port %q: %w", hostPort, err)
	}
	if len(code) < 4 || len(code) > 12 {
		return "", fmt.Errorf("pairing code looks wrong (length %d)", len(code))
	}
	res, err := c.Adb.Run(ctx, "", "pair", hostPort, code)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// DisconnectWireless detaches from a wireless host (or all if hostPort is "").
func (c *Client) DisconnectWireless(ctx context.Context, hostPort string) error {
	args := []string{"disconnect"}
	if hostPort != "" {
		args = append(args, hostPort)
	}
	_, err := c.Adb.Run(ctx, "", args...)
	return err
}
