package apps

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// Intent describes the parameters for `am start` / `am broadcast`.
type Intent struct {
	Mode     string            // "start" (default) or "broadcast"
	Action   string            // -a
	Category string            // -c
	Data     string            // -d (URI)
	MimeType string            // -t
	Package  string            // restrict to a specific package
	Class    string            // -n component, "<pkg>/<class>"
	Flags    []string          // -f hex flags
	StringEx map[string]string // --es key value
	IntEx    map[string]string // --ei key value
	BoolEx   map[string]string // --ez key value
	FloatEx  map[string]string // --ef key value
}

var (
	actionRE   = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)
	categoryRE = regexp.MustCompile(`^[A-Za-z0-9_.,]+$`)
	flagRE     = regexp.MustCompile(`^0x[0-9A-Fa-f]+$|^[0-9]+$`)
	keyRE      = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)
)

// SendIntent dispatches an intent via `am`.
func (c *Client) SendIntent(ctx context.Context, deviceID string, intent Intent) error {
	mode := intent.Mode
	if mode == "" {
		mode = "start"
	}
	if mode != "start" && mode != "broadcast" {
		return fmt.Errorf("invalid intent mode %q (expected start or broadcast)", mode)
	}

	argv := []string{"am", mode}
	if intent.Action != "" {
		if !actionRE.MatchString(intent.Action) {
			return fmt.Errorf("invalid action %q", intent.Action)
		}
		argv = append(argv, "-a", intent.Action)
	}
	if intent.Category != "" {
		if !categoryRE.MatchString(intent.Category) {
			return fmt.Errorf("invalid category %q", intent.Category)
		}
		argv = append(argv, "-c", intent.Category)
	}
	if intent.Data != "" {
		if _, err := url.Parse(intent.Data); err != nil {
			return fmt.Errorf("invalid data uri %q: %w", intent.Data, err)
		}
		argv = append(argv, "-d", intent.Data)
	}
	if intent.MimeType != "" {
		argv = append(argv, "-t", intent.MimeType)
	}
	if intent.Package != "" {
		if _, err := adb.MustQuotePackage(intent.Package); err != nil {
			return err
		}
		argv = append(argv, "-p", intent.Package)
	}
	if intent.Class != "" {
		argv = append(argv, "-n", intent.Class)
	}
	for _, f := range intent.Flags {
		if !flagRE.MatchString(f) {
			return fmt.Errorf("invalid flag %q (expect decimal or 0xHEX)", f)
		}
		argv = append(argv, "-f", f)
	}
	for k, v := range intent.StringEx {
		if !keyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--es", k, v)
	}
	for k, v := range intent.IntEx {
		if !keyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--ei", k, v)
	}
	for k, v := range intent.BoolEx {
		if !keyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--ez", k, v)
	}
	for k, v := range intent.FloatEx {
		if !keyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--ef", k, v)
	}

	_, err := c.Adb.ShellArgv(ctx, deviceID, argv...)
	return err
}
