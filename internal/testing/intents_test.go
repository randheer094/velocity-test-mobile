package testing

import "testing"

func TestParseIntents(t *testing.T) {
	lines := []string{
		"05-02 01:00:00.000  1234  5678 I ActivityManager: START u0 {act=android.intent.action.VIEW dat=https://example.com/foo cat=android.intent.category.BROWSABLE pkg=com.android.chrome} from uid 10090",
		"05-02 01:00:01.000  1234  5678 I ActivityManager: START u0 {act=android.intent.action.MAIN cat=android.intent.category.LAUNCHER cmp=com.example.app/.MainActivity} from uid 10090",
		"unrelated log line",
		"05-02 01:00:02.000  1234  5678 I ActivityManager: START u0 {act=android.intent.action.SEND dat=mailto:user@host.com pkg=com.google.android.gm}",
	}
	intents := parseIntents(lines, "")
	if len(intents) != 3 {
		t.Fatalf("got %d intents, want 3", len(intents))
	}
	if intents[0].Action != "android.intent.action.VIEW" || intents[0].Data != "https://example.com/foo" {
		t.Errorf("first: %+v", intents[0])
	}
	if intents[1].Class != "com.example.app/.MainActivity" {
		t.Errorf("second class: %q", intents[1].Class)
	}
	if intents[2].Data != "mailto:user@host.com" {
		t.Errorf("third data: %q", intents[2].Data)
	}

	// Filter by package
	filtered := parseIntents(lines, "com.android.chrome")
	if len(filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(filtered))
	}
	if filtered[0].Package != "com.android.chrome" {
		t.Errorf("filtered pkg: %q", filtered[0].Package)
	}
}

func TestIntentMatches(t *testing.T) {
	i := CapturedIntent{Action: "android.intent.action.VIEW", Data: "https://example.com/x", Package: "com.foo"}
	if !intentMatches(i, IntentMatcher{Action: "android.intent.action.VIEW"}) {
		t.Error("action match")
	}
	if !intentMatches(i, IntentMatcher{DataContains: "example.com"}) {
		t.Error("data contains")
	}
	if intentMatches(i, IntentMatcher{Action: "VIEW"}) {
		t.Error("partial actions should not match")
	}
}
