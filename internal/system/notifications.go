package system

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// NotificationClient covers `dumpsys notification` parsing and the system
// statusbar shade controls (`cmd statusbar`).
type NotificationClient struct {
	Adb *adb.Client
}

// NewNotificationClient constructs a NotificationClient.
func NewNotificationClient(a *adb.Client) *NotificationClient { return &NotificationClient{Adb: a} }

// Notification is a structured subset of a single record from
// `dumpsys notification --noredact`.
type Notification struct {
	Package      string `json:"bundle_id"`
	Channel      string `json:"channel_id,omitempty"`
	Title        string `json:"title,omitempty"`
	Text         string `json:"text,omitempty"`
	Ongoing      bool   `json:"ongoing"`
	PostedTimeMs int64  `json:"posted_time_ms"`
	Tag          string `json:"tag,omitempty"`
	ID           int    `json:"id,omitempty"`
}

// ListFilter narrows the list to a package and/or channel.
type ListFilter struct {
	Package string
	Channel string
}

// List returns all currently-posted notifications, optionally filtered.
func (c *NotificationClient) List(ctx context.Context, deviceID string, f ListFilter) ([]Notification, error) {
	if f.Package != "" {
		if _, err := adb.MustQuotePackage(f.Package); err != nil {
			return nil, err
		}
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "dumpsys", "notification", "--noredact")
	if err != nil {
		return nil, err
	}
	all := parseNotifications(string(res.Stdout))
	if f.Package == "" && f.Channel == "" {
		return all, nil
	}
	out := make([]Notification, 0, len(all))
	for _, n := range all {
		if f.Package != "" && n.Package != f.Package {
			continue
		}
		if f.Channel != "" && n.Channel != f.Channel {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

// parseNotifications walks the dumpsys output block by block. Each
// notification record begins with `NotificationRecord(...)` and continues
// until the next NotificationRecord or section break.
// API 34 (and most Android versions since channels were introduced in API 26)
// emit the channel id inline in the NotificationRecord header as
// `Notification(channel=<id> ...)`. Older platform builds also emit a separate
// `mChannel=NotificationChannel{id=<id>, ...}` line; we accept either.
var (
	notifHeaderRE   = regexp.MustCompile(`NotificationRecord\([^)]*pkg=([A-Za-z0-9_.]+).*?\bid=(-?\d+)(?:\s+tag=(\S+))?`)
	inlineChannelRE = regexp.MustCompile(`\bNotification\(channel=([^\s)]+)`)
	channelLineRE   = regexp.MustCompile(`mChannel=NotificationChannel\{[^}]*?\bid=([^,}\s]+)`)
	titleLineRE     = regexp.MustCompile(`android\.title=(?:String|CharSequence)\s*\(([^)]*)\)`)
	textLineRE      = regexp.MustCompile(`android\.text=(?:String|CharSequence)\s*\(([^)]*)\)`)
	ongoingLineRE   = regexp.MustCompile(`mOngoing=(true|false)`)
	flagsLineRE     = regexp.MustCompile(`flags=0x([0-9a-fA-F]+)`)
	postedTimeLine  = regexp.MustCompile(`(?:mWhen|when|postTime)=(\d+)`)
)

func parseNotifications(out string) []Notification {
	// Split on NotificationRecord( boundaries. Index 0 is preamble.
	parts := regexp.MustCompile(`(?m)^\s*NotificationRecord\(`).Split(out, -1)
	if len(parts) <= 1 {
		return nil
	}
	results := make([]Notification, 0, len(parts)-1)
	for _, body := range parts[1:] {
		// Re-prepend the marker so the header regex matches.
		full := "NotificationRecord(" + body
		// Stop at the next blank top-level section to avoid bleeding into
		// neighbouring records (defensive: Split already separates them).
		if i := strings.Index(full, "\nNotificationRecord("); i > 0 {
			full = full[:i]
		}
		m := notifHeaderRE.FindStringSubmatch(full)
		if m == nil {
			continue
		}
		n := Notification{Package: m[1]}
		if id, err := strconv.Atoi(m[2]); err == nil {
			n.ID = id
		}
		if len(m) > 3 {
			n.Tag = strings.TrimSpace(m[3])
			if n.Tag == "null" {
				n.Tag = ""
			}
		}
		if cm := inlineChannelRE.FindStringSubmatch(full); cm != nil {
			n.Channel = cm[1]
		} else if cm := channelLineRE.FindStringSubmatch(full); cm != nil {
			n.Channel = cm[1]
		}
		if tm := titleLineRE.FindStringSubmatch(full); tm != nil {
			n.Title = strings.TrimSpace(tm[1])
		}
		if xm := textLineRE.FindStringSubmatch(full); xm != nil {
			n.Text = strings.TrimSpace(xm[1])
		}
		if om := ongoingLineRE.FindStringSubmatch(full); om != nil {
			n.Ongoing = om[1] == "true"
		} else if fm := flagsLineRE.FindStringSubmatch(full); fm != nil {
			if v, err := strconv.ParseInt(fm[1], 16, 64); err == nil {
				// FLAG_ONGOING_EVENT = 0x2
				n.Ongoing = (v & 0x2) != 0
			}
		}
		if pm := postedTimeLine.FindStringSubmatch(full); pm != nil {
			if v, err := strconv.ParseInt(pm[1], 10, 64); err == nil {
				n.PostedTimeMs = v
			}
		}
		results = append(results, n)
	}
	return results
}

// ShadeState is the legal value space for SetShade.
type ShadeState string

const (
	ShadeExpanded     ShadeState = "expanded"
	ShadeCollapsed    ShadeState = "collapsed"
	ShadeQuickSetting ShadeState = "quick_settings"
)

// SetShade opens or closes the system notification shade via `cmd statusbar`.
func (c *NotificationClient) SetShade(ctx context.Context, deviceID string, state ShadeState) error {
	var cmd string
	switch state {
	case ShadeExpanded:
		cmd = "expand-notifications"
	case ShadeCollapsed:
		cmd = "collapse"
	case ShadeQuickSetting:
		cmd = "expand-settings"
	default:
		return fmt.Errorf("invalid shade state %q (expected expanded|collapsed|quick_settings)", state)
	}
	_, err := c.Adb.ShellArgv(ctx, deviceID, "cmd", "statusbar", cmd)
	return err
}
