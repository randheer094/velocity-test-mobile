package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// RegisterRecording registers screen recording tools.
func RegisterRecording(s *mcp.Server, d *Deps) {
	type startArgs struct {
		DeviceArg
		TimeLimitSec int    `json:"timeLimitSec,omitempty" jsonschema:"max recording duration; 0 = use device default"`
		BitrateMbps  int    `json:"bitrateMbps,omitempty" jsonschema:"target bitrate in Mbps; 0 = device default"`
		Size         string `json:"size,omitempty" jsonschema:"video resolution e.g. 720x1280; empty = native"`
		Output       string `json:"output,omitempty" jsonschema:"host destination .mp4; auto-generated when empty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_record_start",
		Description: "Begin a screen recording on the device. One recording per device at a time.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args startArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		res, err := d.Recorder.Start(ctx, dev, ui.StartOptions{
			TimeLimitSec: args.TimeLimitSec,
			BitrateMbps:  args.BitrateMbps,
			SizeWxH:      args.Size,
			Output:       args.Output,
		})
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "screen_record_stop",
		Description: "Stop the active recording on the device, pull the file to host, and report metadata.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args DeviceArg) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		res, err := d.Recorder.Stop(ctx, dev)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})
}
