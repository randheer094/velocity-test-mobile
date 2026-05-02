package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRun_Success(t *testing.T) {
	r := New(2*time.Second, 0)
	res, err := r.Run(context.Background(), Cmd{Bin: "sh", Args: []string{"-c", "printf hello"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(res.Stdout); got != "hello" {
		t.Fatalf("stdout = %q, want %q", got, "hello")
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code = %d", res.ExitCode)
	}
}

func TestRun_NonZeroExit(t *testing.T) {
	r := New(2*time.Second, 0)
	_, err := r.Run(context.Background(), Cmd{Bin: "sh", Args: []string{"-c", "echo boom 1>&2; exit 7"}})
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *ExecError
	if !errors.As(err, &ee) {
		t.Fatalf("error type = %T, want *ExecError", err)
	}
	if ee.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", ee.ExitCode)
	}
	if !strings.Contains(ee.Stderr, "boom") {
		t.Fatalf("stderr = %q", ee.Stderr)
	}
}

func TestRun_Timeout(t *testing.T) {
	r := New(50*time.Millisecond, 0)
	_, err := r.Run(context.Background(), Cmd{Bin: "sh", Args: []string{"-c", "sleep 5"}})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	var ee *ExecError
	if !errors.As(err, &ee) || !ee.TimedOut {
		t.Fatalf("expected TimedOut ExecError, got %v", err)
	}
}

func TestRun_OutputCap(t *testing.T) {
	r := New(2*time.Second, 8)
	res, err := r.Run(context.Background(), Cmd{Bin: "sh", Args: []string{"-c", "printf '%s' aaaaaaaaaaaaaaaa"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Stdout) != 8 {
		t.Fatalf("captured %d bytes, want 8", len(res.Stdout))
	}
}
