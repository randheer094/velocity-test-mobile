package apps

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// AppOpMode is the legal value space for `appops set`.
//
// `default` resets the op to its system default.
type AppOpMode string

const (
	AppOpAllow   AppOpMode = "allow"
	AppOpDeny    AppOpMode = "deny"
	AppOpIgnore  AppOpMode = "ignore"
	AppOpDefault AppOpMode = "default"
)

func (m AppOpMode) valid() bool {
	switch m {
	case AppOpAllow, AppOpDeny, AppOpIgnore, AppOpDefault:
		return true
	}
	return false
}

// op names look like `android:mock_location` or `MOCK_LOCATION`. Allow letters,
// digits, dot, colon, underscore — strict enough to keep shell injection out.
var appOpRE = regexp.MustCompile(`^[A-Za-z0-9_.:]+$`)

func validateOp(op string) error {
	if op == "" {
		return fmt.Errorf("appops op is empty")
	}
	if !appOpRE.MatchString(op) {
		return fmt.Errorf("invalid appops op %q", op)
	}
	return nil
}

// SetAppOp runs `appops set <pkg> <op> <mode>`.
func (c *Client) SetAppOp(ctx context.Context, deviceID, pkg, op string, mode AppOpMode) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	if err := validateOp(op); err != nil {
		return err
	}
	if !mode.valid() {
		return fmt.Errorf("invalid appops mode %q (expected allow|deny|ignore|default)", mode)
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "appops", "set", pkg, op, string(mode))
	if err != nil {
		return err
	}
	combined := strings.TrimSpace(string(res.Stdout) + string(res.Stderr))
	// `appops set` prints nothing on success. Any output usually signals an error.
	if combined != "" && (strings.Contains(combined, "Error") || strings.Contains(combined, "Unknown") || strings.Contains(combined, "Failure")) {
		return fmt.Errorf("appops set failed: %s", combined)
	}
	return nil
}

// GetAppOp returns the mode for a single op on `pkg`. When the op is not
// listed by `appops get`, the system default applies — reported as
// AppOpDefault.
func (c *Client) GetAppOp(ctx context.Context, deviceID, pkg, op string) (AppOpMode, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return "", err
	}
	if err := validateOp(op); err != nil {
		return "", err
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "appops", "get", pkg, op)
	if err != nil {
		return "", err
	}
	return parseAppOpMode(string(res.Stdout) + string(res.Stderr))
}

// parseAppOpMode parses `appops get` output. Lines look like:
//
//	android:mock_location: allow
//	MOCK_LOCATION: deny; time=+1h2m3s
//	No operations.
func parseAppOpMode(out string) (AppOpMode, error) {
	out = strings.TrimSpace(out)
	if out == "" || strings.Contains(out, "No operations") {
		return AppOpDefault, nil
	}
	// Take the first non-empty line. `appops get pkg op` returns at most one.
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<op>: <mode>[; ...]". The op itself can contain colons
		// (`android:mock_location`), so split on the *last* ": " separator.
		idx := strings.LastIndex(line, ": ")
		if idx < 0 {
			// Fall back to last colon — handles a missing space after the op.
			idx = strings.LastIndex(line, ":")
		}
		if idx < 0 {
			return "", fmt.Errorf("could not parse appops mode from %q", line)
		}
		rest := strings.TrimSpace(line[idx+1:])
		if j := strings.IndexAny(rest, "; \t"); j >= 0 {
			rest = rest[:j]
		}
		rest = strings.ToLower(rest)
		switch rest {
		case "allow", "deny", "ignore", "default":
			return AppOpMode(rest), nil
		}
		return "", fmt.Errorf("could not parse appops mode from %q", line)
	}
	return AppOpDefault, nil
}
