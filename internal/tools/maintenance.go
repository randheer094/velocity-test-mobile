package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterMaintenance registers reboot and wireless-ADB tools.
func RegisterMaintenance(s *mcp.Server, d *Deps) {
	type rebootArgs struct {
		DeviceArg
		Mode    string `json:"mode,omitempty" jsonschema:"empty | bootloader | recovery | sideload | fastboot"`
		Confirm bool   `json:"confirm" jsonschema:"must be true; reboots are destructive"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "device_reboot",
		Description: "Reboot the device. Requires confirm=true; default mode boots normally.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptrTrue()},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args rebootArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		if err := d.Maintenance.Reboot(ctx, dev, args.Mode, args.Confirm); err != nil {
			return errResult(err)
		}
		return textResult("reboot initiated")
	})

	type tcpipArgs struct {
		DeviceArg
		Port int `json:"port,omitempty" jsonschema:"TCP port; default 5555"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "wireless_enable",
		Description: "Switch the (currently USB-attached) device into TCP/IP mode for wireless ADB.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args tcpipArgs) (*mcp.CallToolResult, any, error) {
		dev, err := d.resolveDevice(ctx, args.Device)
		if err != nil {
			return errResult(err)
		}
		port := args.Port
		if port == 0 {
			port = 5555
		}
		if err := d.Maintenance.EnableWirelessADB(ctx, dev, port); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"port": port})
	})

	type connectArgs struct {
		HostPort string `json:"hostPort" jsonschema:"e.g. 192.168.1.5:5555"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "wireless_connect",
		Description: "Connect to a wireless ADB endpoint.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args connectArgs) (*mcp.CallToolResult, any, error) {
		out, err := d.Maintenance.ConnectWireless(ctx, args.HostPort)
		if err != nil {
			return errResult(err)
		}
		return textResult(out)
	})

	type pairArgs struct {
		HostPort string `json:"hostPort" jsonschema:"e.g. 192.168.1.5:37123 (the pairing port from the device)"`
		Code     string `json:"code" jsonschema:"the 6-digit pairing code shown on device"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "wireless_pair",
		Description: "Pair with a wireless ADB device (Android 11+).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args pairArgs) (*mcp.CallToolResult, any, error) {
		out, err := d.Maintenance.PairWireless(ctx, args.HostPort, args.Code)
		if err != nil {
			return errResult(err)
		}
		return textResult(out)
	})

	type discArgs struct {
		HostPort string `json:"hostPort,omitempty" jsonschema:"empty disconnects all wireless devices"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "wireless_disconnect",
		Description: "Disconnect from a wireless ADB endpoint (or all if hostPort is empty).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args discArgs) (*mcp.CallToolResult, any, error) {
		if err := d.Maintenance.DisconnectWireless(ctx, args.HostPort); err != nil {
			return errResult(err)
		}
		return textResult("disconnected")
	})
}
