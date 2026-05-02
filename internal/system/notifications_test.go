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
