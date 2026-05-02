package input

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// GetClipboard reads the primary clipboard. Requires API 29+ (Android 10).
func (c *Client) GetClipboard(ctx context.Context, deviceID string) (string, error) {
	res, err := c.Adb.Shell(ctx, deviceID, "cmd clipboard get-primary --user 0")
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(string(res.Stdout))
	if out == "" {
		return "", nil
	}
	if strings.Contains(out, "Cannot get clipboard from") {
		return "", fmt.Errorf("device denied clipboard read; ensure focused window has clipboard access")
	}
	return out, nil
}

// SetClipboard writes the primary clipboard. Encodes unicode via base64 to
// avoid shell quoting hazards.
func (c *Client) SetClipboard(ctx context.Context, deviceID, text string) error {
	if isASCII(text) {
		cmd := fmt.Sprintf("echo -n %s | cmd clipboard set-primary --user 0", adb.QuoteForShell(text))
		_, err := c.Adb.Shell(ctx, deviceID, cmd)
		return err
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	cmd := fmt.Sprintf("echo -n %s | base64 -d | cmd clipboard set-primary --user 0", adb.QuoteForShell(encoded))
	_, err := c.Adb.Shell(ctx, deviceID, cmd)
	return err
}
