package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
	"github.com/randheer094/velocity-mcp-mobile/internal/runner"
)

// Recorder manages live `screenrecord` sessions, one per device.
type Recorder struct {
	Adb *adb.Client

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	deviceID   string
	devicePath string
	hostPath   string
	startedAt  time.Time
	handle     *runner.StreamHandle
	cancelCtx  context.CancelFunc
	wg         sync.WaitGroup
	exitErr    error
	timeLimit  int
}

// NewRecorder constructs a Recorder.
func NewRecorder(a *adb.Client) *Recorder {
	return &Recorder{Adb: a, sessions: map[string]*session{}}
}

// StartOptions controls a recording session.
type StartOptions struct {
	TimeLimitSec int    // 0 = no limit (capped by `screenrecord`'s own 180s ceiling per chunk)
	BitrateMbps  int    // 0 = device default
	SizeWxH      string // e.g. "720x1280"; "" = native
	Output       string // host destination; "" = temp file
}

// StartResult is returned to callers.
type StartResult struct {
	HostPath   string `json:"hostPath"`
	DevicePath string `json:"devicePath"`
}

// Start begins a recording on the given device.
func (r *Recorder) Start(ctx context.Context, deviceID string, opts StartOptions) (StartResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[deviceID]; exists {
		return StartResult{}, fmt.Errorf("a recording is already active for device %q; stop it first", deviceID)
	}

	hostPath := opts.Output
	if hostPath == "" {
		hostPath = filepath.Join(os.TempDir(), fmt.Sprintf("screen-recording-%d.mp4", time.Now().Unix()))
	}
	if filepath.Ext(hostPath) != ".mp4" {
		return StartResult{}, fmt.Errorf("output path must end in .mp4")
	}
	devicePath := fmt.Sprintf("/sdcard/screen-record-%d.mp4", time.Now().UnixNano())

	args := []string{"shell", "screenrecord"}
	if opts.TimeLimitSec > 0 {
		args = append(args, "--time-limit", fmt.Sprintf("%d", opts.TimeLimitSec))
	}
	if opts.BitrateMbps > 0 {
		args = append(args, "--bit-rate", fmt.Sprintf("%d", opts.BitrateMbps*1_000_000))
	}
	if opts.SizeWxH != "" {
		args = append(args, "--size", opts.SizeWxH)
	}
	args = append(args, devicePath)

	streamCtx, cancel := context.WithCancel(context.Background())
	handle, err := r.Adb.Stream(streamCtx, deviceID, args...)
	if err != nil {
		cancel()
		return StartResult{}, err
	}

	sess := &session{
		deviceID:   deviceID,
		devicePath: devicePath,
		hostPath:   hostPath,
		startedAt:  time.Now(),
		handle:     handle,
		cancelCtx:  cancel,
		timeLimit:  opts.TimeLimitSec,
	}
	sess.wg.Add(1)
	go func() {
		defer sess.wg.Done()
		sess.exitErr = handle.Cmd.Wait()
	}()
	r.sessions[deviceID] = sess
	return StartResult{HostPath: hostPath, DevicePath: devicePath}, nil
}

// StopResult is returned by Stop.
type StopResult struct {
	HostPath    string  `json:"hostPath"`
	SizeMB      float64 `json:"sizeMB"`
	DurationSec float64 `json:"durationSec"`
}

// Stop terminates an in-progress recording, pulls it to the host, and
// removes the on-device file. Sends SIGINT first; falls back to KILL after
// 5 minutes (which should never trigger for a healthy screenrecord).
func (r *Recorder) Stop(ctx context.Context, deviceID string) (StopResult, error) {
	r.mu.Lock()
	sess, ok := r.sessions[deviceID]
	if !ok {
		r.mu.Unlock()
		return StopResult{}, fmt.Errorf("no recording active for device %q", deviceID)
	}
	delete(r.sessions, deviceID)
	r.mu.Unlock()

	// Send SIGINT so screenrecord flushes the file.
	if sess.handle.Cmd.Process != nil {
		_ = sess.handle.Cmd.Process.Signal(syscall.SIGINT)
	}

	done := make(chan struct{})
	go func() { sess.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Minute):
		_ = sess.handle.Cmd.Process.Kill()
		<-done
	}
	sess.cancelCtx()

	// Pull from device
	if _, err := r.Adb.Run(ctx, deviceID, "pull", sess.devicePath, sess.hostPath); err != nil {
		return StopResult{}, fmt.Errorf("pulling recording: %w", err)
	}
	// Best-effort cleanup
	_, _ = r.Adb.ShellArgv(ctx, deviceID, "rm", sess.devicePath)

	stat, err := os.Stat(sess.hostPath)
	if err != nil {
		return StopResult{}, err
	}
	return StopResult{
		HostPath:    sess.hostPath,
		SizeMB:      float64(stat.Size()) / (1024.0 * 1024.0),
		DurationSec: time.Since(sess.startedAt).Seconds(),
	}, nil
}

// IsActive reports whether the device currently has a recording.
func (r *Recorder) IsActive(deviceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.sessions[deviceID]
	return ok
}

// StopAll best-effort terminates every active recording (used at shutdown).
func (r *Recorder) StopAll(ctx context.Context) {
	r.mu.Lock()
	devices := make([]string, 0, len(r.sessions))
	for d := range r.sessions {
		devices = append(devices, d)
	}
	r.mu.Unlock()
	for _, d := range devices {
		_, _ = r.Stop(ctx, d)
	}
}

var _ = errors.New
