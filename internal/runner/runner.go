// Package runner is the only place in this codebase that spawns subprocesses.
//
// All adb / android / shell invocations go through a Runner so that timeouts,
// output caps, and structured errors are uniform.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultTimeout  = 30 * time.Second
	DefaultMaxBytes = 8 * 1024 * 1024
)

// Cmd describes a single subprocess invocation.
type Cmd struct {
	Bin      string
	Args     []string
	Stdin    io.Reader
	Timeout  time.Duration
	MaxBytes int64
	Env      []string
}

// Result holds the captured outputs of a finished subprocess.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
}

// ExecError is returned when a subprocess exits non-zero, times out, or
// otherwise fails to complete cleanly. It preserves the command line and a
// trimmed stderr tail so callers can build actionable error messages.
type ExecError struct {
	Bin      string
	Args     []string
	ExitCode int
	Stderr   string
	Cause    error
	TimedOut bool
}

func (e *ExecError) Error() string {
	cmd := e.Bin
	if len(e.Args) > 0 {
		cmd = e.Bin + " " + strings.Join(e.Args, " ")
	}
	switch {
	case e.TimedOut:
		return fmt.Sprintf("%s: timed out", cmd)
	case e.Stderr != "":
		return fmt.Sprintf("%s: exit %d: %s", cmd, e.ExitCode, strings.TrimSpace(e.Stderr))
	case e.Cause != nil:
		return fmt.Sprintf("%s: %v", cmd, e.Cause)
	default:
		return fmt.Sprintf("%s: exit %d", cmd, e.ExitCode)
	}
}

func (e *ExecError) Unwrap() error { return e.Cause }

// Runner executes commands. The zero value is unusable; use New.
type Runner struct {
	timeout  time.Duration
	maxBytes int64
}

// New constructs a Runner with the given defaults; pass 0 for stdlib defaults.
func New(timeout time.Duration, maxBytes int64) *Runner {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	return &Runner{timeout: timeout, maxBytes: maxBytes}
}

// Run executes c synchronously, returning captured stdout/stderr.
func (r *Runner) Run(ctx context.Context, c Cmd) (Result, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = r.timeout
	}
	maxBytes := c.MaxBytes
	if maxBytes <= 0 {
		maxBytes = r.maxBytes
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, c.Bin, c.Args...)
	if c.Stdin != nil {
		cmd.Stdin = c.Stdin
	}
	if c.Env != nil {
		cmd.Env = c.Env
	}

	stdout := &cappedBuffer{max: maxBytes}
	stderr := &cappedBuffer{max: maxBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	res := Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: cmd.ProcessState.ExitCode(),
		Duration: elapsed,
	}

	if err != nil {
		exErr := &ExecError{
			Bin:      c.Bin,
			Args:     c.Args,
			ExitCode: res.ExitCode,
			Stderr:   string(stderr.Bytes()),
			Cause:    err,
		}
		if errors.Is(cctx.Err(), context.DeadlineExceeded) {
			exErr.TimedOut = true
		}
		return res, exErr
	}
	return res, nil
}

// Stream starts the command and returns a handle so the caller can manage
// the process lifecycle (e.g. send signals to flush a screenrecord). Stdout
// is exposed as a ReadCloser; stderr is captured to an in-memory buffer
// reachable via the returned StreamHandle.
type StreamHandle struct {
	Cmd    *exec.Cmd
	Stdout io.ReadCloser
	stderr *cappedBuffer
	cancel context.CancelFunc
}

// Stderr returns whatever stderr has been accumulated so far.
func (h *StreamHandle) Stderr() []byte { return h.stderr.Bytes() }

// Cancel terminates the underlying process and releases the timeout context.
func (h *StreamHandle) Cancel() { h.cancel() }

// Stream launches a long-running command. Caller is responsible for waiting
// or cancelling.
func (r *Runner) Stream(ctx context.Context, c Cmd) (*StreamHandle, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		// Streamed processes get a generous ceiling; callers should still
		// cancel explicitly when done.
		timeout = 24 * time.Hour
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)

	cmd := exec.CommandContext(cctx, c.Bin, c.Args...)
	if c.Stdin != nil {
		cmd.Stdin = c.Stdin
	}
	if c.Env != nil {
		cmd.Env = c.Env
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, &ExecError{Bin: c.Bin, Args: c.Args, Cause: err}
	}
	stderr := &cappedBuffer{max: r.maxBytes}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, &ExecError{Bin: c.Bin, Args: c.Args, Cause: err}
	}

	return &StreamHandle{
		Cmd:    cmd,
		Stdout: stdout,
		stderr: stderr,
		cancel: cancel,
	}, nil
}

// cappedBuffer drops writes once the limit is reached.
type cappedBuffer struct {
	buf bytes.Buffer
	max int64
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	remaining := c.max - int64(c.buf.Len())
	if remaining <= 0 {
		return len(p), nil
	}
	if int64(len(p)) > remaining {
		c.buf.Write(p[:remaining])
		return len(p), nil
	}
	c.buf.Write(p)
	return len(p), nil
}

func (c *cappedBuffer) Bytes() []byte { return c.buf.Bytes() }
