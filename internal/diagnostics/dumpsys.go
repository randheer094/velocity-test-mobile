package diagnostics

import (
	"bufio"
	"context"
	"strconv"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// DumpsysClient runs and parses select `dumpsys` outputs.
type DumpsysClient struct {
	Adb *adb.Client
}

// NewDumpsysClient builds a DumpsysClient.
func NewDumpsysClient(a *adb.Client) *DumpsysClient { return &DumpsysClient{Adb: a} }

// MemInfo holds parsed `dumpsys meminfo` numbers (KB).
type MemInfo struct {
	Package    string `json:"package"`
	TotalPSS   int    `json:"totalPssKb"`
	NativeHeap int    `json:"nativeHeapKb"`
	DalvikHeap int    `json:"dalvikHeapKb"`
	Code       int    `json:"codeKb"`
	Stack      int    `json:"stackKb"`
	Graphics   int    `json:"graphicsKb"`
	Private    int    `json:"privateKb"`
	System     int    `json:"systemKb"`
	JavaHeap   int    `json:"javaHeapKb"`
}

// MemInfo parses `dumpsys meminfo <pkg>`.
func (d *DumpsysClient) MemInfo(ctx context.Context, deviceID, pkg string) (MemInfo, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return MemInfo{}, err
	}
	res, err := d.Adb.ShellArgv(ctx, deviceID, "dumpsys", "meminfo", pkg)
	if err != nil {
		return MemInfo{}, err
	}
	return parseMemInfo(pkg, string(res.Stdout)), nil
}

func parseMemInfo(pkg, out string) MemInfo {
	m := MemInfo{Package: pkg}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		first := strings.ToLower(fields[0])
		switch {
		case strings.HasPrefix(line, "TOTAL "):
			m.TotalPSS = atoi(fields[1])
		case first == "native" && len(fields) >= 3 && strings.ToLower(fields[1]) == "heap":
			m.NativeHeap = atoi(fields[2])
		case first == "dalvik" && len(fields) >= 3 && strings.ToLower(fields[1]) == "heap":
			m.DalvikHeap = atoi(fields[2])
		case first == "code":
			m.Code = atoi(fields[1])
		case first == "stack":
			m.Stack = atoi(fields[1])
		case first == "graphics":
			m.Graphics = atoi(fields[1])
		case first == "private" && len(fields) >= 3 && strings.ToLower(fields[1]) == "other":
			m.Private = atoi(fields[2])
		case first == "system":
			m.System = atoi(fields[1])
		case strings.HasPrefix(line, "Java Heap:"):
			if len(fields) >= 3 {
				m.JavaHeap = atoi(fields[2])
			}
		}
	}
	return m
}

// GfxInfo holds parsed `dumpsys gfxinfo <pkg>` jank stats.
type GfxInfo struct {
	Package      string  `json:"package"`
	TotalFrames  int     `json:"totalFrames"`
	JankyFrames  int     `json:"jankyFrames"`
	JankPercent  float64 `json:"jankPercent"`
	Percentile50 int     `json:"percentile50Ms"`
	Percentile90 int     `json:"percentile90Ms"`
	Percentile95 int     `json:"percentile95Ms"`
	Percentile99 int     `json:"percentile99Ms"`
}

// GfxInfo parses `dumpsys gfxinfo <pkg>`.
func (d *DumpsysClient) GfxInfo(ctx context.Context, deviceID, pkg string) (GfxInfo, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return GfxInfo{}, err
	}
	res, err := d.Adb.ShellArgv(ctx, deviceID, "dumpsys", "gfxinfo", pkg)
	if err != nil {
		return GfxInfo{}, err
	}
	return parseGfxInfo(pkg, string(res.Stdout)), nil
}

func parseGfxInfo(pkg, out string) GfxInfo {
	g := GfxInfo{Package: pkg}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "Total frames rendered:"):
			g.TotalFrames = atoi(lastField(line))
		case strings.HasPrefix(line, "Janky frames:"):
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				g.JankyFrames = atoi(fields[2])
			}
			if i := strings.Index(line, "("); i >= 0 {
				if j := strings.Index(line[i:], "%"); j > 0 {
					raw := strings.TrimSpace(line[i+1 : i+j])
					if v, err := strconv.ParseFloat(raw, 64); err == nil {
						g.JankPercent = v
					}
				}
			}
		case strings.HasPrefix(line, "50th percentile:"):
			g.Percentile50 = atoi(lastField(line))
		case strings.HasPrefix(line, "90th percentile:"):
			g.Percentile90 = atoi(lastField(line))
		case strings.HasPrefix(line, "95th percentile:"):
			g.Percentile95 = atoi(lastField(line))
		case strings.HasPrefix(line, "99th percentile:"):
			g.Percentile99 = atoi(lastField(line))
		}
	}
	return g
}

