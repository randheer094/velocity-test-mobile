package system

import (
	"context"
	"testing"
	"time"
)

func TestParseWMSize(t *testing.T) {
	cases := []struct {
		in   string
		w, h int
	}{
		{"Physical size: 1080x2400\n", 1080, 2400},
		{"Physical size: 1080x2400\nOverride size: 720x1600\n", 720, 1600},
		{"\n", 0, 0},
	}
	for _, c := range cases {
		w, h := parseWMSize(c.in)
		if w != c.w || h != c.h {
			t.Errorf("parseWMSize(%q) = %dx%d, want %dx%d", c.in, w, h, c.w, c.h)
		}
	}
}

func TestParseWMDensity(t *testing.T) {
	out := "Physical density: 420\nOverride density: 480\n"
	if d := parseWMDensity(out); d != 480 {
		t.Errorf("density = %d, want 480", d)
	}
}

func TestBuildSetOrientationCmd(t *testing.T) {
	got := buildSetOrientationCmd("1")
	want := "settings put system accelerometer_rotation 0 && settings put system user_rotation 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func newTestScreenClient(fetch func(ctx context.Context, deviceID string) (Size, error)) *ScreenClient {
	s := NewScreenClient(nil)
	s.fetch = fetch
	return s
}

func TestScreenGet_CachesWithinTTL(t *testing.T) {
	calls := 0
	s := newTestScreenClient(func(ctx context.Context, deviceID string) (Size, error) {
		calls++
		return Size{Width: 1080, Height: 2400, Density: 420}, nil
	})

	if _, err := s.Get(context.Background(), "dev"); err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	if _, err := s.Get(context.Background(), "dev"); err != nil {
		t.Fatalf("Get #2: %v", err)
	}
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
}

func TestScreenGet_RefetchesAfterTTL(t *testing.T) {
	calls := 0
	now := time.Now()
	s := newTestScreenClient(func(ctx context.Context, deviceID string) (Size, error) {
		calls++
		return Size{Width: 1080, Height: 2400}, nil
	})
	s.cacheTTL = 5 * time.Millisecond
	s.now = func() time.Time { return now }

	if _, err := s.Get(context.Background(), "dev"); err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	now = now.Add(10 * time.Millisecond)
	if _, err := s.Get(context.Background(), "dev"); err != nil {
		t.Fatalf("Get #2: %v", err)
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2", calls)
	}
}

func TestScreenGet_CachedPerDevice(t *testing.T) {
	calls := 0
	s := newTestScreenClient(func(ctx context.Context, deviceID string) (Size, error) {
		calls++
		return Size{Width: 1080, Height: 2400}, nil
	})

	if _, err := s.Get(context.Background(), "device-a"); err != nil {
		t.Fatalf("Get device-a: %v", err)
	}
	if _, err := s.Get(context.Background(), "device-b"); err != nil {
		t.Fatalf("Get device-b: %v", err)
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2 (distinct devices must not share a cache entry)", calls)
	}
}

func TestScreenGet_DoesNotCacheErrors(t *testing.T) {
	calls := 0
	s := newTestScreenClient(func(ctx context.Context, deviceID string) (Size, error) {
		calls++
		return Size{}, context.DeadlineExceeded
	})

	if _, err := s.Get(context.Background(), "dev"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := s.Get(context.Background(), "dev"); err == nil {
		t.Fatal("expected error")
	}
	if calls != 2 {
		t.Errorf("fetch called %d times, want 2 (errors must not be cached)", calls)
	}
}
