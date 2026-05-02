package system

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// ServiceClient inspects running services via `dumpsys activity services`.
type ServiceClient struct {
	Adb *adb.Client
}

// NewServiceClient constructs a ServiceClient.
func NewServiceClient(a *adb.Client) *ServiceClient { return &ServiceClient{Adb: a} }

// ServiceState summarises a single ServiceRecord. Fields not surfaced by the
// dump for this service are zero/empty.
type ServiceState struct {
	Running          bool   `json:"running"`
	Foreground       bool   `json:"foreground"`
	Component        string `json:"component,omitempty"`
	NotificationID   *int   `json:"notification_id,omitempty"`
	StartID          *int   `json:"start_id,omitempty"`
	LastIntentAction string `json:"last_intent_action,omitempty"`
}

// GetState parses `dumpsys activity services <pkg>`. When `component` is
// supplied, only the matching ServiceRecord is considered; otherwise the
// first record found for `pkg` is used.
//
// Returns a zero-valued ServiceState (Running=false) when no record exists.
func (c *ServiceClient) GetState(ctx context.Context, deviceID, pkg, component string) (ServiceState, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return ServiceState{}, err
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "dumpsys", "activity", "services", pkg)
	if err != nil {
		return ServiceState{}, err
	}
	return parseServiceState(string(res.Stdout), pkg, component), nil
}

var (
	serviceRecordRE = regexp.MustCompile(`ServiceRecord\{[^}]*\s+([A-Za-z0-9_.]+/[A-Za-z0-9_.\$]+)\}`)
	isForegroundRE  = regexp.MustCompile(`isForeground=(true|false)`)
	startIDRE       = regexp.MustCompile(`startId=(\d+)`)
	// foregroundId comes from ServiceRecord.dumpDebug; older dumps used the
	// inline `id=0x..` token inside Notification{...}.
	notificationRE    = regexp.MustCompile(`foregroundId=(-?\d+)`)
	notificationHexRE = regexp.MustCompile(`Notification[\{(][^})]*\bid=0x([0-9a-fA-F]+)\b`)
	// `intent={act=...}` is lowercase in dumpsys output.
	intentActionRE = regexp.MustCompile(`(?i)intent[=]?\{[^}]*\bact=([^\s}]+)`)
)

// parseServiceState walks the dumpsys body block-by-block. A service block
// starts with a ServiceRecord{} header and ends at the next blank-then-non-
// indented line (or another ServiceRecord). We capture everything within the
// chosen block before extracting fields.
func parseServiceState(out, pkg, component string) ServiceState {
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var (
		current   strings.Builder
		currentID string
		picked    strings.Builder
		pickedID  string
	)
	flush := func() {
		if current.Len() == 0 {
			return
		}
		body := current.String()
		// Heuristic: service is "running" iff it has a ServiceRecord at all.
		// Pick the first matching record for `pkg`; if `component` is set,
		// require it to match `<pkg>/<component>`.
		if currentID == "" {
			current.Reset()
			return
		}
		if !strings.HasPrefix(currentID, pkg+"/") {
			current.Reset()
			currentID = ""
			return
		}
		if component != "" && !componentMatches(currentID, pkg, component) {
			current.Reset()
			currentID = ""
			return
		}
		if picked.Len() == 0 {
			picked.WriteString(body)
			pickedID = currentID
		}
		current.Reset()
		currentID = ""
	}
	for sc.Scan() {
		line := sc.Text()
		if m := serviceRecordRE.FindStringSubmatch(line); m != nil {
			flush()
			currentID = m[1]
		}
		if currentID != "" {
			current.WriteString(line)
			current.WriteByte('\n')
		}
	}
	flush()

	if picked.Len() == 0 {
		return ServiceState{}
	}
	body := picked.String()
	state := ServiceState{Running: true, Component: pickedID}
	if m := isForegroundRE.FindStringSubmatch(body); m != nil {
		state.Foreground = m[1] == "true"
	}
	if m := startIDRE.FindStringSubmatch(body); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			state.StartID = &v
		}
	}
	if m := notificationRE.FindStringSubmatch(body); m != nil {
		if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
			id := int(v)
			state.NotificationID = &id
		}
	} else if m := notificationHexRE.FindStringSubmatch(body); m != nil {
		if v, err := strconv.ParseInt(m[1], 16, 64); err == nil {
			id := int(v)
			state.NotificationID = &id
		}
	}
	if m := intentActionRE.FindStringSubmatch(body); m != nil {
		state.LastIntentAction = m[1]
	}
	return state
}

// ServiceExpectation lets WaitForState assert a subset of fields.
type ServiceExpectation struct {
	Running    *bool
	Foreground *bool
}

// WaitForState polls GetState until every set field of `want` matches, or the
// timeout elapses.
func (c *ServiceClient) WaitForState(ctx context.Context, deviceID, pkg, component string, want ServiceExpectation, timeout time.Duration) (ServiceState, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var last ServiceState
	for {
		st, err := c.GetState(ctx, deviceID, pkg, component)
		if err == nil {
			last = st
			if expectationMet(st, want) {
				return st, nil
			}
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return last, fmt.Errorf("service state did not converge within %s (last: running=%v foreground=%v)", timeout, last.Running, last.Foreground)
}

// componentMatches accepts any of the common forms a caller might supply for
// a service component name and compares it to the `<pkg>/<class>` token that
// dumpsys actually printed. Accepted forms:
//
//	".MockLocationService"                          (relative)
//	"MockLocationService"                           (simple class name)
//	"dev.randheer094.dev.MockLocationService"       (FQN)
func componentMatches(currentID, pkg, component string) bool {
	slash := strings.Index(currentID, "/")
	if slash < 0 {
		return false
	}
	currentClass := currentID[slash+1:]
	currentFQN := currentClass
	if strings.HasPrefix(currentFQN, ".") {
		currentFQN = pkg + currentFQN
	}
	wantFQN := component
	if strings.HasPrefix(wantFQN, ".") {
		wantFQN = pkg + wantFQN
	} else if !strings.Contains(wantFQN, ".") {
		// simple name like "MockLocationService"
		wantFQN = pkg + "." + wantFQN
	}
	return currentFQN == wantFQN
}

func expectationMet(st ServiceState, want ServiceExpectation) bool {
	if want.Running != nil && st.Running != *want.Running {
		return false
	}
	if want.Foreground != nil && st.Foreground != *want.Foreground {
		return false
	}
	return true
}
