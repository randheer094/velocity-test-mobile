package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// LocationClient sets a mock GPS coordinate.
type LocationClient struct {
	Adb *adb.Client
}

// NewLocationClient constructs a LocationClient.
func NewLocationClient(a *adb.Client) *LocationClient { return &LocationClient{Adb: a} }

// SetResult describes which path was used.
type SetResult struct {
	Mode  string `json:"mode"` // "emulator" | "device"
	Notes string `json:"notes,omitempty"`
}

// Set attempts to inject the supplied coordinates. On emulators it uses
// `adb emu geo fix`; on physical devices it cannot truly mock a location
// without a developer-options mock-location provider, so it returns a
// description of what the caller still needs to do.
func (l *LocationClient) Set(ctx context.Context, deviceID string, lat, lon float64, alt *float64) (SetResult, error) {
	if lat < -90 || lat > 90 {
		return SetResult{}, fmt.Errorf("lat %v out of range", lat)
	}
	if lon < -180 || lon > 180 {
		return SetResult{}, fmt.Errorf("lon %v out of range", lon)
	}
	args := []string{"emu", "geo", "fix", trimFloat(lon), trimFloat(lat)}
	if alt != nil {
		args = append(args, trimFloat(*alt))
	}
	res, err := l.Adb.Run(ctx, deviceID, args...)
	if err == nil {
		out := strings.TrimSpace(string(res.Stdout) + string(res.Stderr))
		if !strings.Contains(strings.ToLower(out), "error") {
			return SetResult{Mode: "emulator"}, nil
		}
	}
	return SetResult{
		Mode:  "device",
		Notes: "physical devices require a mock-location provider app and Developer Options → Select mock location app set to it; this server cannot inject GPS without that provider in place",
	}, nil
}

func trimFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

var _ = adb.QuoteForShell
