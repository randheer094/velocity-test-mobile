package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, path string, w, h int, fill color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestDiff_Identical(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	writePNG(t, a, 16, 16, color.RGBA{R: 50, G: 50, B: 50, A: 255})
	writePNG(t, b, 16, 16, color.RGBA{R: 50, G: 50, B: 50, A: 255})
	res, err := Diff(a, b, "", 0, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Compatible || res.MismatchedPixels != 0 || res.MismatchPct != 0 {
		t.Errorf("identical -> %+v", res)
	}
}

func TestDiff_Different(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	d := filepath.Join(dir, "diff.png")
	writePNG(t, a, 8, 8, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	writePNG(t, b, 8, 8, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	res, err := Diff(a, b, d, 0, 50)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Compatible {
		t.Fatalf("expected compatible")
	}
	if res.MismatchedPixels != 64 {
		t.Errorf("mismatched: %d, want 64", res.MismatchedPixels)
	}
	if !res.ExceedsTolerance {
		t.Errorf("expected ExceedsTolerance to be true")
	}
	if _, err := os.Stat(d); err != nil {
		t.Errorf("diff image missing: %v", err)
	}
}

func TestDiff_DifferentSizes(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	writePNG(t, a, 8, 8, color.RGBA{A: 255})
	writePNG(t, b, 16, 16, color.RGBA{A: 255})
	res, err := Diff(a, b, "", 0, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Compatible {
		t.Errorf("expected incompatible result")
	}
}
