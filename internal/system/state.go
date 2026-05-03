package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// StateClient bundles device-state simulation knobs that environmental
// regression tests routinely need: font scale, dark mode, airplane mode,
// battery state, mobile/wifi connectivity, and per-app locale.
//
// Each method maps to one or two `cmd`/`settings`/`svc`/`dumpsys`
// invocations. None of these survive a device reboot — call them again at
// the start of each test session.
type StateClient struct {
	Adb *adb.Client
}

// NewStateClient constructs a StateClient.
func NewStateClient(a *adb.Client) *StateClient { return &StateClient{Adb: a} }

// SetFontScale writes the system-wide font scale (Settings → Display →
// Font size). 1.0 is the platform default; tests typically use 1.3 (large),
// 0.85 (small), or 2.0 (largest) to verify layout robustness.
func (s *StateClient) SetFontScale(ctx context.Context, deviceID string, scale float64) error {
	if scale < 0.5 || scale > 2.5 {
		return fmt.Errorf("font scale must be in [0.5, 2.5], got %g", scale)
	}
	_, err := s.Adb.ShellArgv(ctx, deviceID, "settings", "put", "system", "font_scale", fmt.Sprintf("%g", scale))
	return err
}

// SetDarkMode toggles the system UI dark mode. Maps to `cmd uimode night
// yes|no|auto`. "auto" follows the device schedule (day/night sensor or
// time of day, depending on Android version).
func (s *StateClient) SetDarkMode(ctx context.Context, deviceID, mode string) error {
	switch mode {
	case "yes", "no", "auto":
	default:
		return fmt.Errorf("dark mode must be yes|no|auto, got %q", mode)
	}
	_, err := s.Adb.ShellArgv(ctx, deviceID, "cmd", "uimode", "night", mode)
	return err
}

// SetAirplaneMode toggles airplane mode via `cmd connectivity
// airplane-mode`. On older Android versions this command may not exist;
// the error surfaces verbatim from the shell.
func (s *StateClient) SetAirplaneMode(ctx context.Context, deviceID string, on bool) error {
	verb := "disable"
	if on {
		verb = "enable"
	}
	_, err := s.Adb.ShellArgv(ctx, deviceID, "cmd", "connectivity", "airplane-mode", verb)
	return err
}

// BatteryState describes the test-time overrides supplied to `dumpsys
// battery set`. Only set the fields you want to override; the rest stay at
// the device's actual reported values until BatteryReset is called.
//
// Status uses the integer codes defined by BatteryManager:
//
//	1 unknown · 2 charging · 3 discharging · 4 not_charging · 5 full
type BatteryState struct {
	// Level is the battery percentage in [0,100]; -1 leaves it unset.
	Level int `json:"level"`
	// Status is the integer status code; 0 leaves it unset.
	Status int `json:"status"`
	// AC, USB, Wireless: 0 = unset, 1 = unplugged, 2 = plugged.
	AC       int `json:"ac"`
	USB      int `json:"usb"`
	Wireless int `json:"wireless"`
}

// SetBattery applies every set field via `dumpsys battery set ...`. Always
// pair test-time overrides with `BatteryReset` in cleanup; otherwise the
// device keeps reporting fake state until reboot.
func (s *StateClient) SetBattery(ctx context.Context, deviceID string, st BatteryState) error {
	type kv struct{ key, val string }
	var ops []kv
	if st.Level >= 0 && st.Level <= 100 {
		ops = append(ops, kv{"level", fmt.Sprintf("%d", st.Level)})
	} else if st.Level != -1 {
		return fmt.Errorf("battery level must be 0..100, got %d", st.Level)
	}
	if st.Status != 0 {
		if st.Status < 1 || st.Status > 5 {
			return fmt.Errorf("battery status must be 1..5, got %d", st.Status)
		}
		ops = append(ops, kv{"status", fmt.Sprintf("%d", st.Status)})
	}
	for _, p := range []struct {
		field string
		v     int
	}{{"ac", st.AC}, {"usb", st.USB}, {"wireless", st.Wireless}} {
		if p.v == 0 {
			continue
		}
		if p.v != 1 && p.v != 2 {
			return fmt.Errorf("battery %s must be 1 (unplugged) or 2 (plugged), got %d", p.field, p.v)
		}
		ops = append(ops, kv{p.field, fmt.Sprintf("%d", p.v-1)})
	}
	if len(ops) == 0 {
		return fmt.Errorf("battery_set_state: at least one field must be set")
	}
	for _, op := range ops {
		if _, err := s.Adb.ShellArgv(ctx, deviceID, "dumpsys", "battery", "set", op.key, op.val); err != nil {
			return err
		}
	}
	return nil
}

// BatteryReset clears every override applied via SetBattery.
func (s *StateClient) BatteryReset(ctx context.Context, deviceID string) error {
	_, err := s.Adb.ShellArgv(ctx, deviceID, "dumpsys", "battery", "reset")
	return err
}

// SetWifi toggles the wifi radio via `svc wifi enable|disable`.
func (s *StateClient) SetWifi(ctx context.Context, deviceID string, on bool) error {
	verb := "disable"
	if on {
		verb = "enable"
	}
	_, err := s.Adb.ShellArgv(ctx, deviceID, "svc", "wifi", verb)
	return err
}

// SetMobileData toggles cellular data via `svc data enable|disable`.
func (s *StateClient) SetMobileData(ctx context.Context, deviceID string, on bool) error {
	verb := "disable"
	if on {
		verb = "enable"
	}
	_, err := s.Adb.ShellArgv(ctx, deviceID, "svc", "data", verb)
	return err
}

// SetAppLocale applies a per-app locale tag via `cmd locale
// set-app-locales`. tag is a BCP-47 string (e.g. "ja-JP", "fr-CA"); pass
// empty to clear the override and fall back to device locale.
func (s *StateClient) SetAppLocale(ctx context.Context, deviceID, pkg, tag string) error {
	if pkg == "" {
		return fmt.Errorf("package is required")
	}
	args := []string{"cmd", "locale", "set-app-locales", "--user", "0", "--package", pkg}
	if strings.TrimSpace(tag) == "" {
		args = append(args, "--locales", "")
	} else {
		args = append(args, "--locales", tag)
	}
	_, err := s.Adb.ShellArgv(ctx, deviceID, args...)
	return err
}
