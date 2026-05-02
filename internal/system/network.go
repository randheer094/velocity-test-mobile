package system

import (
	"context"
	"fmt"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// NetworkClient toggles airplane / wifi / mobile-data state.
type NetworkClient struct {
	Adb *adb.Client
}

// NewNetworkClient constructs a NetworkClient.
func NewNetworkClient(a *adb.Client) *NetworkClient { return &NetworkClient{Adb: a} }

// SetAirplaneMode enables/disables airplane mode (Android 11+ via cmd connectivity;
// older devices fall back to settings + broadcast).
func (n *NetworkClient) SetAirplaneMode(ctx context.Context, deviceID string, enabled bool) error {
	state := "disable"
	if enabled {
		state = "enable"
	}
	if _, err := n.Adb.ShellArgv(ctx, deviceID, "cmd", "connectivity", "airplane-mode", state); err == nil {
		return nil
	}
	val := "0"
	if enabled {
		val = "1"
	}
	if _, err := n.Adb.ShellArgv(ctx, deviceID, "settings", "put", "global", "airplane_mode_on", val); err != nil {
		return err
	}
	_, err := n.Adb.ShellArgv(ctx, deviceID, "am", "broadcast", "-a", "android.intent.action.AIRPLANE_MODE", "--ez", "state", boolStr(enabled))
	return err
}

// SetWiFi enables/disables Wi-Fi via the svc helper.
func (n *NetworkClient) SetWiFi(ctx context.Context, deviceID string, enabled bool) error {
	state := "disable"
	if enabled {
		state = "enable"
	}
	_, err := n.Adb.ShellArgv(ctx, deviceID, "svc", "wifi", state)
	if err != nil {
		return fmt.Errorf("svc wifi %s: %w (root may be required on some devices)", state, err)
	}
	return nil
}

// SetMobileData enables/disables mobile data via the svc helper.
func (n *NetworkClient) SetMobileData(ctx context.Context, deviceID string, enabled bool) error {
	state := "disable"
	if enabled {
		state = "enable"
	}
	_, err := n.Adb.ShellArgv(ctx, deviceID, "svc", "data", state)
	if err != nil {
		return fmt.Errorf("svc data %s: %w (root may be required on some devices)", state, err)
	}
	return nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

var _ = adb.QuoteForShell
