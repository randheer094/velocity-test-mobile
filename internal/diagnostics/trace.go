package diagnostics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// TraceClient handles atrace and perfetto captures.
type TraceClient struct {
	Adb *adb.Client
}

// NewTraceClient builds a TraceClient.
func NewTraceClient(a *adb.Client) *TraceClient { return &TraceClient{Adb: a} }

// allowedAtraceCategories restricts the user-supplied category list to a
// known-safe set, preventing shell injection in `atrace -c`.
var allowedAtraceCategories = map[string]struct{}{
	"gfx": {}, "view": {}, "input": {}, "wm": {}, "am": {}, "sm": {},
	"audio": {}, "video": {}, "camera": {}, "hal": {}, "app": {},
	"res": {}, "dalvik": {}, "rs": {}, "bionic": {}, "power": {},
	"pm": {}, "ss": {}, "database": {}, "network": {}, "binder_driver": {},
	"binder_lock": {}, "freq": {}, "idle": {}, "disk": {}, "mmc": {}, "load": {},
	"sync": {}, "workq": {}, "memreclaim": {}, "regulators": {}, "thermal": {},
	"sched": {}, "i2c": {}, "irq": {}, "ext4": {},
}

// AtraceCapture runs an atrace session and pulls the binary trace to host.
// duration is in seconds (1..300). categories defaults to gfx,view,input,wm,am.
func (t *TraceClient) AtraceCapture(ctx context.Context, deviceID string, duration int, categories []string, hostOutput string) (string, error) {
	if duration <= 0 {
		duration = 5
	}
	if duration > 300 {
		return "", fmt.Errorf("duration must be <= 300 seconds")
	}
	if len(categories) == 0 {
		categories = []string{"gfx", "view", "input", "wm", "am"}
	}
	for _, c := range categories {
		if _, ok := allowedAtraceCategories[c]; !ok {
			return "", fmt.Errorf("category %q not in allowlist", c)
		}
	}
	if hostOutput == "" {
		hostOutput = filepath.Join(os.TempDir(), fmt.Sprintf("atrace-%d.trace", time.Now().Unix()))
	}
	if !strings.HasSuffix(hostOutput, ".trace") {
		return "", fmt.Errorf("output path should end in .trace")
	}
	devicePath := fmt.Sprintf("/data/local/tmp/atrace-%d.trace", time.Now().UnixNano())

	argv := append([]string{"atrace", "--async_start", "-b", "32768", "-c"}, categories...)
	if _, err := t.Adb.ShellArgv(ctx, deviceID, argv...); err != nil {
		return "", fmt.Errorf("atrace start: %w", err)
	}
	select {
	case <-ctx.Done():
		_, _ = t.Adb.ShellArgv(context.Background(), deviceID, "atrace", "--async_stop", "-z", "-o", devicePath)
		return "", ctx.Err()
	case <-time.After(time.Duration(duration) * time.Second):
	}
	if _, err := t.Adb.ShellArgv(ctx, deviceID, "atrace", "--async_stop", "-z", "-o", devicePath); err != nil {
		return "", fmt.Errorf("atrace stop: %w", err)
	}
	if _, err := t.Adb.Run(ctx, deviceID, "pull", devicePath, hostOutput); err != nil {
		return "", fmt.Errorf("pull trace: %w", err)
	}
	_, _ = t.Adb.ShellArgv(ctx, deviceID, "rm", devicePath)
	return hostOutput, nil
}

// PerfettoCapture runs perfetto with the bundled default config (or the
// supplied configPath) for `duration` seconds, then pulls the .pftrace.
//
// Note: perfetto on consumer devices typically requires API 28+ and tracebox
// availability. If perfetto is unavailable, callers should prefer atrace.
func (t *TraceClient) PerfettoCapture(ctx context.Context, deviceID string, duration int, configContent string, hostOutput string) (string, error) {
	if duration <= 0 {
		duration = 10
	}
	if duration > 300 {
		return "", fmt.Errorf("duration must be <= 300 seconds")
	}
	if configContent == "" {
		configContent = defaultPerfettoConfig(duration)
	}
	if hostOutput == "" {
		hostOutput = filepath.Join(os.TempDir(), fmt.Sprintf("perfetto-%d.pftrace", time.Now().Unix()))
	}
	devicePath := fmt.Sprintf("/data/misc/perfetto-traces/perfetto-%d.pftrace", time.Now().UnixNano())

	cmd := fmt.Sprintf("printf '%%s' %s | perfetto --txt -c - -o %s",
		adb.QuoteForShell(configContent),
		adb.QuoteForShell(devicePath),
	)
	if _, err := t.Adb.Shell(ctx, deviceID, cmd); err != nil {
		return "", fmt.Errorf("perfetto run: %w", err)
	}
	if _, err := t.Adb.Run(ctx, deviceID, "pull", devicePath, hostOutput); err != nil {
		return "", fmt.Errorf("pull perfetto trace: %w", err)
	}
	_, _ = t.Adb.ShellArgv(ctx, deviceID, "rm", devicePath)
	return hostOutput, nil
}

func defaultPerfettoConfig(durationSec int) string {
	durMs := durationSec * 1000
	return fmt.Sprintf(`buffers: { size_kb: 63488 fill_policy: DISCARD }
data_sources: {
  config {
    name: "linux.ftrace"
    ftrace_config {
      ftrace_events: "sched/sched_switch"
      ftrace_events: "power/cpu_frequency"
      ftrace_events: "power/cpu_idle"
      ftrace_events: "sched/sched_process_exit"
      ftrace_events: "sched/sched_process_free"
      ftrace_events: "task/task_newtask"
      ftrace_events: "task/task_rename"
      atrace_categories: "gfx"
      atrace_categories: "view"
      atrace_categories: "input"
      atrace_categories: "wm"
      atrace_categories: "am"
    }
  }
}
data_sources: { config { name: "linux.process_stats" } }
duration_ms: %d
`, durMs)
}
