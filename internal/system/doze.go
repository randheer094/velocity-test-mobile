package system

import (
	"context"
	"fmt"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// DozeClient simulates Doze / battery-saver state.
type DozeClient struct {
	Adb *adb.Client
}

// NewDozeClient constructs a DozeClient.
func NewDozeClient(a *adb.Client) *DozeClient { return &DozeClient{Adb: a} }

// SetState forces deep idle ("idle") or restores normal operation ("active").
func (d *DozeClient) SetState(ctx context.Context, deviceID, state string) error {
	switch state {
	case "idle":
		if _, err := d.Adb.ShellArgv(ctx, deviceID, "cmd", "deviceidle", "force-idle", "deep"); err != nil {
			return fmt.Errorf("force-idle: %w", err)
		}
	case "active":
		if _, err := d.Adb.ShellArgv(ctx, deviceID, "cmd", "deviceidle", "unforce"); err != nil {
			return fmt.Errorf("unforce: %w", err)
		}
		_, _ = d.Adb.ShellArgv(ctx, deviceID, "cmd", "deviceidle", "disable")
	default:
		return fmt.Errorf("invalid state %q (expected idle|active)", state)
	}
	return nil
}

var _ = adb.QuoteForShell
