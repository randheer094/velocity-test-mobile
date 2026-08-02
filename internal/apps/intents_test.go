package apps

import "testing"

func newTestClient() *Client { return &Client{} }

func TestSendIntent_InvalidMode(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{Mode: "bogus"}); err == nil {
		t.Error("expected error for an invalid mode")
	}
}

func TestSendIntent_InvalidAction(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{Action: "android.intent.action.VIEW; rm -rf /"}); err == nil {
		t.Error("expected error for an action containing shell metacharacters")
	}
}

func TestSendIntent_InvalidCategory(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{Category: "android.intent.category.DEFAULT`whoami`"}); err == nil {
		t.Error("expected error for a category containing shell metacharacters")
	}
}

func TestSendIntent_InvalidDataURI(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{Data: "://not a valid uri"}); err == nil {
		t.Error("expected error for an invalid data URI")
	}
}

func TestSendIntent_InvalidPackage(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{Package: "com.example; reboot"}); err == nil {
		t.Error("expected error for an invalid package name")
	}
}

func TestSendIntent_InvalidFlag(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{Flags: []string{"not-a-flag"}}); err == nil {
		t.Error("expected error for a malformed flag")
	}
}

func TestSendIntent_InvalidExtraKey(t *testing.T) {
	c := newTestClient()
	if err := c.SendIntent(nil, "dev", Intent{StringEx: map[string]string{"bad key!": "v"}}); err == nil {
		t.Error("expected error for an invalid extra key")
	}
}

func TestSendIntent_ValidFlagFormats(t *testing.T) {
	// Both decimal and 0xHEX flag forms must pass validation; the only way
	// to observe that without a fake adb client is that the call proceeds
	// past validation to the (nil-Adb) dispatch, which panics rather than
	// returning a validation error — so we just assert the regex accepts
	// both forms directly.
	for _, f := range []string{"0x10", "16", "0"} {
		if !flagRE.MatchString(f) {
			t.Errorf("flagRE should accept %q", f)
		}
	}
	for _, f := range []string{"", "abc", "0X10", "0xZZ", "-1"} {
		if flagRE.MatchString(f) {
			t.Errorf("flagRE should reject %q", f)
		}
	}
}
