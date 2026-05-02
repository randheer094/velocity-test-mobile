package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
)

// ScreenshotClient takes screenshots from a connected device.
type ScreenshotClient struct {
	Adb        *adb.Client
	AndroidCLI *androidcli.Client
}

// NewScreenshotClient builds a ScreenshotClient.
func NewScreenshotClient(a *adb.Client, c *androidcli.Client) *ScreenshotClient {
	return &ScreenshotClient{Adb: a, AndroidCLI: c}
}

// Capture returns raw PNG bytes for the device's primary display (or the
// supplied displayID for multi-display devices, when non-empty).
func (s *ScreenshotClient) Capture(ctx context.Context, deviceID, displayID string) ([]byte, error) {
	if displayID == "" && s.AndroidCLI != nil && s.AndroidCLI.Available() {
		if data, err := s.captureViaAndroidCLI(ctx, deviceID); err == nil {
			return data, nil
		}
		// fall through
	}
	args := []string{"screencap", "-p"}
	if displayID != "" {
		args = append(args, "-d", displayID)
	}
	return s.Adb.ExecOut(ctx, deviceID, args...)
}

func (s *ScreenshotClient) captureViaAndroidCLI(ctx context.Context, deviceID string) ([]byte, error) {
	f, err := os.CreateTemp("", "android-screen-*.png")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	args := []string{"screen", "capture", "--output=" + path}
	if deviceID != "" {
		args = append(args, "--device", deviceID)
	}
	if _, err := s.AndroidCLI.Run(ctx, args...); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Save writes png to a host path. The extension determines the encoded
// format. The path must be absolute or under cwd / os.TempDir().
func Save(png []byte, dest string) (string, error) {
	resolved, err := safeOutputPath(dest)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(resolved))
	switch ext {
	case ".png":
		if err := os.WriteFile(resolved, png, 0o644); err != nil {
			return "", err
		}
		return resolved, nil
	case ".jpg", ".jpeg":
		img, err := decodePNG(png)
		if err != nil {
			return "", err
		}
		f, err := os.Create(resolved)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
			return "", err
		}
		return resolved, nil
	default:
		return "", fmt.Errorf("unsupported extension %q (use .png, .jpg, or .jpeg)", ext)
	}
}

func decodePNG(b []byte) (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("decoding PNG: %w", err)
	}
	return img, nil
}

func safeOutputPath(dest string) (string, error) {
	if dest == "" {
		return "", errors.New("output path is empty")
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	tmpDir, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		tmpDir = os.TempDir()
	}
	parent := filepath.Dir(abs)
	parentResolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		// Parent must exist; surface a clear error.
		return "", fmt.Errorf("output dir %s does not exist", parent)
	}
	if !insideAny(parentResolved, []string{cwd, tmpDir}) {
		return "", fmt.Errorf("output path %s is outside the working directory and the system temp dir", abs)
	}
	if strings.Contains(abs, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("output path must not contain ..")
	}
	return filepath.Join(parentResolved, filepath.Base(abs)), nil
}

func insideAny(target string, roots []string) bool {
	for _, r := range roots {
		if r == "" {
			continue
		}
		rel, err := filepath.Rel(r, target)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && rel != "") {
			return true
		}
	}
	return false
}
