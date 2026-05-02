// Package androidcli wraps Google's `android` agent CLI.
//
// See https://developer.android.com/tools/agents/android-cli .
//
// Tools that depend on `android` should fall back to plain adb whenever
// possible; if there is no fallback, surface a clear error pointing the
// user at the install page.
package androidcli

import (
	"context"
	"errors"
	"os/exec"

	"github.com/randheer094/velocity-test-mobile/internal/runner"
)

// ErrNotInstalled is returned by capability-gated calls when the `android`
// binary is missing.
var ErrNotInstalled = errors.New("the `android` agent CLI is not installed; download it from https://developer.android.com/tools/agents/android-cli")

// Client wraps the `android` binary.
type Client struct {
	bin       string
	available bool
	runner    *runner.Runner
}

// New attempts to locate the `android` binary; absence is not an error.
func New(r *runner.Runner) *Client {
	bin, err := exec.LookPath("android")
	if err != nil {
		return &Client{runner: r}
	}
	return &Client{bin: bin, available: true, runner: r}
}

// Available reports whether the android CLI was found at startup.
func (c *Client) Available() bool { return c.available }

// Bin returns the resolved path or "" if not installed.
func (c *Client) Bin() string { return c.bin }

// Run invokes the android CLI; returns ErrNotInstalled if unavailable.
func (c *Client) Run(ctx context.Context, args ...string) (runner.Result, error) {
	if !c.available {
		return runner.Result{}, ErrNotInstalled
	}
	return c.runner.Run(ctx, runner.Cmd{Bin: c.bin, Args: args})
}
