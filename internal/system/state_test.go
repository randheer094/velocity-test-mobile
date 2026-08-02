package system

import "testing"

func TestBuildBatterySetCmd(t *testing.T) {
	ops := []batteryOp{{"level", "50"}, {"status", "2"}}
	got := buildBatterySetCmd(ops)
	want := "dumpsys battery set level 50; dumpsys battery set status 2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetBattery_ValidatesLevel(t *testing.T) {
	s := NewStateClient(nil)
	if err := s.SetBattery(nil, "dev", BatteryState{Level: 101}); err == nil {
		t.Error("expected error for level > 100")
	}
	if err := s.SetBattery(nil, "dev", BatteryState{Level: -5}); err == nil {
		t.Error("expected error for level < -1")
	}
}

func TestSetBattery_ValidatesStatus(t *testing.T) {
	s := NewStateClient(nil)
	if err := s.SetBattery(nil, "dev", BatteryState{Level: -1, Status: 6}); err == nil {
		t.Error("expected error for status out of [1,5]")
	}
}

func TestSetBattery_ValidatesPlugField(t *testing.T) {
	s := NewStateClient(nil)
	if err := s.SetBattery(nil, "dev", BatteryState{Level: -1, AC: 3}); err == nil {
		t.Error("expected error for ac not in {1,2}")
	}
}

func TestSetBattery_RequiresAtLeastOneField(t *testing.T) {
	s := NewStateClient(nil)
	if err := s.SetBattery(nil, "dev", BatteryState{Level: -1}); err == nil {
		t.Error("expected error when no fields are set")
	}
}

func TestSetFontScale_ValidatesRange(t *testing.T) {
	s := NewStateClient(nil)
	if err := s.SetFontScale(nil, "dev", 0.1); err == nil {
		t.Error("expected error for scale below 0.5")
	}
	if err := s.SetFontScale(nil, "dev", 3.0); err == nil {
		t.Error("expected error for scale above 2.5")
	}
}

func TestSetDarkMode_ValidatesMode(t *testing.T) {
	s := NewStateClient(nil)
	if err := s.SetDarkMode(nil, "dev", "sometimes"); err == nil {
		t.Error("expected error for an invalid mode")
	}
}
