// Package system covers device-state knobs: screen, animations, doze,
// network, time, location.
package system

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// DefaultSizeCacheTTL bounds how long a Get() result is reused per device.
// Screen size/density is read-mostly (it doesn't change with orientation —
// `wm size`/`wm density` report the display's physical/override size, not
// the current rotation), so this avoids two subprocess spawns on every
// device_get_screen_size call within the TTL window.
const DefaultSizeCacheTTL = 30 * time.Second

// ScreenClient covers wm size, orientation, wake/sleep.
type ScreenClient struct {
	Adb      *adb.Client
	cacheTTL time.Duration

	cacheMu sync.Mutex
	cache   map[string]cachedSize

	now   func() time.Time                                         // test seam; nil means time.Now
	fetch func(ctx context.Context, deviceID string) (Size, error) // test seam; nil means real adb call
}

type cachedSize struct {
	size Size
	at   time.Time
}

// NewScreenClient constructs a ScreenClient.
func NewScreenClient(a *adb.Client) *ScreenClient {
	return &ScreenClient{Adb: a, cacheTTL: DefaultSizeCacheTTL}
}

// Size is reported by `wm size`.
type Size struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Density int `json:"density"`
}

// Get returns physical screen size and density, fusing `wm size` and
// `wm density` into a single adb shell call instead of two, and caching the
// result per device for cacheTTL.
func (s *ScreenClient) Get(ctx context.Context, deviceID string) (Size, error) {
	s.cacheMu.Lock()
	if cached, ok := s.cache[deviceID]; ok && s.nowFunc().Sub(cached.at) < s.cacheTTL {
		s.cacheMu.Unlock()
		return cached.size, nil
	}
	s.cacheMu.Unlock()

	size, err := s.fetchSize(ctx, deviceID)
	if err != nil {
		return Size{}, err
	}

	s.cacheMu.Lock()
	if s.cache == nil {
		s.cache = make(map[string]cachedSize)
	}
	s.cache[deviceID] = cachedSize{size: size, at: s.nowFunc()}
	s.cacheMu.Unlock()
	return size, nil
}

func (s *ScreenClient) nowFunc() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *ScreenClient) fetchSize(ctx context.Context, deviceID string) (Size, error) {
	if s.fetch != nil {
		return s.fetch(ctx, deviceID)
	}
	// `wm density` runs first so its exit status can't mask `wm size`'s —
	// matches the original two-call behavior, where a density failure was
	// non-fatal (density just came back 0) but a size failure was not.
	res, err := s.Adb.Shell(ctx, deviceID, "wm density; wm size")
	if err != nil {
		return Size{}, err
	}
	out := string(res.Stdout)
	w, h := parseWMSize(out)
	d := parseWMDensity(out)
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
	_, err := s.Adb.Shell(ctx, deviceID, buildSetOrientationCmd(rot))
	return err
}

func buildSetOrientationCmd(rot string) string {
	return "settings put system accelerometer_rotation 0; settings put system user_rotation " + rot
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
