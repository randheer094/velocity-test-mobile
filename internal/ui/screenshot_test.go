package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeOutputPath_Empty(t *testing.T) {
	if _, err := safeOutputPath(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestSafeOutputPath_InsideCwd(t *testing.T) {
	dest := "safe-output-path-test.png"
	got, err := safeOutputPath(dest)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected an absolute path, got %q", got)
	}
	if filepath.Base(got) != dest {
		t.Fatalf("got %q, want basename %q", got, dest)
	}
}

func TestSafeOutputPath_InsideTempDir(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.png")
	got, err := safeOutputPath(dest)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if filepath.Base(got) != "out.png" {
		t.Fatalf("got %q", got)
	}
}

func TestSafeOutputPath_ParentDoesNotExist(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "does-not-exist-subdir", "out.png")
	if _, err := safeOutputPath(dest); err == nil {
		t.Fatal("expected error for a non-existent parent directory")
	}
}

func TestSafeOutputPath_OutsideCwdAndTempDir(t *testing.T) {
	_, err := safeOutputPath("/safe-output-path-outside-test-should-not-exist.png")
	if err == nil {
		t.Fatal("expected error for a path outside cwd and the system temp dir")
	}
	if !strings.Contains(err.Error(), "outside") {
		t.Fatalf("err = %v, want an 'outside' message", err)
	}
}

func samplePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding fixture PNG: %v", err)
	}
	return buf.Bytes()
}

func TestSave_PNG(t *testing.T) {
	data := samplePNG(t)
	dest := filepath.Join(t.TempDir(), "out.png")
	path, err := Save(data, dest)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal(".png save should write raw bytes unchanged")
	}
}

func TestSave_JPEG(t *testing.T) {
	data := samplePNG(t)
	dest := filepath.Join(t.TempDir(), "out.jpg")
	path, err := Save(data, dest)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if len(got) < 2 || got[0] != 0xFF || got[1] != 0xD8 {
		t.Fatalf("saved file doesn't look like a JPEG (missing SOI marker): %x", got[:min(len(got), 8)])
	}
}

func TestSave_UnsupportedExtension(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.gif")
	if _, err := Save(samplePNG(t), dest); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}
