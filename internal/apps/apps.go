// Package apps wraps adb commands related to application lifecycle and state.
package apps

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
)

// Client groups operations on installed apps.
type Client struct {
	Adb        *adb.Client
	AndroidCLI *androidcli.Client
}

// New constructs a Client.
func New(a *adb.Client, c *androidcli.Client) *Client { return &Client{Adb: a, AndroidCLI: c} }

// App identifies an installed launchable application.
type App struct {
	Package  string `json:"package"`
	Activity string `json:"activity,omitempty"`
}

// List returns installed apps that have a launcher activity.
func (c *Client) List(ctx context.Context, deviceID string) ([]App, error) {
	res, err := c.Adb.Shell(ctx, deviceID, "cmd package query-activities -a android.intent.action.MAIN -c android.intent.category.LAUNCHER")
	if err != nil {
		return nil, err
	}
	return parseLauncherActivities(string(res.Stdout)), nil
}

var launcherRE = regexp.MustCompile(`([A-Za-z0-9_.]+)/([A-Za-z0-9_.\$]+)`)

func parseLauncherActivities(out string) []App {
	seen := make(map[string]struct{})
	var apps []App
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, "ActivityInfo") && !strings.Contains(line, "name=") && !strings.Contains(line, "/") {
			continue
		}
		m := launcherRE.FindAllStringSubmatch(line, -1)
		for _, mm := range m {
			pkg := mm[1]
			act := mm[2]
			key := pkg + "/" + act
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			apps = append(apps, App{Package: pkg, Activity: act})
		}
	}
	return apps
}

// Install pushes an APK and installs it. When the android CLI is available
// and apkPath is on the host, prefers `android run --apks=...` (no rebuild).
func (c *Client) Install(ctx context.Context, deviceID, apkPath string) (string, error) {
	if c.AndroidCLI != nil && c.AndroidCLI.Available() {
		args := []string{"run", "--apks=" + apkPath}
		if deviceID != "" {
			args = append(args, "--device="+deviceID)
		}
		if res, err := c.AndroidCLI.Run(ctx, args...); err == nil {
			return strings.TrimSpace(string(res.Stdout)), nil
		}
		// Fall through to plain adb on android-CLI failure.
	}
	res, err := c.Adb.Run(ctx, deviceID, "install", "-r", apkPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// Uninstall removes a package.
func (c *Client) Uninstall(ctx context.Context, deviceID, pkg string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	_, err := c.Adb.Run(ctx, deviceID, "uninstall", pkg)
	return err
}

var localeRE = regexp.MustCompile(`^[A-Za-z0-9, \-]+$`)

// Launch starts an app's main launcher activity. If locale is non-empty,
// `cmd locale set-app-locales` is invoked first.
func (c *Client) Launch(ctx context.Context, deviceID, pkg, locale string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	if locale != "" {
		if !localeRE.MatchString(locale) {
			return fmt.Errorf("invalid locale %q", locale)
		}
		if _, err := c.Adb.ShellArgv(ctx, deviceID, "cmd", "locale", "set-app-locales", pkg, "--locales", locale); err != nil {
			return fmt.Errorf("setting locale: %w", err)
		}
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
	if err != nil {
		return err
	}
	combined := string(res.Stdout) + string(res.Stderr)
	if strings.Contains(combined, "No activities found to run") {
		return fmt.Errorf("package %s has no launcher activity", pkg)
	}
	return nil
}

// Terminate force-stops the given package.
func (c *Client) Terminate(ctx context.Context, deviceID, pkg string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "am", "force-stop", pkg)
	return err
}

// Kill soft-kills the package via `am kill`. Unlike Terminate, the package's
// services may be restarted by the system (relevant for START_STICKY) and
// future broadcasts are still delivered. Use this to simulate task swipe
// without the heavier semantics of force-stop.
func (c *Client) Kill(ctx context.Context, deviceID, pkg string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "am", "kill", pkg)
	return err
}

// ClearData wipes a package's user data.
func (c *Client) ClearData(ctx context.Context, deviceID, pkg string) error {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return err
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "pm", "clear", pkg)
	if err != nil {
		return err
	}
	if !strings.Contains(string(res.Stdout), "Success") {
		return errors.New(strings.TrimSpace(string(res.Stdout) + string(res.Stderr)))
	}
	return nil
}
