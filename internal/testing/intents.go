package testing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/randheer094/velocity-mcp-mobile/internal/diagnostics"
)

// IntentRecorder approximates Espresso-Intents — but only the *recording*
// half. Stubbing (`intending(...).respondWith(...)`) requires in-process
// instrumentation and is not supported externally.
//
// The implementation scrapes ActivityManager logcat lines like
// "START u0 {act=android.intent.action.VIEW dat=https://example.com pkg=com.foo}"
// after a recording window opens, and re-parses them on each assert call.
type IntentRecorder struct {
	Logs *diagnostics.LogClient

	mu       sync.Mutex
	sessions map[string]intentSession
}

type intentSession struct {
	startedAt     time.Time
	since         string // logcat -T compatible string
	packageFilter string
}

// CapturedIntent is what the parser surfaces.
type CapturedIntent struct {
	Action   string `json:"action,omitempty"`
	Data     string `json:"data,omitempty"`
	Category string `json:"category,omitempty"`
	Package  string `json:"package,omitempty"`
	Class    string `json:"class,omitempty"`
	From     string `json:"from,omitempty"`
	Raw      string `json:"raw"`
	When     string `json:"when,omitempty"`
}

// NewIntentRecorder builds a recorder.
func NewIntentRecorder(logs *diagnostics.LogClient) *IntentRecorder {
	return &IntentRecorder{Logs: logs, sessions: map[string]intentSession{}}
}

// Start opens a new recording window for the given device. Multiple Start
// calls overwrite the prior window for that device.
func (r *IntentRecorder) Start(ctx context.Context, deviceID, packageFilter string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.Logs.Clear(ctx, deviceID); err != nil {
		return fmt.Errorf("clearing logcat to start intent monitor: %w", err)
	}
	r.sessions[deviceID] = intentSession{
		startedAt:     time.Now(),
		packageFilter: packageFilter,
	}
	return nil
}

// List returns every captured intent in the active window.
func (r *IntentRecorder) List(ctx context.Context, deviceID string) ([]CapturedIntent, error) {
	r.mu.Lock()
	sess, ok := r.sessions[deviceID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no intent monitor active for device %q; call intent_monitor_start first", deviceID)
	}
	lines, err := r.Logs.Tail(ctx, deviceID, diagnostics.LogOptions{
		Tag:      "ActivityManager",
		Priority: "I",
		MaxLines: 5000,
	})
	if err != nil {
		return nil, err
	}
	intents := parseIntents(lines, sess.packageFilter)
	return intents, nil
}

// IntentMatcher describes a subset of fields to look for in captured intents.
type IntentMatcher struct {
	Action       string `json:"action,omitempty"`
	Data         string `json:"data,omitempty" jsonschema:"exact match"`
	DataContains string `json:"dataContains,omitempty"`
	Package      string `json:"package,omitempty"`
	Category     string `json:"category,omitempty"`
}

// IntentAssertResult is what assert_intent_sent returns.
type IntentAssertResult struct {
	OK     bool             `json:"ok"`
	Match  *CapturedIntent  `json:"match,omitempty"`
	All    []CapturedIntent `json:"all"`
	Reason string           `json:"reason,omitempty"`
}

// AssertSent reports whether at least one captured intent satisfies im.
func (r *IntentRecorder) AssertSent(ctx context.Context, deviceID string, im IntentMatcher) (IntentAssertResult, error) {
	intents, err := r.List(ctx, deviceID)
	if err != nil {
		return IntentAssertResult{}, err
	}
	for _, it := range intents {
		if intentMatches(it, im) {
			return IntentAssertResult{OK: true, Match: &it, All: intents}, nil
		}
	}
	return IntentAssertResult{OK: false, All: intents, Reason: "no captured intent satisfied the matcher"}, nil
}

func intentMatches(i CapturedIntent, m IntentMatcher) bool {
	if m.Action != "" && i.Action != m.Action {
		return false
	}
	if m.Data != "" && i.Data != m.Data {
		return false
	}
	if m.DataContains != "" && !strings.Contains(i.Data, m.DataContains) {
		return false
	}
	if m.Package != "" && i.Package != m.Package {
		return false
	}
	if m.Category != "" && !strings.Contains(i.Category, m.Category) {
		return false
	}
	return true
}

var (
	startLineRE = regexp.MustCompile(`START\s+u\d+\s+\{(.+?)\}\s*(?:from\s+(.+))?`)
	kvRE        = regexp.MustCompile(`(\w+)=([^\s\}]+)`)
)

func parseIntents(lines []string, pkgFilter string) []CapturedIntent {
	out := []CapturedIntent{}
	for _, line := range lines {
		if !strings.Contains(line, "ActivityManager") {
			continue
		}
		if !strings.Contains(line, "START ") {
			continue
		}
		m := startLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		body := m[1]
		from := ""
		if len(m) > 2 {
			from = strings.TrimSpace(m[2])
		}
		ci := CapturedIntent{Raw: line, From: from}
		for _, kv := range kvRE.FindAllStringSubmatch(body, -1) {
			switch kv[1] {
			case "act":
				ci.Action = kv[2]
			case "dat":
				ci.Data = kv[2]
			case "cat":
				ci.Category = kv[2]
			case "pkg":
				ci.Package = kv[2]
			case "cmp":
				ci.Class = kv[2]
			}
		}
		// Derive the package from the component when only `cmp=pkg/.Class`
		// was supplied (common for explicit intents).
		if ci.Package == "" && ci.Class != "" {
			if i := strings.Index(ci.Class, "/"); i > 0 {
				ci.Package = ci.Class[:i]
			}
		}
		if pkgFilter != "" && ci.Package != pkgFilter {
			continue
		}
		// Best-effort timestamp prefix
		if i := strings.Index(line, " "); i > 0 {
			ci.When = line[:i]
		}
		out = append(out, ci)
	}
	return out
}
