package apps

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// SafeRelPath validates a relative path under an app's data dir. It rejects
// absolute paths, traversal segments, and any byte that could break out of
// the shell-escaped argument.
func SafeRelPath(rel string) (string, error) {
	if rel == "" {
		return ".", nil
	}
	if strings.ContainsAny(rel, "\x00") {
		return "", fmt.Errorf("path contains NUL")
	}
	cleaned := path.Clean(rel)
	if path.IsAbs(cleaned) {
		return "", fmt.Errorf("path %q must be relative to the package data dir", rel)
	}
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("path %q contains a parent traversal", rel)
	}
	for _, r := range cleaned {
		if r == 0 || r == '\n' || r == '`' || r == '$' || r == '\\' || r == '"' {
			return "", fmt.Errorf("path %q contains an unsafe character", rel)
		}
	}
	return cleaned, nil
}

// ListAppData runs `run-as <pkg> ls -la <relPath?>`.
func (c *Client) ListAppData(ctx context.Context, deviceID, pkg, relPath string) (string, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return "", err
	}
	rel, err := SafeRelPath(relPath)
	if err != nil {
		return "", err
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "run-as", pkg, "ls", "-la", rel)
	if err != nil {
		out := strings.TrimSpace(string(res.Stderr))
		if out == "" {
			out = strings.TrimSpace(string(res.Stdout))
		}
		if strings.Contains(out, "is not debuggable") || strings.Contains(out, "package not debuggable") {
			return "", fmt.Errorf("package %s is not debuggable; run-as is unavailable on release builds", pkg)
		}
		return "", err
	}
	return string(res.Stdout), nil
}

// ReadAppData runs `run-as <pkg> cat <relPath>`.
func (c *Client) ReadAppData(ctx context.Context, deviceID, pkg, relPath string) ([]byte, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return nil, err
	}
	if relPath == "" {
		return nil, fmt.Errorf("relativePath is required")
	}
	rel, err := SafeRelPath(relPath)
	if err != nil {
		return nil, err
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "run-as", pkg, "cat", rel)
	if err != nil {
		errOut := strings.TrimSpace(string(res.Stderr))
		if strings.Contains(errOut, "is not debuggable") || strings.Contains(errOut, "package not debuggable") {
			return nil, fmt.Errorf("package %s is not debuggable; run-as is unavailable on release builds", pkg)
		}
		return nil, err
	}
	return res.Stdout, nil
}
