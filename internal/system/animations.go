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

// Get reads the current scales, fusing the three `settings get` calls into
// a single adb shell invocation instead of three.
func (a *AnimationsClient) Get(ctx context.Context, deviceID string) (AnimationState, error) {
	res, err := a.Adb.Shell(ctx, deviceID, buildAnimationsGetCmd())
	if err != nil {
		return AnimationState{}, err
	}
	lines := strings.Split(strings.TrimRight(string(res.Stdout), "\n"), "\n")
	out := AnimationState{}
	for i := range animationKeys {
		var v string
		if i < len(lines) {
			v = strings.TrimSpace(lines[i])
		}
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

func buildAnimationsGetCmd() string {
	parts := make([]string, len(animationKeys))
	for i, k := range animationKeys {
		parts[i] = "settings get global " + k
	}
	return strings.Join(parts, " && ")
}

// Set writes a single scale value (commonly 0 to disable, 1 for default) to
// all three animation keys, fused into a single adb shell invocation.
func (a *AnimationsClient) Set(ctx context.Context, deviceID string, scale float64) error {
	if scale < 0 || scale > 10 {
		return fmt.Errorf("scale must be in [0,10]")
	}
	v := fmt.Sprintf("%g", scale)
	_, err := a.Adb.Shell(ctx, deviceID, buildAnimationsSetCmd(v))
	return err
}

func buildAnimationsSetCmd(value string) string {
	parts := make([]string, len(animationKeys))
	for i, k := range animationKeys {
		parts[i] = "settings put global " + k + " " + value
	}
	return strings.Join(parts, " && ")
}
