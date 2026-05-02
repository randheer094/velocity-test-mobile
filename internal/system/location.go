package system

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// LocationClient parses `dumpsys location` for last-known fixes.
type LocationClient struct {
	Adb *adb.Client
}

// NewLocationClient constructs a LocationClient.
func NewLocationClient(a *adb.Client) *LocationClient { return &LocationClient{Adb: a} }

// LocationFix is a last-known location reported by LocationManager.
type LocationFix struct {
	Provider string   `json:"provider"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Accuracy *float64 `json:"accuracy_m,omitempty"`
	TimeMs   *int64   `json:"time_ms,omitempty"`
}

var validProviders = map[string]struct{}{
	"gps":     {},
	"network": {},
	"passive": {},
	"fused":   {},
}

// GetLastKnown returns the most recent location reported to the LocationManager.
// When `provider` is empty, the first reported provider in the dump wins;
// otherwise only the matching provider is considered. Returns nil when no
// fix is reported.
func (c *LocationClient) GetLastKnown(ctx context.Context, deviceID, provider string) (*LocationFix, error) {
	if provider != "" {
		if _, ok := validProviders[provider]; !ok {
			return nil, fmt.Errorf("invalid provider %q (expected gps|network|passive|fused)", provider)
		}
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "dumpsys", "location")
	if err != nil {
		return nil, err
	}
	return parseLastLocation(string(res.Stdout), provider), nil
}

// parseLastLocation accepts both the legacy and modern `dumpsys location`
// shapes:
//
//	last location=Location[gps 37.7749,-122.4194 hAcc=5.0 et=+1d2h ...]
//	  last location: Location[gps 37.77,−122.41 hAcc=5.0 ...]
//	gps: last location=Location[gps 37.77,-122.41 ...]
//
// We scan line-by-line, attribute each `Location[<provider> ...]` to its
// nearest preceding "last location" header, and pick the requested provider.
var locationLineRE = regexp.MustCompile(`Location\[(\w+)\s+(-?\d+\.\d+)[,\s]+(-?\d+\.\d+)([^\]]*)\]`)

func parseLastLocation(out, want string) *LocationFix {
	// Restrict to lines that mention "last location" — there are many
	// transient Location[] dumps for active providers we don't care about.
	var candidates []string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(strings.ToLower(line), "last location") {
			candidates = append(candidates, line)
		}
	}
	for _, line := range candidates {
		m := locationLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		prov := m[1]
		if want != "" && prov != want {
			continue
		}
		lat, errLat := strconv.ParseFloat(m[2], 64)
		lng, errLng := strconv.ParseFloat(m[3], 64)
		if errLat != nil || errLng != nil {
			continue
		}
		fix := &LocationFix{Provider: prov, Lat: lat, Lng: lng}
		extras := m[4]
		if v, ok := parseFloatField(extras, "hAcc="); ok {
			fix.Accuracy = &v
		}
		if v, ok := parseInt64Field(extras, "time="); ok {
			fix.TimeMs = &v
		}
		return fix
	}
	return nil
}

func parseFloatField(haystack, key string) (float64, bool) {
	idx := strings.Index(haystack, key)
	if idx < 0 {
		return 0, false
	}
	rest := haystack[idx+len(key):]
	end := strings.IndexAny(rest, " \t")
	if end > 0 {
		rest = rest[:end]
	}
	v, err := strconv.ParseFloat(rest, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseInt64Field(haystack, key string) (int64, bool) {
	idx := strings.Index(haystack, key)
	if idx < 0 {
		return 0, false
	}
	rest := haystack[idx+len(key):]
	end := strings.IndexAny(rest, " \t")
	if end > 0 {
		rest = rest[:end]
	}
	v, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
