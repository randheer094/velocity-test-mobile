package diagnostics

import "testing"

func TestRecordStart_RequiresLocalFile(t *testing.T) {
	c := NewRecordClient(nil)
	if _, err := c.Start(nil, "dev", RecordOptions{}); err == nil {
		t.Error("expected error when local_file is empty")
	}
}

func TestRecordStart_RejectsConcurrentRecording(t *testing.T) {
	c := NewRecordClient(nil)
	c.sessions["dev"] = &recordSession{} // simulate an in-flight recording
	if _, err := c.Start(nil, "dev", RecordOptions{LocalFile: "/tmp/out.mp4"}); err == nil {
		t.Error("expected error for a device that already has a recording in progress")
	}
}

func TestRecordStop_NoRecordingInProgress(t *testing.T) {
	c := NewRecordClient(nil)
	if _, err := c.Stop(nil, "dev"); err == nil {
		t.Error("expected error when no recording is in progress")
	}
}

func TestRecordStop_StillInitializing(t *testing.T) {
	c := NewRecordClient(nil)
	c.sessions["dev"] = nil // placeholder inserted by Start before Stream succeeds
	if _, err := c.Stop(nil, "dev"); err == nil {
		t.Error("expected error while the recording is still initializing")
	}
}

func TestPullFile_RequiresBothPaths(t *testing.T) {
	c := NewRecordClient(nil)
	if err := c.PullFile(nil, "dev", "", "/tmp/out"); err == nil {
		t.Error("expected error for an empty remote path")
	}
	if err := c.PullFile(nil, "dev", "/sdcard/out", ""); err == nil {
		t.Error("expected error for an empty local path")
	}
}
