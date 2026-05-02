package diagnostics

import "testing"

func TestParseMemInfo(t *testing.T) {
	out := `Applications Memory Usage (in Kilobytes):
Uptime: 12345 Realtime: 67890

** MEMINFO in pid 1234 [com.example.app] **
                   Pss  Private  Private     Swap      Rss     Heap     Heap     Heap
                 Total    Dirty    Clean    Dirty    Total     Size    Alloc     Free
                ------   ------   ------   ------   ------   ------   ------   ------
  Native Heap     5120     5000        0        0    20000   24576    20000     4576
  Dalvik Heap     2048     2000        0        0    10000    8192     6000     2192
        Code      4096     3000     1000        0     8000        0        0        0
       Stack       512      500        0        0      512        0        0        0
    Graphics      1024     1000        0        0     1024        0        0        0
       Other      1234       12        0        0     1500        0        0        0
       System      300        0      300        0      300        0        0        0
       TOTAL    13434    11512     1300        0    41336

 Java Heap:    8048
 Native Heap:    5000
`
	m := parseMemInfo("com.example.app", out)
	if m.Package != "com.example.app" {
		t.Errorf("pkg: %q", m.Package)
	}
	if m.TotalPSS != 13434 {
		t.Errorf("TotalPSS: %d", m.TotalPSS)
	}
	if m.NativeHeap != 5120 {
		t.Errorf("NativeHeap: %d", m.NativeHeap)
	}
	if m.DalvikHeap != 2048 {
		t.Errorf("DalvikHeap: %d", m.DalvikHeap)
	}
}

func TestParseGfxInfo(t *testing.T) {
	out := `Stats since: 1234567ms

Total frames rendered: 500
Janky frames: 25 (5.00%)
50th percentile: 8ms
90th percentile: 14ms
95th percentile: 18ms
99th percentile: 32ms
`
	g := parseGfxInfo("com.example", out)
	if g.TotalFrames != 500 || g.JankyFrames != 25 {
		t.Errorf("frames=%d janky=%d", g.TotalFrames, g.JankyFrames)
	}
	if g.JankPercent != 5.0 {
		t.Errorf("jank pct: %v", g.JankPercent)
	}
	if g.Percentile90 != 14 || g.Percentile99 != 32 {
		t.Errorf("percentiles: %+v", g)
	}
}

func TestParseBattery(t *testing.T) {
	out := `Current Battery Service state:
  AC powered: false
  USB powered: true
  level: 87
  scale: 100
  voltage: 4123
  temperature: 305
  technology: Li-ion
  status: 2
  health: 2
  plugged: 2
`
	b := parseBattery(out)
	if b.Level != 87 || b.Scale != 100 {
		t.Errorf("level/scale: %+v", b)
	}
	if !b.USBPowered || b.ACPowered {
		t.Errorf("power flags: %+v", b)
	}
	if b.Technology != "Li-ion" {
		t.Errorf("tech: %q", b.Technology)
	}
}
