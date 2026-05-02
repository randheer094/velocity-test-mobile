package testing

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/randheer094/velocity-test-mobile/internal/matcher"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// WaitResult is returned by every wait_until_* verb.
type WaitResult struct {
	OK         bool        `json:"ok"`
	Attempts   int         `json:"attempts"`
	WaitedMs   int64       `json:"waitedMs"`
	Element    *ui.Element `json:"element,omitempty"`
	MatchedNow int         `json:"matchedNow"`
	Reason     string      `json:"reason,omitempty"`
}

// WaitUntilVisible — Compose waitUntilExists(matcher).
func (o *Orchestrator) WaitUntilVisible(ctx context.Context, deviceID string, m *matcher.Matcher, timeoutMs, intervalMs int) (WaitResult, error) {
	if m == nil || m.IsEmpty() {
		return WaitResult{}, matcher.ErrEmptyMatcher
	}
	start := time.Now()
	var lastFound *ui.Element
	attempts, ok, err := pollUntil(ctx, timeoutMs, intervalMs, func(ctx context.Context) (bool, error) {
		root, err := o.Layout.Tree(ctx, deviceID)
		if err != nil {
			return false, err
		}
		all, err := matcher.FindAll(root, m)
		if err != nil && err != matcher.ErrNotFound {
			return false, nil
		}
		for _, e := range all {
			if matcher.IsDisplayed(e) {
				ec := e
				lastFound = &ec
				return true, nil
			}
		}
		return false, nil
	})
	res := WaitResult{
		Attempts: attempts,
		WaitedMs: time.Since(start).Milliseconds(),
		OK:       ok,
		Element:  lastFound,
	}
	if !ok && err == nil {
		res.Reason = "timed out waiting for element to be visible"
	}
	return res, nil
}

// WaitUntilNotVisible — Compose waitUntilDoesNotExist (or hidden).
func (o *Orchestrator) WaitUntilNotVisible(ctx context.Context, deviceID string, m *matcher.Matcher, timeoutMs, intervalMs int) (WaitResult, error) {
	if m == nil || m.IsEmpty() {
		return WaitResult{}, matcher.ErrEmptyMatcher
	}
	start := time.Now()
	attempts, ok, _ := pollUntil(ctx, timeoutMs, intervalMs, func(ctx context.Context) (bool, error) {
		root, err := o.Layout.Tree(ctx, deviceID)
		if err != nil {
			return false, err
		}
		all, _ := matcher.FindAll(root, m)
		for _, e := range all {
			if matcher.IsDisplayed(e) {
				return false, nil
			}
		}
		return true, nil
	})
	res := WaitResult{Attempts: attempts, WaitedMs: time.Since(start).Milliseconds(), OK: ok}
	if !ok {
		res.Reason = "element still visible after timeout"
	}
	return res, nil
}

// WaitUntilText — wait until a node matching m has the given text.
func (o *Orchestrator) WaitUntilText(ctx context.Context, deviceID string, m *matcher.Matcher, expected string, timeoutMs, intervalMs int) (WaitResult, error) {
	if m == nil || m.IsEmpty() {
		return WaitResult{}, matcher.ErrEmptyMatcher
	}
	start := time.Now()
	var lastFound *ui.Element
	attempts, ok, _ := pollUntil(ctx, timeoutMs, intervalMs, func(ctx context.Context) (bool, error) {
		root, err := o.Layout.Tree(ctx, deviceID)
		if err != nil {
			return false, err
		}
		all, _ := matcher.FindAll(root, m)
		for _, e := range all {
			if e.Text == expected || strings.Contains(e.Text, expected) {
				ec := e
				lastFound = &ec
				return true, nil
			}
		}
		return false, nil
	})
	res := WaitResult{Attempts: attempts, WaitedMs: time.Since(start).Milliseconds(), OK: ok, Element: lastFound}
	if !ok {
		res.Reason = fmt.Sprintf("never observed text %q on a matching node", expected)
	}
	return res, nil
}

