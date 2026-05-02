package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
)

// AnimationsClient toggles the three global animation scales.
type AnimationsClient struct {
	Adb *adb.Client
}

// NewAnimationsClient constructs an AnimationsClient.
func NewAnimationsClient(a *adb.Client) *AnimationsClient { return &AnimationsClient{Adb: a} }

var animationKeys = []string{
	"window_animation_scale",
	"transition_animation_scale",
	"animator_duration_scale",
}

// AnimationState reports the current scales.
type AnimationState struct {
	Window     string `json:"window_animation_scale"`
	Transition string `json:"transition_animation_scale"`
	Animator   string `json:"animator_duration_scale"`
}

// Get reads the current scales.
func (a *AnimationsClient) Get(ctx context.Context, deviceID string) (AnimationState, error) {
	out := AnimationState{}
	for i, k := range animationKeys {
		res, err := a.Adb.ShellArgv(ctx, deviceID, "settings", "get", "global", k)
		if err != nil {
			return out, err
		}
		v := strings.TrimSpace(string(res.Stdout))
		switch i {
		case 0:
			out.Window = v
		case 1:
			out.Transition = v
		case 2:
			out.Animator = v
		}
	}
	return out, nil
}

// Set writes a single scale value (commonly 0 to disable, 1 for default).
func (a *AnimationsClient) Set(ctx context.Context, deviceID string, scale float64) error {
	if scale < 0 || scale > 10 {
		return fmt.Errorf("scale must be in [0,10]")
	}
	v := fmt.Sprintf("%g", scale)
	for _, k := range animationKeys {
		if _, err := a.Adb.ShellArgv(ctx, deviceID, "settings", "put", "global", k, v); err != nil {
			return err
		}
	}
	return nil
}

// Quote-only no-op import to satisfy linters.
var _ = adb.QuoteForShell
