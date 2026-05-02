package system

import "testing"

const dumpsysNotifications = `
Current Notification Manager state:

  Notification List:
  NotificationRecord(0xabcd: pkg=dev.randheer094.dev user=UserHandle{0} id=42 tag=null importance=DEFAULT key=...)
    icon=Icon(android.R.drawable.ic_menu_mylocation)
    mPriority=0
    mWhen=1714600000000
    flags=0x62
    mChannel=NotificationChannel{id=mock_location_channel, name=Mock Location}
    extras={
      android.title=String (Mock location active)
      android.text=String (Sending Stockholm preset)
    }
    mOngoing=true

  NotificationRecord(0xabce: pkg=com.android.systemui id=1 tag=note)
    flags=0x0
    mChannel=NotificationChannel{id=other_channel}
    extras={
      android.title=String (Hello)
    }
    mOngoing=false
`

// dumpsysNotificationsApi34 is captured verbatim from
// `adb shell dumpsys notification --noredact` on a Pixel emulator running
// API 34. The format differs from older APIs in two ways the parser must
// handle: (1) the channel id is embedded inline in the `NotificationRecord(`
// header as `Notification(channel=<id> ...)` rather than on a separate
// `mChannel=NotificationChannel{id=...}` line, and (2) the structured
// channel block uses `effectiveNotificationChannel=NotificationChannel{mId='<id>', ...}`.
const dumpsysNotificationsApi34 = `
Current Notification Manager state:

  Notification List:
    NotificationRecord(0x01a0bcb9: pkg=dev.randheer094.dev.location.debug user=UserHandle{0} id=1 tag=null importance=2 key=0|dev.randheer094.dev.location.debug|1|null|10230: Notification(channel=mock_location_channel shortcut=null contentView=null vibrate=null sound=null defaults=0x0 flags=0x62 color=0x00000000 groupKey=silent vis=SECRET))
      uid=10230 userId=0
      flags=0x62
      pri=-1
      notification=
            when=1777723466492
            extras={
                android.title=String (Mocking Location)
                android.text=String (Latitude: 59.3383223, Longitude: 18.0549621)
            }
      effectiveNotificationChannel=NotificationChannel{mId='mock_location_channel', mName=Mock Location, mImportance=2}
`

func TestParseNotifications_OurApp(t *testing.T) {
	all := parseNotifications(dumpsysNotifications)
	if len(all) < 2 {
		t.Fatalf("expected >=2 notifications, got %d (%+v)", len(all), all)
	}
	var our *Notification
	for i := range all {
		if all[i].Package == "dev.randheer094.dev" {
			our = &all[i]
			break
		}
	}
	if our == nil {
		t.Fatalf("missing dev.randheer094.dev notification: %+v", all)
	}
	if our.Channel != "mock_location_channel" {
		t.Errorf("Channel = %q", our.Channel)
	}
	if our.Title != "Mock location active" {
		t.Errorf("Title = %q", our.Title)
	}
	if our.Text != "Sending Stockholm preset" {
		t.Errorf("Text = %q", our.Text)
	}
	if !our.Ongoing {
		t.Errorf("expected Ongoing=true")
	}
	if our.ID != 42 {
		t.Errorf("ID = %d", our.ID)
	}
}

func TestParseNotifications_Api34InlineChannel(t *testing.T) {
	all := parseNotifications(dumpsysNotificationsApi34)
	if len(all) != 1 {
		t.Fatalf("expected 1 notification, got %d (%+v)", len(all), all)
	}
	n := all[0]
	if n.Package != "dev.randheer094.dev.location.debug" {
		t.Errorf("Package = %q", n.Package)
	}
	if n.Channel != "mock_location_channel" {
		t.Errorf("Channel = %q (parser must extract from inline `Notification(channel=...)`)", n.Channel)
	}
	if n.ID != 1 {
		t.Errorf("ID = %d", n.ID)
	}
	if n.Title != "Mocking Location" {
		t.Errorf("Title = %q", n.Title)
	}
	if !n.Ongoing {
		t.Errorf("expected Ongoing=true (flags=0x62 has FLAG_ONGOING_EVENT bit)")
	}
}
