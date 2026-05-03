// Package tools wires every MCP tool the server exposes.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-test-mobile/internal/adb"
	"github.com/randheer094/velocity-test-mobile/internal/androidcli"
	"github.com/randheer094/velocity-test-mobile/internal/apps"
	"github.com/randheer094/velocity-test-mobile/internal/device"
	"github.com/randheer094/velocity-test-mobile/internal/diagnostics"
	"github.com/randheer094/velocity-test-mobile/internal/input"
	"github.com/randheer094/velocity-test-mobile/internal/system"
	apptest "github.com/randheer094/velocity-test-mobile/internal/testing"
	"github.com/randheer094/velocity-test-mobile/internal/ui"
)

// Deps bundles every shared client used by tool handlers.
type Deps struct {
	Adb           *adb.Client
	AndroidCLI    *androidcli.Client
	Resolver      *device.Resolver
	Apps          *apps.Client
	Layout        *ui.LayoutClient
	Screenshot    *ui.ScreenshotClient
	Input         *input.Client
	Logs          *diagnostics.LogClient
	Record        *diagnostics.RecordClient
	Screen        *system.ScreenClient
	Animations    *system.AnimationsClient
	Activity      *system.ActivityClient
	Service       *system.ServiceClient
	Location      *system.LocationClient
	Notifications *system.NotificationClient
	Shell         *system.ShellClient
	State         *system.StateClient
	Tester        *apptest.Orchestrator
	Intents       *apptest.IntentRecorder
}

// resolveDevice returns the chosen device's serial or an actionable error.
func (d *Deps) resolveDevice(ctx context.Context, id string) (string, error) {
	dev, err := d.Resolver.Resolve(ctx, id)
	if err != nil {
		return "", err
	}
	return dev.Serial, nil
}

// jsonResult wraps any JSON-marshalable value as a tool TextContent payload.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, v, nil
}

// textResult emits a plain text payload.
func textResult(text string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

// errResult wraps an error into a tool error result (IsError=true) so it is
// surfaced to the client rather than being treated as protocol failure.
func errResult(err error) (*mcp.CallToolResult, any, error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}, nil, nil
}

// ptrTrue / ptrFalse for the tool annotation pointer fields.
func ptrTrue() *bool  { v := true; return &v }
func ptrFalse() *bool { v := false; return &v }

// Direct-form helpers for Server.AddTool (non-generic) — return a single
// *mcp.CallToolResult instead of the 3-tuple expected by the generic AddTool.
func jsonResultDirect(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "marshal error: " + err.Error()}},
		}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

func textResultDirect(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func errResultDirect(err error) *mcp.CallToolResult {
	if err == nil {
		err = errors.New("unknown error")
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}

// Common types reused by many handlers ----------------------------------

// DeviceArg is embedded in every device-targeted argument struct.
type DeviceArg struct {
	Device string `json:"device,omitempty" jsonschema:"the target device serial; omit if exactly one device is connected"`
}
