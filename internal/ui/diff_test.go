package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
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

// TestDiffFastPath_MatchesGeneric guards against the NRGBA/RGBA fast paths
// (diffNRGBA/diffRGBA) drifting from the generic image.Image-interface path
// (diffGeneric) — in particular the NRGBA premultiplication math, which is
// the one place the fast path has to replicate a nontrivial stdlib formula
// rather than just compare raw bytes.
func TestDiffFastPath_MatchesGeneric(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	bounds := image.Rect(0, 0, 20, 20)

	t.Run("NRGBA", func(t *testing.T) {
		a := randomNRGBA(rng, bounds)
		b := randomNRGBA(rng, bounds)
		assertFastPathMatchesGeneric(t, a, b, bounds)
	})

	t.Run("RGBA", func(t *testing.T) {
		a := randomRGBA(rng, bounds)
		b := randomRGBA(rng, bounds)
		assertFastPathMatchesGeneric(t, a, b, bounds)
	})
}

func assertFastPathMatchesGeneric(t *testing.T, a, b image.Image, bounds image.Rectangle) {
	t.Helper()
	for _, tolerance := range []int{0, 10, 50} {
		fastDiffImg := image.NewRGBA(bounds)
		genericDiffImg := image.NewRGBA(bounds)
		fastCount := diffPixels(a, b, bounds, tolerance, fastDiffImg)
		genericCount := diffGeneric(a, b, bounds, tolerance, genericDiffImg)
		if fastCount != genericCount {
			t.Errorf("tolerance=%d: fast path mismatch count = %d, generic = %d", tolerance, fastCount, genericCount)
		}
		if !bytes.Equal(fastDiffImg.Pix, genericDiffImg.Pix) {
			t.Errorf("tolerance=%d: fast path diff image bytes differ from generic path", tolerance)
		}
	}
}

func randomNRGBA(rng *rand.Rand, r image.Rectangle) *image.NRGBA {
	img := image.NewNRGBA(r)
	for i := range img.Pix {
		img.Pix[i] = uint8(rng.Intn(256))
	}
	return img
}

func randomRGBA(rng *rand.Rand, r image.Rectangle) *image.RGBA {
	img := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			a := uint8(rng.Intn(256))
			// RGBA channels must not exceed the pixel's own alpha
			// (premultiplied invariant); otherwise this isn't a valid
			// *image.RGBA pixel and .At().RGBA() would misbehave.
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(rng.Intn(int(a) + 1)),
				G: uint8(rng.Intn(int(a) + 1)),
				B: uint8(rng.Intn(int(a) + 1)),
				A: a,
			})
		}
	}
	return img
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
