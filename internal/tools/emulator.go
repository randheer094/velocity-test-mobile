package tools

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
)

// RegisterEmulator registers emulator-lifecycle tools (require android CLI).
func RegisterEmulator(s *mcp.Server, d *Deps) {
	requireCLI := func() error {
		if d.AndroidCLI == nil || !d.AndroidCLI.Available() {
			return androidcli.ErrNotInstalled
		}
		return nil
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "emulator_list",
		Description: "List Android emulators known to the Android CLI (running and stopped).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		if err := requireCLI(); err != nil {
			return errResult(err)
		}
		res, err := d.AndroidCLI.Run(ctx, "emulator", "list")
		if err != nil {
			return errResult(err)
		}
		return textResult(string(res.Stdout))
	})

	type startArgs struct {
		Profile string `json:"profile" jsonschema:"emulator profile name (e.g. medium_phone)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "emulator_start",
		Description: "Start an Android emulator by profile name.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args startArgs) (*mcp.CallToolResult, any, error) {
		if err := requireCLI(); err != nil {
			return errResult(err)
		}
		if strings.TrimSpace(args.Profile) == "" {
			return errResult(errInvalid("profile is required"))
		}
		res, err := d.AndroidCLI.Run(ctx, "emulator", "start", args.Profile)
		if err != nil {
			return errResult(err)
		}
		return textResult(string(res.Stdout))
	})

	type stopArgs struct {
		Serial string `json:"serial" jsonschema:"running emulator serial (e.g. emulator-5554)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "emulator_stop",
		Description: "Stop a running Android emulator.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptrTrue()},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args stopArgs) (*mcp.CallToolResult, any, error) {
		if err := requireCLI(); err != nil {
			return errResult(err)
		}
		if strings.TrimSpace(args.Serial) == "" {
			return errResult(errInvalid("serial is required"))
		}
		res, err := d.AndroidCLI.Run(ctx, "emulator", "stop", args.Serial)
		if err != nil {
			return errResult(err)
		}
		return textResult(string(res.Stdout))
	})
}

func errInvalid(msg string) error { return &invalidArgErr{msg: msg} }

type invalidArgErr struct{ msg string }

func (e *invalidArgErr) Error() string { return "invalid argument: " + e.msg }
