package system

import (
	"context"
	"errors"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
	"github.com/randheer094/velocity-test-mobile/internal/runner"
)

// ShellClient wraps `adb shell` for the explicit `shell_exec` MCP tool.
//
// Most operations should use a typed wrapper (service_get_state,
// notification_list, etc.). This is the documented backstop for one-off
// introspection (new dumpsys services, debug-only setprop, etc.).
type ShellClient struct {
	Adb *adb.Client
}

// NewShellClient constructs a ShellClient.
func NewShellClient(a *adb.Client) *ShellClient { return &ShellClient{Adb: a} }

// ShellResult is the surface returned by `shell_exec`.
type ShellResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// Exec runs `adb shell <command>`, forwarding `command` verbatim to the
// device shell. A non-zero exit code is surfaced through ExitCode rather
// than as a Go error — `shell_exec` callers usually want to inspect both
// streams unconditionally.
func (c *ShellClient) Exec(ctx context.Context, deviceID, command string, timeout time.Duration) (ShellResult, error) {
	if command == "" {
		return ShellResult{}, errors.New("shell_exec: command is empty")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	res, err := c.Adb.Shell(ctx, deviceID, command)
	if err != nil {
		var ex *runner.ExecError
		if errors.As(err, &ex) {
			return ShellResult{Stdout: string(res.Stdout), Stderr: ex.Stderr, ExitCode: ex.ExitCode}, nil
		}
		return ShellResult{}, err
	}
	return ShellResult{Stdout: string(res.Stdout), Stderr: string(res.Stderr), ExitCode: res.ExitCode}, nil
}
