// Package diagnostics covers logcat, dumpsys, and tracing.
package diagnostics

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// LogClient handles logcat tailing & clearing.
type LogClient struct {
	Adb *adb.Client
}

// NewLogClient builds a LogClient.
func NewLogClient(a *adb.Client) *LogClient { return &LogClient{Adb: a} }

// LogOptions controls Tail.
type LogOptions struct {
	Package  string // resolves to a PID via `pidof`
	Tag      string // tag filter spec, e.g. "MyTag"
	Priority string // V D I W E F
	MaxLines int    // default 1000
	Since    string // logcat -T <since>; empty for full dump
	Regex    string // post-filter regex
}

// Tail returns up to MaxLines log lines using `logcat -d` (dump mode).
func (l *LogClient) Tail(ctx context.Context, deviceID string, opts LogOptions) ([]string, error) {
	if opts.MaxLines <= 0 {
		opts.MaxLines = 1000
	}
	args := []string{"logcat", "-d", "-v", "time"}
	if opts.Since != "" {
		args = append(args, "-T", opts.Since)
	}
	if opts.Package != "" {
		if _, err := adb.MustQuotePackage(opts.Package); err != nil {
			return nil, err
		}
		pid, err := l.pidOf(ctx, deviceID, opts.Package)
		if err != nil {
			return nil, err
		}
		if pid > 0 {
			args = append(args, "--pid", fmt.Sprintf("%d", pid))
		}
	}
	// Tag/priority filterspec must come last, after a "*:S" silencer if a tag is given.
	if opts.Tag != "" {
		pri := opts.Priority
		if pri == "" {
			pri = "V"
		}
		args = append(args, opts.Tag+":"+pri, "*:S")
	} else if opts.Priority != "" {
		args = append(args, "*:"+opts.Priority)
	}

	res, err := l.Adb.ShellArgv(ctx, deviceID, args...)
	if err != nil {
		return nil, err
	}
	lines := splitLines(res.Stdout)
	if opts.Regex != "" {
		re, rerr := regexp.Compile(opts.Regex)
		if rerr != nil {
			return nil, fmt.Errorf("invalid regex: %w", rerr)
		}
		filtered := lines[:0]
		for _, ln := range lines {
			if re.MatchString(ln) {
				filtered = append(filtered, ln)
			}
		}
		lines = filtered
	}
	if len(lines) > opts.MaxLines {
		lines = lines[len(lines)-opts.MaxLines:]
	}
	return lines, nil
}

// Clear empties the log buffers.
func (l *LogClient) Clear(ctx context.Context, deviceID string) error {
	_, err := l.Adb.ShellArgv(ctx, deviceID, "logcat", "-c")
	return err
}

func splitLines(b []byte) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// pidOf returns the first PID matching the given package, or 0 if none.
func (l *LogClient) pidOf(ctx context.Context, deviceID, pkg string) (int, error) {
	res, err := l.Adb.ShellArgv(ctx, deviceID, "pidof", pkg)
	if err != nil {
		// `pidof` may return non-zero for "no process"; treat that as 0.
		return 0, nil
	}
	out := strings.TrimSpace(string(res.Stdout))
	if out == "" {
		return 0, nil
	}
	first := strings.Fields(out)[0]
	var pid int
	if _, err := fmt.Sscanf(first, "%d", &pid); err != nil {
		return 0, nil
	}
	return pid, nil
}
