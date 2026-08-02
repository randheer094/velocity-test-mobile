package device

import (
	"context"
	"testing"
	"time"
)

func TestParseAdbDevices(t *testing.T) {
	out := `List of devices attached
emulator-5554          device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa transport_id:1
ABCDEF1234             unauthorized
192.168.1.5:5555       offline product:redfin model:Pixel_5 device:redfin transport_id:3
`
	got := parseAdbDevices(out)
	if len(got) != 3 {
		t.Fatalf("got %d devices, want 3", len(got))
	}
	if got[0].Serial != "emulator-5554" || got[0].State != "device" || got[0].Model != "sdk_gphone64_x86_64" {
		t.Errorf("first: %+v", got[0])
	}
	if got[1].State != "unauthorized" {
		t.Errorf("second state: %q", got[1].State)
	}
	if got[2].Transport != "3" {
		t.Errorf("third transport: %q", got[2].Transport)
	}
}

func TestParseGetprop(t *testing.T) {
	out := `[ro.product.model]: [Pixel 5]
[ro.product.brand]: [google]
[ro.build.version.sdk]: [33]
[malformed line without brackets]
[ro.build.fingerprint]: [google/redfin/redfin:13/foo/bar:user/release-keys]
`
	m := parseGetprop(out)
	if m["ro.product.model"] != "Pixel 5" {
		t.Errorf("model: %q", m["ro.product.model"])
	}
	if m["ro.build.version.sdk"] != "33" {
		t.Errorf("sdk: %q", m["ro.build.version.sdk"])
	}
	if m["ro.build.fingerprint"] != "google/redfin/redfin:13/foo/bar:user/release-keys" {
		t.Errorf("fingerprint: %q", m["ro.build.fingerprint"])
	}
}

func newTestResolver(fetch func(ctx context.Context) ([]Device, error), ttl time.Duration) *Resolver {
	r := NewResolver(nil, nil, time.Second, ttl)
	r.fetch = fetch
	return r
}

func TestList_CachesWithinTTL(t *testing.T) {
	calls := 0
	r := newTestResolver(func(ctx context.Context) ([]Device, error) {
		calls++
		return []Device{{Serial: "emulator-5554", State: "device"}}, nil
	}, time.Minute)

	if _, err := r.List(context.Background()); err != nil {
		t.Fatalf("List #1: %v", err)
	}
	if _, err := r.List(context.Background()); err != nil {
		t.Fatalf("List #2: %v", err)
	}
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
}

func TestList_RefetchesAfterTTL(t *testing.T) {
	calls := 0
	now := time.Now()
	r := newTestResolver(func(ctx context.Context) ([]Device, error) {
		calls++
		return []Device{{Serial: "emulator-5554", State: "device"}}, nil
	}, 5*time.Millisecond)
	r.now = func() time.Time { return now }

	if _, err := r.List(context.Background()); err != nil {
		t.Fatalf("List #1: %v", err)
	}
	now = now.Add(10 * time.Millisecond)
	if _, err := r.List(context.Background()); err != nil {
		t.Fatalf("List #2: %v", err)
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2", calls)
	}
}

func TestForceList_BypassesCache(t *testing.T) {
	calls := 0
	r := newTestResolver(func(ctx context.Context) ([]Device, error) {
		calls++
		return []Device{{Serial: "emulator-5554", State: "device"}}, nil
	}, time.Minute)

	if _, err := r.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := r.ForceList(context.Background()); err != nil {
		t.Fatalf("ForceList: %v", err)
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2", calls)
	}
}

func TestResolve_RefreshesOnUnknownDeviceID(t *testing.T) {
	calls := 0
	r := newTestResolver(func(ctx context.Context) ([]Device, error) {
		calls++
		return []Device{{Serial: "emulator-5554", State: "device"}}, nil
	}, time.Minute)

	if _, err := r.Resolve(context.Background(), "emulator-5556"); err == nil {
		t.Fatal("expected error for unknown device")
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2 (initial + refresh-on-miss)", calls)
	}
}

func TestResolve_FindsKnownDeviceWithoutRefetch(t *testing.T) {
	calls := 0
	r := newTestResolver(func(ctx context.Context) ([]Device, error) {
		calls++
		return []Device{{Serial: "emulator-5554", State: "device"}}, nil
	}, time.Minute)

	d, err := r.Resolve(context.Background(), "emulator-5554")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if d.Serial != "emulator-5554" {
		t.Errorf("serial = %q", d.Serial)
	}
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
}

func TestList_DoesNotCacheErrors(t *testing.T) {
	calls := 0
	r := newTestResolver(func(ctx context.Context) ([]Device, error) {
		calls++
		return nil, context.DeadlineExceeded
	}, time.Minute)

	if _, err := r.List(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if _, err := r.List(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2 (errors must not be cached)", calls)
	}
}
