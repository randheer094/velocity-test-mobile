package system

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// ActivityClient covers `dumpsys activity` introspection and `am start`
// component dispatch — the primitives needed for "what's on top?" assertions
// and explicit activity launches that bypass launcher resolution.
type ActivityClient struct {
	Adb *adb.Client
}

// NewActivityClient constructs an ActivityClient.
func NewActivityClient(a *adb.Client) *ActivityClient { return &ActivityClient{Adb: a} }

// TopActivity describes the currently-resumed activity (`topResumedActivity`
// in `dumpsys activity activities`). Nil when no resumed activity is reported.
type TopActivity struct {
	Package  string `json:"bundle_id"`
	Activity string `json:"activity"`
	TaskID   *int   `json:"task_id,omitempty"`
}

// topRE matches a dumpsys line like:
//
//	  topResumedActivity=ActivityRecord{abcd1234 u0 com.example/.MainActivity t42}
//
// or
//
//	  mResumedActivity: ActivityRecord{... com.example/.MainActivity t42}
var topRE = regexp.MustCompile(`(?:topResumedActivity|mResumedActivity)[=:].*?ActivityRecord\{[^}]*?\s+([A-Za-z0-9_.]+)/(\.?[A-Za-z0-9_.\$]+)(?:\s+t(\d+))?`)

// GetTop returns the currently-resumed activity, or nil if none was reported.
func (c *ActivityClient) GetTop(ctx context.Context, deviceID string) (*TopActivity, error) {
	res, err := c.Adb.ShellArgv(ctx, deviceID, "dumpsys", "activity", "activities")
	if err != nil {
		return nil, err
	}
	return parseTopActivity(string(res.Stdout))
}

func parseTopActivity(out string) (*TopActivity, error) {
	m := topRE.FindStringSubmatch(out)
	if m == nil {
		return nil, nil
	}
	pkg := m[1]
	act := m[2]
	if strings.HasPrefix(act, ".") {
		act = pkg + act
	}
	t := &TopActivity{Package: pkg, Activity: act}
	if len(m) > 3 && m[3] != "" {
		if v, err := strconv.Atoi(m[3]); err == nil {
			t.TaskID = &v
		}
	}
	return t, nil
}

// WaitForTop polls GetTop until the resumed activity matches `pkg` (and
// `activityClass` when non-empty), or the timeout elapses.
func (c *ActivityClient) WaitForTop(ctx context.Context, deviceID, pkg, activityClass string, timeout time.Duration) (*TopActivity, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var last *TopActivity
	for {
		t, err := c.GetTop(ctx, deviceID)
		if err == nil && t != nil {
			last = t
			if (pkg == "" || t.Package == pkg) && (activityClass == "" || matchesActivity(t.Activity, pkg, activityClass)) {
				return t, nil
			}
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	if last == nil {
		return nil, fmt.Errorf("no resumed activity within %s", timeout)
	}
	return last, fmt.Errorf("top activity is %s/%s, expected %s/%s", last.Package, last.Activity, pkg, activityClass)
}

// matchesActivity accepts either fully-qualified ("com.foo.MainActivity") or
// relative (".MainActivity") forms for the expected class.
func matchesActivity(actual, pkg, expected string) bool {
	if expected == "" {
		return true
	}
	if actual == expected {
		return true
	}
	if strings.HasPrefix(expected, ".") && actual == pkg+expected {
		return true
	}
	if strings.HasSuffix(actual, "."+strings.TrimPrefix(expected, ".")) {
		return true
	}
	return false
}

// activityClassRE permits both relative (".MainActivity") and absolute forms.
var activityClassRE = regexp.MustCompile(`^\.?[A-Za-z][A-Za-z0-9_.\$]*$`)

// flagRE matches decimal or 0xHEX values, the same shape used by Intent.Flags.
var flagREActivity = regexp.MustCompile(`^0x[0-9A-Fa-f]+$|^[0-9]+$`)

// extraKeyRE matches Android intent extra keys.
var extraKeyRE = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)

// StartArgs describes an explicit `am start -n <pkg>/<activity>` invocation.
type StartArgs struct {
	Package  string
	Activity string
	Action   string
	Data     string
	Flags    []string
	StringEx map[string]string
	IntEx    map[string]string
	BoolEx   map[string]string
	FloatEx  map[string]string
}

// Start invokes `am start -n <pkg>/<activity>`, bypassing launcher resolution.
// Used when the launcher activity Android picks for `monkey -c LAUNCHER` is
// the wrong one (e.g. LeakCanary's LeakLauncherActivity outranks the app's
// real entry point).
func (c *ActivityClient) Start(ctx context.Context, deviceID string, args StartArgs) error {
	if _, err := adb.MustQuotePackage(args.Package); err != nil {
		return err
	}
	if args.Activity == "" || !activityClassRE.MatchString(args.Activity) {
		return fmt.Errorf("invalid activity class %q", args.Activity)
	}
	component := args.Package + "/" + args.Activity
	argv := []string{"am", "start", "-n", component}
	if args.Action != "" {
		// Reuse the strict action regex from apps/intents — same shape.
		if !regexp.MustCompile(`^[A-Za-z0-9_.]+$`).MatchString(args.Action) {
			return fmt.Errorf("invalid action %q", args.Action)
		}
		argv = append(argv, "-a", args.Action)
	}
	if args.Data != "" {
		if _, err := url.Parse(args.Data); err != nil {
			return fmt.Errorf("invalid data uri %q: %w", args.Data, err)
		}
		argv = append(argv, "-d", args.Data)
	}
	for _, f := range args.Flags {
		if !flagREActivity.MatchString(f) {
			return fmt.Errorf("invalid flag %q (expect decimal or 0xHEX)", f)
		}
		argv = append(argv, "-f", f)
	}
	for k, v := range args.StringEx {
		if !extraKeyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--es", k, v)
	}
	for k, v := range args.IntEx {
		if !extraKeyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--ei", k, v)
	}
	for k, v := range args.BoolEx {
		if !extraKeyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--ez", k, v)
	}
	for k, v := range args.FloatEx {
		if !extraKeyRE.MatchString(k) {
			return fmt.Errorf("invalid extra key %q", k)
		}
		argv = append(argv, "--ef", k, v)
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, argv...)
	if err != nil {
		return err
	}
	combined := string(res.Stdout) + string(res.Stderr)
	if strings.Contains(combined, "Error:") || strings.Contains(combined, "Exception") {
		return fmt.Errorf("am start failed: %s", strings.TrimSpace(combined))
	}
	return nil
}
