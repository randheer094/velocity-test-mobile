package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterFiles registers push/pull file transfer tools.
func RegisterFiles(s *mcp.Server, d *Deps) {
	type pushArgs struct {
		DeviceArg
		Local  string `json:"local" jsonschema:"absolute path on the host"`
		Remote string `json:"remote" jsonschema:"absolute path on the device, e.g. /sdcard/Download/foo.txt"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "file_push",
		Description: "Upload a local file to the device.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pushArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Files.Push(ctx, dev, args.Local, args.Remote); err != nil {
			return errResult(err)
		}
		return textResult("pushed " + args.Local + " → " + args.Remote)
	})

	type pullArgs struct {
		DeviceArg
		Remote string `json:"remote"`
		Local  string `json:"local"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "file_pull",
		Description: "Download a file from the device to the host.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pullArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Files.Pull(ctx, dev, args.Remote, args.Local); err != nil {
			return errResult(err)
		}
		return textResult("pulled " + args.Remote + " → " + args.Local)
	})
}
