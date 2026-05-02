package system

import "testing"

const dumpsysServiceFG = `
ACTIVITY MANAGER SERVICES (dumpsys activity services)
  User 0 active services:
  * ServiceRecord{abcd1234 u0 dev.randheer094.dev/.MockLocationService}
    intent={act=dev.randheer094.dev.location.action.START cmp=dev.randheer094.dev/.MockLocationService}
    packageName=dev.randheer094.dev
    processName=dev.randheer094.dev
    isForeground=true
    foregroundId=42
    foregroundNoti=Notification(channel=mock_location_channel pri=0 flags=0x40)
    startId=1
    createTime=-1m6s ago
`

const dumpsysServiceStopped = `
ACTIVITY MANAGER SERVICES (dumpsys activity services)
  (no running services)
`

func TestParseServiceState_Foreground(t *testing.T) {
	st := parseServiceState(dumpsysServiceFG, "dev.randheer094.dev", "")
	if !st.Running {
		t.Fatalf("expected Running=true, got %+v", st)
	}
	if !st.Foreground {
		t.Errorf("expected Foreground=true, got %+v", st)
	}
	if st.StartID == nil || *st.StartID != 1 {
		t.Errorf("StartID = %v, want 1", st.StartID)
	}
	if st.NotificationID == nil || *st.NotificationID != 42 {
		t.Errorf("NotificationID = %v, want 42 (0x2a)", st.NotificationID)
	}
	if st.LastIntentAction != "dev.randheer094.dev.location.action.START" {
		t.Errorf("LastIntentAction = %q", st.LastIntentAction)
	}
}

func TestParseServiceState_NotRunning(t *testing.T) {
	st := parseServiceState(dumpsysServiceStopped, "dev.randheer094.dev", "")
	if st.Running {
		t.Fatalf("expected Running=false, got %+v", st)
	}
}

func TestParseServiceState_OtherPackageIgnored(t *testing.T) {
	st := parseServiceState(dumpsysServiceFG, "com.other.app", "")
	if st.Running {
		t.Fatalf("expected service for other package not to match, got %+v", st)
	}
}

func TestParseServiceState_ComponentFilter(t *testing.T) {
	st := parseServiceState(dumpsysServiceFG, "dev.randheer094.dev", "dev.randheer094.dev.OtherService")
	if st.Running {
		t.Fatalf("expected non-matching component to fall through, got %+v", st)
	}
	st = parseServiceState(dumpsysServiceFG, "dev.randheer094.dev", "dev.randheer094.dev.MockLocationService")
	if !st.Running {
		t.Fatalf("expected component match, got %+v", st)
	}
}
