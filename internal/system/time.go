package system

import (
	"context"
	"fmt"
	"regexp"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// TimeClient changes the device timezone.
type TimeClient struct {
	Adb *adb.Client
}

// NewTimeClient constructs a TimeClient.
func NewTimeClient(a *adb.Client) *TimeClient { return &TimeClient{Adb: a} }

var tzRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+\-_/]*$`)

// SetTimezone applies an Olson timezone identifier (e.g. "America/Los_Angeles")
// and broadcasts TIMEZONE_CHANGED.
func (t *TimeClient) SetTimezone(ctx context.Context, deviceID, tz string) error {
	if !tzRE.MatchString(tz) {
		return fmt.Errorf("invalid timezone %q", tz)
	}
	if _, err := t.Adb.ShellArgv(ctx, deviceID, "setprop", "persist.sys.timezone", tz); err != nil {
		return err
	}
	_, _ = t.Adb.ShellArgv(ctx, deviceID, "am", "broadcast", "-a", "android.intent.action.TIMEZONE_CHANGED", "--es", "time-zone", tz)
	return nil
}

var _ = adb.QuoteForShell
