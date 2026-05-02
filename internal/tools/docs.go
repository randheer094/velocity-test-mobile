package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
)

// RegisterDocs registers the Android Knowledge Base lookup tools.
// Both require the `android` agent CLI.
func RegisterDocs(s *mcp.Server, d *Deps) {
	requireCLI := func() error {
		if d.AndroidCLI == nil || !d.AndroidCLI.Available() {
			return androidcli.ErrNotInstalled
		}
		return nil
	}

	type searchArgs struct {
		Query string `json:"query"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "docs_search",
		Description: "Search the Android Knowledge Base via the Android CLI.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args searchArgs) (*mcp.CallToolResult, any, error) {
		if err := requireCLI(); err != nil {
			return errResult(err)
		}
		res, err := d.AndroidCLI.Run(ctx, "docs", "search", args.Query)
		if err != nil {
			return errResult(err)
		}
		return textResult(string(res.Stdout))
	})

	type fetchArgs struct {
		URL string `json:"url" jsonschema:"a kb:// URL returned by docs_search"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "docs_fetch",
		Description: "Fetch a specific entry from the Android Knowledge Base.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args fetchArgs) (*mcp.CallToolResult, any, error) {
		if err := requireCLI(); err != nil {
			return errResult(err)
		}
		res, err := d.AndroidCLI.Run(ctx, "docs", "fetch", args.URL)
		if err != nil {
			return errResult(err)
		}
		return textResult(string(res.Stdout))
	})
}
