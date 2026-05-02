// Package device handles Android device discovery, default-device resolution,
// and a small set of read-only property queries.
package device

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
)

// Device is a minimal view of a connected Android device.
type Device struct {
	Serial    string `json:"serial"`
	State     string `json:"state"` // device, offline, unauthorized, ...
	Transport string `json:"transport,omitempty"`
	Model     string `json:"model,omitempty"`
	Product   string `json:"product,omitempty"`
	Source    string `json:"source"` // "adb" | "android"
}

// Resolver bundles the clients needed for discovery & resolution.
type Resolver struct {
	Adb         *adb.Client
	AndroidCLI  *androidcli.Client
	listTimeout time.Duration
}

// NewResolver constructs a Resolver. Pass 0 for default timeout (5s).
func NewResolver(a *adb.Client, c *androidcli.Client, listTimeout time.Duration) *Resolver {
	if listTimeout == 0 {
		listTimeout = 5 * time.Second
	}
	return &Resolver{Adb: a, AndroidCLI: c, listTimeout: listTimeout}
}

// List enumerates devices visible to adb. The android CLI's `emulator list`
// only catalogs known AVDs (running or not), which is informational; live
// devices come from `adb devices -l`.
func (r *Resolver) List(ctx context.Context) ([]Device, error) {
	cctx, cancel := context.WithTimeout(ctx, r.listTimeout)
	defer cancel()

	res, err := r.Adb.Run(cctx, "", "devices", "-l")
	if err != nil {
		return nil, err
	}
	devices := parseAdbDevices(string(res.Stdout))
	return devices, nil
}

func parseAdbDevices(out string) []Device {
	var devices []Device
	sc := bufio.NewScanner(strings.NewReader(out))
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			// Header: "List of devices attached"
			first = false
			if strings.HasPrefix(line, "List of devices") {
				continue
			}
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		d := Device{Serial: fields[0], State: fields[1], Source: "adb"}
		for _, kv := range fields[2:] {
			parts := strings.SplitN(kv, ":", 2)
			if len(parts) != 2 {
				continue
			}
			switch parts[0] {
			case "model":
				d.Model = parts[1]
			case "product":
				d.Product = parts[1]
			case "transport_id":
				d.Transport = parts[1]
			}
		}
		devices = append(devices, d)
	}
	return devices
}

// Resolve picks a device given an optional preference. If id is empty and
// exactly one connected device is in the "device" state, that one is used;
// otherwise an actionable error is returned.
func (r *Resolver) Resolve(ctx context.Context, id string) (Device, error) {
	devices, err := r.List(ctx)
	if err != nil {
		return Device{}, err
	}
	if id != "" {
		for _, d := range devices {
			if d.Serial == id {
				if d.State != "device" {
					return Device{}, fmt.Errorf("device %s is in state %q (expected \"device\")", id, d.State)
				}
				return d, nil
			}
		}
		return Device{}, fmt.Errorf("device %q not found among %s", id, summarize(devices))
	}
	var ready []Device
	for _, d := range devices {
		if d.State == "device" {
			ready = append(ready, d)
		}
	}
	switch len(ready) {
	case 0:
		if len(devices) == 0 {
			return Device{}, errors.New("no devices connected; run `adb devices` to verify")
		}
		return Device{}, fmt.Errorf("no usable devices; states: %s", summarize(devices))
	case 1:
		return ready[0], nil
	default:
		return Device{}, fmt.Errorf("multiple devices connected — pass `device`: %s", summarize(ready))
	}
}

func summarize(ds []Device) string {
	if len(ds) == 0 {
		return "<none>"
	}
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = fmt.Sprintf("%s(%s)", d.Serial, d.State)
	}
	return strings.Join(parts, ", ")
}

// Props is a curated subset of `getprop` values returned by GetProps.
type Props struct {
	Serial       string `json:"serial"`
	Model        string `json:"model"`
	Brand        string `json:"brand"`
	Manufacturer string `json:"manufacturer"`
	Device       string `json:"device"`
	SDKLevel     string `json:"sdkLevel"`
	Release      string `json:"release"`
	Fingerprint  string `json:"fingerprint"`
	ABIList      string `json:"abiList"`
}

// GetProps fetches a curated set of read-only system properties.
func (r *Resolver) GetProps(ctx context.Context, deviceID string) (Props, error) {
	res, err := r.Adb.Shell(ctx, deviceID, "getprop")
	if err != nil {
		return Props{}, err
	}
	all := parseGetprop(string(res.Stdout))
	return Props{
		Serial:       all["ro.serialno"],
		Model:        all["ro.product.model"],
		Brand:        all["ro.product.brand"],
		Manufacturer: all["ro.product.manufacturer"],
		Device:       all["ro.product.device"],
		SDKLevel:     all["ro.build.version.sdk"],
		Release:      all["ro.build.version.release"],
		Fingerprint:  all["ro.build.fingerprint"],
		ABIList:      all["ro.product.cpu.abilist"],
	}, nil
}

func parseGetprop(out string) map[string]string {
	m := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		// Format: [key]: [value]
		if !strings.HasPrefix(line, "[") {
			continue
		}
		i := strings.Index(line, "]: [")
		if i < 0 {
			continue
		}
		key := line[1:i]
		rest := line[i+4:]
		j := strings.LastIndex(rest, "]")
		if j < 0 {
			continue
		}
		m[key] = rest[:j]
	}
	return m
}
