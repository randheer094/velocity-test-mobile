package diagnostics

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
	"github.com/randheer094/velocity-test-mobile/internal/runner"
)

// RecordClient drives `adb shell screenrecord` for video evidence. One
// active recording per device; concurrent starts on the same device are
// rejected.
type RecordClient struct {
	Adb *adb.Client
	mu  sync.Mutex
	// sessions[deviceID] = active session; absent when no recording is in flight.
	sessions map[string]*recordSession
}

// NewRecordClient constructs a RecordClient.
func NewRecordClient(a *adb.Client) *RecordClient {
	return &RecordClient{Adb: a, sessions: make(map[string]*recordSession)}
}

type recordSession struct {
	Stream     *runner.StreamHandle
	RemoteFile string
	LocalFile  string
	StartedAt  time.Time
}

// RecordOptions covers the screenrecord knobs that matter for tests.
type RecordOptions struct {
	// LocalFile is the host path the recording is pulled to on stop.
	LocalFile string
	// MaxDurationS caps the recording at the device level (screenrecord's own
	// `--time-limit`, max 180 in older Android, 1800+ in 11+).
	MaxDurationS int
	// SizeWidth / SizeHeight optionally downscale the recording.
	SizeWidth  int
	SizeHeight int
	// BitRate in bits per second; 0 uses the device default.
	BitRate int
}

// Start begins a recording. Returns the remote file path; the local file
// is not populated until Stop completes the pull.
func (c *RecordClient) Start(ctx context.Context, deviceID string, opts RecordOptions) (remote string, err error) {
	if strings.TrimSpace(opts.LocalFile) == "" {
		return "", fmt.Errorf("local_file is required")
	}
	abs, err := filepath.Abs(opts.LocalFile)
	if err != nil {
		return "", fmt.Errorf("local_file: %w", err)
	}

	// Insert a placeholder before releasing the lock so a concurrent Start on
	// the same device fails the busy-check before the adb.Stream call below.
	c.mu.Lock()
	if _, busy := c.sessions[deviceID]; busy {
		c.mu.Unlock()
		return "", fmt.Errorf("a recording is already in progress on this device — call screen_record_stop first")
	}
	c.sessions[deviceID] = nil
	c.mu.Unlock()

	remote = fmt.Sprintf("/sdcard/velocity-record-%d.mp4", time.Now().UnixNano())
	args := []string{"shell", "screenrecord"}
	if opts.MaxDurationS > 0 {
		args = append(args, "--time-limit", fmt.Sprintf("%d", opts.MaxDurationS))
	}
	if opts.SizeWidth > 0 && opts.SizeHeight > 0 {
		args = append(args, "--size", fmt.Sprintf("%dx%d", opts.SizeWidth, opts.SizeHeight))
	}
	if opts.BitRate > 0 {
		args = append(args, "--bit-rate", fmt.Sprintf("%d", opts.BitRate))
	}
	args = append(args, remote)

	stream, err := c.Adb.Stream(ctx, deviceID, args...)
	if err != nil {
		c.mu.Lock()
		delete(c.sessions, deviceID)
		c.mu.Unlock()
		return "", fmt.Errorf("start screenrecord: %w", err)
	}
	// Drain stdout so the pipe doesn't fill and block the device.
	go func() { _, _ = io.Copy(io.Discard, stream.Stdout) }()

	c.mu.Lock()
	c.sessions[deviceID] = &recordSession{
		Stream:     stream,
		RemoteFile: remote,
		LocalFile:  abs,
		StartedAt:  time.Now(),
	}
	c.mu.Unlock()
	return remote, nil
}

// StopResult reports the outcome of a Stop call.
type StopResult struct {
	LocalFile  string `json:"local_file"`
	RemoteFile string `json:"remote_file"`
	DurationMs int64  `json:"duration_ms"`
}

// Stop terminates the active recording, pulls it to LocalFile, and removes
// the device-side temporary.
func (c *RecordClient) Stop(ctx context.Context, deviceID string) (StopResult, error) {
	c.mu.Lock()
	sess, ok := c.sessions[deviceID]
	if !ok {
		c.mu.Unlock()
		return StopResult{}, fmt.Errorf("no recording in progress on this device")
	}
	delete(c.sessions, deviceID)
	c.mu.Unlock()

	// Cancel the stream to send SIGINT — screenrecord on the device flushes
	// the file when it sees the parent shell go away.
	sess.Stream.Cancel()
	// Give the device a brief moment to finalize the MP4 container.
	select {
	case <-time.After(750 * time.Millisecond):
	case <-ctx.Done():
	}

	// Pull file off the device.
	if _, err := c.Adb.Run(ctx, deviceID, "pull", sess.RemoteFile, sess.LocalFile); err != nil {
		return StopResult{}, fmt.Errorf("pull recording: %w", err)
	}
	// Remove the device-side temp; failures here are non-fatal.
	_, _ = c.Adb.ShellArgv(ctx, deviceID, "rm", "-f", sess.RemoteFile)

	return StopResult{
		LocalFile:  sess.LocalFile,
		RemoteFile: sess.RemoteFile,
		DurationMs: time.Since(sess.StartedAt).Milliseconds(),
	}, nil
}

// PullFile copies a remote file off the device via `adb pull`.
//
// `app_data_read` covers the run-as path for debuggable apps; PullFile is
// for everything else: /sdcard/, /data/local/tmp/, captured artefacts.
func (c *RecordClient) PullFile(ctx context.Context, deviceID, remote, local string) error {
	if strings.TrimSpace(remote) == "" || strings.TrimSpace(local) == "" {
		return fmt.Errorf("both remote and local paths are required")
	}
	abs, err := filepath.Abs(local)
	if err != nil {
		return fmt.Errorf("local: %w", err)
	}
	if _, err := c.Adb.Run(ctx, deviceID, "pull", remote, abs); err != nil {
		return fmt.Errorf("adb pull: %w", err)
	}
	return nil
}
