package device

import "testing"

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
