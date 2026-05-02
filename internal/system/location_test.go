package system

import "testing"

const dumpsysLocationGPS = `
LOCATION MANAGER STATE (dumpsys location)
  Location Settings:
  ...
  Last Known Locations:
    gps: last location=Location[gps 59.331707,18.075729 hAcc=15.0 et=+1d2h time=1714600000000 alt=42.0]
    network: last location=Location[network 0.000000,0.000000 hAcc=0.0]
`

const dumpsysLocationFusedOnly = `
  Last Known Locations:
    fused: last location=Location[fused 37.774929,-122.419418 hAcc=8.5 time=1714600000000]
`

func TestParseLastLocation_GPS(t *testing.T) {
	fix := parseLastLocation(dumpsysLocationGPS, "gps")
	if fix == nil {
		t.Fatal("expected gps fix, got nil")
	}
	if fix.Provider != "gps" {
		t.Errorf("Provider = %q", fix.Provider)
	}
	if fix.Lat != 59.331707 {
		t.Errorf("Lat = %v", fix.Lat)
	}
	if fix.Lng != 18.075729 {
		t.Errorf("Lng = %v", fix.Lng)
	}
	if fix.Accuracy == nil || *fix.Accuracy != 15.0 {
		t.Errorf("Accuracy = %v", fix.Accuracy)
	}
}

func TestParseLastLocation_NoProviderPicksFirst(t *testing.T) {
	fix := parseLastLocation(dumpsysLocationFusedOnly, "")
	if fix == nil || fix.Provider != "fused" {
		t.Fatalf("expected fused fix, got %+v", fix)
	}
}

func TestParseLastLocation_None(t *testing.T) {
	if got := parseLastLocation("no location info here", ""); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestParseLastLocation_ProviderMismatch(t *testing.T) {
	if got := parseLastLocation(dumpsysLocationGPS, "fused"); got != nil {
		t.Errorf("expected nil for unmatched provider, got %+v", got)
	}
}
