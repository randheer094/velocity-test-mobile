// Package system covers device-state knobs: screen, animations, doze,
// network, time, location.
package system

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// ScreenClient covers wm size, orientation, wake/sleep.
type ScreenClient struct {
	Adb *adb.Client
}

// NewScreenClient constructs a ScreenClient.
func NewScreenClient(a *adb.Client) *ScreenClient { return &ScreenClient{Adb: a} }

// Size is reported by `wm size`.
type Size struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Density int `json:"density"`
}

// Get returns physical screen size and density.
func (s *ScreenClient) Get(ctx context.Context, deviceID string) (Size, error) {
	res, err := s.Adb.ShellArgv(ctx, deviceID, "wm", "size")
	if err != nil {
		return Size{}, err
	}
	w, h := parseWMSize(string(res.Stdout))
	d := 0
	if res2, err := s.Adb.ShellArgv(ctx, deviceID, "wm", "density"); err == nil {
		d = parseWMDensity(string(res2.Stdout))
	}
	return Size{Width: w, Height: h, Density: d}, nil
}

// parseWMSize returns the active viewport. When `wm size` reports both
// "Physical size" and "Override size", the override wins (it is what the
// system is actually rendering at).
func parseWMSize(out string) (int, int) {
	var pw, ph, ow, oh int
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.Contains(line, "size:") {
			continue
		}
		idx := strings.LastIndex(line, " ")
		if idx < 0 {
			continue
		}
		parts := strings.Split(line[idx+1:], "x")
		if len(parts) != 2 {
			continue
		}
		w, _ := strconv.Atoi(parts[0])
		h, _ := strconv.Atoi(parts[1])
		if w <= 0 || h <= 0 {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "override") {
			ow, oh = w, h
		} else {
			pw, ph = w, h
		}
	}
	if ow > 0 {
		return ow, oh
	}
	return pw, ph
}

// parseWMDensity returns the override density when present, otherwise the
// physical density.
func parseWMDensity(out string) int {
	var phys, over int
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.Contains(line, "density:") {
			continue
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		v, _ := strconv.Atoi(f[len(f)-1])
		if v <= 0 {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "override") {
			over = v
		} else {
			phys = v
		}
	}
	if over > 0 {
		return over
	}
	return phys
}

// GetOrientation returns "portrait" / "landscape" based on user_rotation.
func (s *ScreenClient) GetOrientation(ctx context.Context, deviceID string) (string, error) {
	res, err := s.Adb.ShellArgv(ctx, deviceID, "settings", "get", "system", "user_rotation")
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(res.Stdout))
	switch v {
	case "0", "2":
		return "portrait", nil
	case "1", "3":
		return "landscape", nil
	default:
		return v, nil
	}
}

// SetOrientation locks the device into portrait or landscape.
func (s *ScreenClient) SetOrientation(ctx context.Context, deviceID, orientation string) error {
	var rot string
	switch orientation {
	case "portrait":
		rot = "0"
	case "landscape":
		rot = "1"
	default:
		return fmt.Errorf("invalid orientation %q (expected portrait|landscape)", orientation)
	}
	if _, err := s.Adb.ShellArgv(ctx, deviceID, "settings", "put", "system", "accelerometer_rotation", "0"); err != nil {
		return err
	}
	if _, err := s.Adb.ShellArgv(ctx, deviceID, "settings", "put", "system", "user_rotation", rot); err != nil {
		return err
	}
	return nil
}

// Wake wakes the device (KEYCODE_WAKEUP) and dismisses simple lock screens.
func (s *ScreenClient) Wake(ctx context.Context, deviceID string) error {
	if _, err := s.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", "224"); err != nil {
		return err
	}
	// Best-effort: swipe up from bottom to dismiss a simple lock screen.
	_, _ = s.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", "82")
	return nil
}

// Lock turns the screen off (KEYCODE_SLEEP).
func (s *ScreenClient) Lock(ctx context.Context, deviceID string) error {
	_, err := s.Adb.ShellArgv(ctx, deviceID, "input", "keyevent", "223")
	return err
}

// Reused by adb package consumers wanting QuoteForShell-style validation.
var _ = adb.QuoteForShell
