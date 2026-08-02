package system

import "testing"

func TestBuildAnimationsGetCmd(t *testing.T) {
	got := buildAnimationsGetCmd()
	want := "settings get global window_animation_scale && settings get global transition_animation_scale && settings get global animator_duration_scale"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildAnimationsSetCmd(t *testing.T) {
	got := buildAnimationsSetCmd("0")
	want := "settings put global window_animation_scale 0 && settings put global transition_animation_scale 0 && settings put global animator_duration_scale 0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAnimationsSet_RejectsOutOfRangeScale(t *testing.T) {
	a := NewAnimationsClient(nil)
	if err := a.Set(nil, "dev", -1); err == nil {
		t.Error("expected error for scale < 0")
	}
	if err := a.Set(nil, "dev", 11); err == nil {
		t.Error("expected error for scale > 10")
	}
}