// WaitUntilAtLeastOneExists — Compose waitUntilAtLeastOneExists. Polls
// until at least `min` matching elements are present.
func (o *Orchestrator) WaitUntilAtLeastOneExists(ctx context.Context, deviceID string, m *matcher.Matcher, minCount, timeoutMs, intervalMs int) (WaitResult, error) {
	if m == nil || m.IsEmpty() {
		return WaitResult{}, matcher.ErrEmptyMatcher
	}
	if minCount <= 0 {
		minCount = 1
	}
	start := time.Now()
	lastCount := 0
	attempts, ok, _ := pollUntil(ctx, timeoutMs, intervalMs, func(ctx context.Context) (bool, error) {
		root, err := o.Layout.Tree(ctx, deviceID)
		if err != nil {
			return false, err
		}
		all, _ := matcher.FindAll(root, m)
		lastCount = len(all)
		return lastCount >= minCount, nil
	})
	res := WaitResult{Attempts: attempts, WaitedMs: time.Since(start).Milliseconds(), OK: ok, MatchedNow: lastCount}
	if !ok {
		res.Reason = fmt.Sprintf("only %d matched, need >= %d", lastCount, minCount)
	}
	return res, nil
}

// WaitUntilCount — wait until the matcher resolves to exactly `count` nodes.
func (o *Orchestrator) WaitUntilCount(ctx context.Context, deviceID string, m *matcher.Matcher, count, timeoutMs, intervalMs int) (WaitResult, error) {
	if m == nil || m.IsEmpty() {
		return WaitResult{}, matcher.ErrEmptyMatcher
	}
	start := time.Now()
	lastCount := 0
	attempts, ok, _ := pollUntil(ctx, timeoutMs, intervalMs, func(ctx context.Context) (bool, error) {
		root, err := o.Layout.Tree(ctx, deviceID)
		if err != nil {
			return false, err
		}
		all, _ := matcher.FindAll(root, m)
		lastCount = len(all)
		return lastCount == count, nil
	})
	res := WaitResult{Attempts: attempts, WaitedMs: time.Since(start).Milliseconds(), OK: ok, MatchedNow: lastCount}
	if !ok {
		res.Reason = fmt.Sprintf("count stuck at %d, want %d", lastCount, count)
	}
	return res, nil
}

// WaitForIdle approximates Espresso onIdle() / Compose waitForIdle by
// polling the accessibility tree and waiting for two consecutive snapshots
// to hash identically over an `idleWindowMs` window.
func (o *Orchestrator) WaitForIdle(ctx context.Context, deviceID string, timeoutMs, idleWindowMs int) (WaitResult, error) {
	if timeoutMs <= 0 {
		timeoutMs = 8000
	}
	if idleWindowMs <= 0 {
		idleWindowMs = 500
	}
	intervalMs := idleWindowMs / 3
	if intervalMs < 100 {
		intervalMs = 100
	}
	start := time.Now()
	var lastHash string
	stableSince := time.Time{}
	attempts := 0
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		attempts++
		root, err := o.Layout.Tree(ctx, deviceID)
		if err == nil {
			h := hashTree(root)
			if h == lastHash {
				if stableSince.IsZero() {
					stableSince = time.Now()
				}
				if time.Since(stableSince) >= time.Duration(idleWindowMs)*time.Millisecond {
					return WaitResult{OK: true, Attempts: attempts, WaitedMs: time.Since(start).Milliseconds()}, nil
				}
			} else {
				lastHash = h
				stableSince = time.Time{}
			}
		}
		if time.Now().After(deadline) {
			return WaitResult{OK: false, Attempts: attempts, WaitedMs: time.Since(start).Milliseconds(), Reason: "tree did not stabilise within timeout"}, nil
		}
		select {
		case <-ctx.Done():
			return WaitResult{}, ctx.Err()
		case <-time.After(time.Duration(intervalMs) * time.Millisecond):
		}
	}
}

// hashTree produces a stable digest of the salient parts of a UI tree —
// class, text, content-desc, resource-id, bounds, and a few state flags.
func hashTree(root ui.Element) string {
	h := sha1.New()
	var walk func(ui.Element)
	walk = func(e ui.Element) {
		fmt.Fprintf(h, "%s|%s|%s|%s|%d,%d,%d,%d|%t|%t|%t|%t\n",
			e.Class, e.Text, e.Label, e.ResourceID,
			e.Bounds.X, e.Bounds.Y, e.Bounds.Width, e.Bounds.Height,
			e.Enabled, e.Focused, e.Checked, e.Selected,
		)
		for _, c := range e.Children {
			walk(c)
		}
	}
	walk(root)
	return hex.EncodeToString(h.Sum(nil))
}
