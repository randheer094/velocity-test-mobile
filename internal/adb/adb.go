// Package adb is a thin typed wrapper around the `adb` binary.
package adb

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/runner"
)

// Client invokes the adb binary located on PATH.
type Client struct {
	bin    string
	runner *runner.Runner
}

// ErrAdbMissing is returned when the adb binary cannot be located.
var ErrAdbMissing = errors.New("adb not found on PATH; install Android platform-tools and ensure adb is on PATH")

// New locates adb and returns a Client. The binary is resolved once.
func New(r *runner.Runner) (*Client, error) {
	bin, err := exec.LookPath("adb")
	if err != nil {
		return nil, ErrAdbMissing
	}
	return &Client{bin: bin, runner: r}, nil
}

// Bin returns the resolved adb path; useful for diagnostics.
func (c *Client) Bin() string { return c.bin }

func (c *Client) baseArgs(deviceID string, args ...string) []string {
	if deviceID == "" {
		return args
	}
	out := make([]string, 0, len(args)+2)
	out = append(out, "-s", deviceID)
	out = append(out, args...)
	return out
}

// Run executes `adb [-s deviceID] args...` and returns captured output.
func (c *Client) Run(ctx context.Context, deviceID string, args ...string) (runner.Result, error) {
	return c.runner.Run(ctx, runner.Cmd{Bin: c.bin, Args: c.baseArgs(deviceID, args...)})
}

// Shell is a convenience for `adb [-s deviceID] shell <command>`.
func (c *Client) Shell(ctx context.Context, deviceID, command string) (runner.Result, error) {
	return c.Run(ctx, deviceID, "shell", command)
}

// ShellArgv runs `adb shell` with separately-supplied argv tokens. Use this
// when arguments may contain spaces or shell metacharacters; adb forwards
// each arg as-is to the device shell.
func (c *Client) ShellArgv(ctx context.Context, deviceID string, argv ...string) (runner.Result, error) {
	args := append([]string{"shell"}, argv...)
	return c.Run(ctx, deviceID, args...)
}

// ExecOut runs `adb exec-out <argv...>` returning raw bytes (binary mode).
// Used for `screencap -p`, `uiautomator dump /dev/tty`.
func (c *Client) ExecOut(ctx context.Context, deviceID string, argv ...string) ([]byte, error) {
	args := append([]string{"exec-out"}, argv...)
	res, err := c.runner.Run(ctx, runner.Cmd{
		Bin:      c.bin,
		Args:     c.baseArgs(deviceID, args...),
		MaxBytes: 32 * 1024 * 1024, // raw frames can exceed default
	})
	if err != nil {
		return nil, err
	}
	return res.Stdout, nil
}

// Stream starts an adb command and returns a handle for the caller to manage.
func (c *Client) Stream(ctx context.Context, deviceID string, args ...string) (*runner.StreamHandle, error) {
	return c.runner.Stream(ctx, runner.Cmd{Bin: c.bin, Args: c.baseArgs(deviceID, args...)})
}

// KeyCombination dispatches `input keycombination <code1> <code2> [...]`,
// available on API 31+. On older devices it returns an error so callers
// can fall back to issuing keys individually.
func (c *Client) KeyCombination(ctx context.Context, deviceID string, codes ...int) error {
	if len(codes) < 2 {
		return errors.New("keycombination requires at least two keycodes")
	}
	args := []string{"input", "keycombination"}
	for _, code := range codes {
		args = append(args, fmt.Sprintf("%d", code))
	}
	res, err := c.ShellArgv(ctx, deviceID, args...)
	if err != nil {
		return err
	}
	combined := string(res.Stdout) + string(res.Stderr)
	if strings.Contains(strings.ToLower(combined), "unknown command") ||
		strings.Contains(strings.ToLower(combined), "error: invalid") {
		return errors.New("device does not support `input keycombination` (Android < 12)")
	}
	return nil
}

// QuoteForShell wraps a string in single quotes for safe inclusion in an
// `adb shell <command>` invocation. Internal single quotes are escaped.
func QuoteForShell(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// MustQuotePackage validates a package name. It is intentionally strict:
// alphanumerics, dot, underscore. Anything else is rejected to prevent
// shell injection in `cmd package`, `pm`, `am` subcommands.
func MustQuotePackage(pkg string) (string, error) {
	for _, r := range pkg {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_':
		default:
			return "", fmt.Errorf("invalid package name %q", pkg)
		}
	}
	if pkg == "" {
		return "", errors.New("package name is empty")
	}
	return pkg, nil
}
