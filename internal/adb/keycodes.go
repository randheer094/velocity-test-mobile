package adb

import (
	"fmt"
	"sort"
	"strings"
)

// Keycodes maps friendly names to Android KeyEvent codes used by
// `input keyevent <code>`.
var Keycodes = map[string]int{
	"BACK":             4,
	"HOME":             3,
	"RECENTS":          187,
	"APP_SWITCH":       187,
	"MENU":             82,
	"ENTER":            66,
	"TAB":              61,
	"SPACE":            62,
	"DEL":              67,
	"ESCAPE":           111,
	"DPAD_UP":          19,
	"DPAD_DOWN":        20,
	"DPAD_LEFT":        21,
	"DPAD_RIGHT":       22,
	"DPAD_CENTER":      23,
	"VOLUME_UP":        24,
	"VOLUME_DOWN":      25,
	"VOLUME_MUTE":      164,
	"MUTE":             91,
	"POWER":            26,
	"WAKEUP":           224,
	"SLEEP":            223,
	"CAMERA":           27,
	"SEARCH":           84,
	"BRIGHTNESS_UP":    221,
	"BRIGHTNESS_DOWN":  220,
	"MEDIA_PLAY_PAUSE": 85,
	"MEDIA_NEXT":       87,
	"MEDIA_PREVIOUS":   88,
	"MEDIA_STOP":       86,
	"NOTIFICATION":     83,
	"PASTE":            279,
	"COPY":             278,
	"CUT":              277,
	"MOVE_END":         123,
	"MOVE_HOME":        122,
	"FORWARD_DEL":      112,
	"CTRL_LEFT":        113,
	"CTRL_RIGHT":       114,
	"SHIFT_LEFT":       59,
	"SHIFT_RIGHT":      60,
	"ALT_LEFT":         57,
	"ALT_RIGHT":        58,
	"SELECT_ALL":       2003, // androidx test-style alias; we map to CTRL_LEFT+A via combination
	"PAGE_UP":          92,
	"PAGE_DOWN":        93,
	"NUMPAD_ENTER":     160,
}

// Keycode looks up a keyname (case-insensitive, leading KEYCODE_ stripped).
//
// Single uppercase letters (A-Z) map to Android keycodes 29..54, and single
// digits (0-9) map to keycodes 7..16. This lets callers pass a literal
// character to PressKeyCombination, e.g. PressKeyCombination("CTRL_LEFT", "A").
func Keycode(name string) (int, error) {
	key := strings.ToUpper(strings.TrimSpace(name))
	key = strings.TrimPrefix(key, "KEYCODE_")
	if c, ok := Keycodes[key]; ok {
		return c, nil
	}
	if len(key) == 1 {
		r := key[0]
		if r >= 'A' && r <= 'Z' {
			return 29 + int(r-'A'), nil
		}
		if r >= '0' && r <= '9' {
			return 7 + int(r-'0'), nil
		}
	}
	return 0, fmt.Errorf("unknown button %q (try one of: %s)", name, KnownButtons())
}

// KnownButtons returns a comma-separated, alphabetically sorted list of
// supported button names; intended for error messages.
func KnownButtons() string {
	names := make([]string, 0, len(Keycodes))
	for k := range Keycodes {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