// Battery is a key/value snapshot from `dumpsys battery`.
type Battery struct {
	Level       int    `json:"level"`
	Scale       int    `json:"scale"`
	Status      int    `json:"status"`
	Health      int    `json:"health"`
	Plugged     int    `json:"plugged"`
	Voltage     int    `json:"voltage"`
	Temperature int    `json:"temperature"`
	Technology  string `json:"technology"`
	ACPowered   bool   `json:"acPowered"`
	USBPowered  bool   `json:"usbPowered"`
}

// BatteryInfo parses `dumpsys battery`.
func (d *DumpsysClient) BatteryInfo(ctx context.Context, deviceID string) (Battery, error) {
	res, err := d.Adb.ShellArgv(ctx, deviceID, "dumpsys", "battery")
	if err != nil {
		return Battery{}, err
	}
	return parseBattery(string(res.Stdout)), nil
}

func parseBattery(out string) Battery {
	b := Battery{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "level:"):
			b.Level = atoi(strings.TrimSpace(strings.TrimPrefix(line, "level:")))
		case strings.HasPrefix(line, "scale:"):
			b.Scale = atoi(strings.TrimSpace(strings.TrimPrefix(line, "scale:")))
		case strings.HasPrefix(line, "status:"):
			b.Status = atoi(strings.TrimSpace(strings.TrimPrefix(line, "status:")))
		case strings.HasPrefix(line, "health:"):
			b.Health = atoi(strings.TrimSpace(strings.TrimPrefix(line, "health:")))
		case strings.HasPrefix(line, "plugged:"):
			b.Plugged = atoi(strings.TrimSpace(strings.TrimPrefix(line, "plugged:")))
		case strings.HasPrefix(line, "voltage:"):
			b.Voltage = atoi(strings.TrimSpace(strings.TrimPrefix(line, "voltage:")))
		case strings.HasPrefix(line, "temperature:"):
			b.Temperature = atoi(strings.TrimSpace(strings.TrimPrefix(line, "temperature:")))
		case strings.HasPrefix(line, "technology:"):
			b.Technology = strings.TrimSpace(strings.TrimPrefix(line, "technology:"))
		case strings.HasPrefix(line, "AC powered:"):
			b.ACPowered = strings.Contains(line, "true")
		case strings.HasPrefix(line, "USB powered:"):
			b.USBPowered = strings.Contains(line, "true")
		}
	}
	return b
}

// ActivityInfo summarizes `dumpsys activity activities`.
type ActivityInfo struct {
	FocusedActivity string   `json:"focusedActivity"`
	TopResumed      string   `json:"topResumed"`
	Recents         []string `json:"recents"`
}

// Activity returns a short summary of the activity stack. If pkg is non-empty,
// the recents list is filtered to that package.
func (d *DumpsysClient) Activity(ctx context.Context, deviceID, pkg string) (ActivityInfo, error) {
	args := []string{"dumpsys", "activity", "activities"}
	res, err := d.Adb.ShellArgv(ctx, deviceID, args...)
	if err != nil {
		return ActivityInfo{}, err
	}
	return parseActivity(string(res.Stdout), pkg), nil
}

func parseActivity(out, pkgFilter string) ActivityInfo {
	a := ActivityInfo{}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "mFocusedActivity:") || strings.HasPrefix(line, "mResumedActivity:") || strings.HasPrefix(line, "topResumedActivity="):
			a.FocusedActivity = trimAfter(line)
		case strings.HasPrefix(line, "mTopResumedActivity:"):
			a.TopResumed = trimAfter(line)
		case strings.HasPrefix(line, "Hist  #") || strings.HasPrefix(line, "* Hist #"):
			if pkgFilter != "" && !strings.Contains(line, pkgFilter) {
				continue
			}
			a.Recents = append(a.Recents, line)
		}
	}
	return a
}

func trimAfter(line string) string {
	if i := strings.Index(line, ":"); i >= 0 {
		return strings.TrimSpace(line[i+1:])
	}
	return line
}

func atoi(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func lastField(line string) string {
	fields := strings.Fields(strings.TrimSuffix(line, "ms"))
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	return strings.TrimSuffix(last, "ms")
}
