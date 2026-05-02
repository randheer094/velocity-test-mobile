package system

import "testing"

const dumpsysActivities = `
ACTIVITY MANAGER ACTIVITIES (dumpsys activity activities)
  Display #0 (default) (HOME):
    Stack #1: type=standard mode=fullscreen
      Task{aabb 0:dev.randheer094.dev/.MainActivity bounds=...}
        ActivityRecord{1234 u0 dev.randheer094.dev/.MainActivity t77}
        topResumedActivity=ActivityRecord{1234 u0 dev.randheer094.dev/.MainActivity t77}
`

const dumpsysActivitiesFQN = `
  topResumedActivity=ActivityRecord{cafe u0 com.android.settings/.SubSettings t12}
`

func TestParseTopActivity_Relative(t *testing.T) {
	t1, err := parseTopActivity(dumpsysActivities)
	if err != nil {
		t.Fatal(err)
	}
	if t1 == nil {
		t.Fatal("expected non-nil top")
	}
	if t1.Package != "dev.randheer094.dev" {
		t.Errorf("Package = %q", t1.Package)
	}
	if t1.Activity != "dev.randheer094.dev.MainActivity" {
		t.Errorf("Activity = %q (expected absolute form)", t1.Activity)
	}
	if t1.TaskID == nil || *t1.TaskID != 77 {
		t.Errorf("TaskID = %v", t1.TaskID)
	}
}

func TestParseTopActivity_FQN(t *testing.T) {
	t1, _ := parseTopActivity(dumpsysActivitiesFQN)
	if t1 == nil || t1.Package != "com.android.settings" {
		t.Fatalf("got %+v", t1)
	}
}

func TestParseTopActivity_None(t *testing.T) {
	t1, _ := parseTopActivity("no activity in this dump")
	if t1 != nil {
		t.Errorf("expected nil, got %+v", t1)
	}
}

func TestMatchesActivity(t *testing.T) {
	cases := []struct {
		actual, pkg, expected string
		want                  bool
	}{
		{"com.example.MainActivity", "com.example", ".MainActivity", true},
		{"com.example.MainActivity", "com.example", "com.example.MainActivity", true},
		{"com.example.MainActivity", "com.example", "MainActivity", true},
		{"com.example.OtherActivity", "com.example", ".MainActivity", false},
		{"com.example.MainActivity", "com.example", "", true},
	}
	for _, c := range cases {
		if got := matchesActivity(c.actual, c.pkg, c.expected); got != c.want {
			t.Errorf("matchesActivity(%q, %q, %q) = %v, want %v", c.actual, c.pkg, c.expected, got, c.want)
		}
	}
}
