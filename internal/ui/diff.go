package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
)

// DiffResult is the structured outcome of a screen-diff comparison.
type DiffResult struct {
	WidthA           int     `json:"widthA"`
	HeightA          int     `json:"heightA"`
	WidthB           int     `json:"widthB"`
	HeightB          int     `json:"heightB"`
	Compatible       bool    `json:"compatible"`
	TotalPixels      int     `json:"totalPixels"`
	MismatchedPixels int     `json:"mismatchedPixels"`
	MismatchPct      float64 `json:"mismatchPct"`
	Tolerance        int     `json:"tolerance"`
	ExceedsTolerance bool    `json:"exceedsTolerance"`
	DiffImagePath    string  `json:"diffImagePath,omitempty"`
	Notes            string  `json:"notes,omitempty"`
}

// Diff compares two PNGs and optionally writes a diff image highlighting
// the differing pixels in red. tolerance is per-channel 0..255; pctThreshold
// is 0..100.
func Diff(pathA, pathB, diffOut string, tolerance int, pctThreshold float64) (DiffResult, error) {
	if tolerance < 0 {
		tolerance = 0
	}
	imgA, err := readPNG(pathA)
	if err != nil {
		return DiffResult{}, fmt.Errorf("reading %s: %w", pathA, err)
	}
	imgB, err := readPNG(pathB)
	if err != nil {
		return DiffResult{}, fmt.Errorf("reading %s: %w", pathB, err)
	}
	bA := imgA.Bounds()
	bB := imgB.Bounds()
	res := DiffResult{
		WidthA:    bA.Dx(),
		HeightA:   bA.Dy(),
		WidthB:    bB.Dx(),
		HeightB:   bB.Dy(),
		Tolerance: tolerance,
	}
	if bA.Size() != bB.Size() {
		res.Compatible = false
		res.Notes = "image dimensions differ; pixel-level diff requires identical sizes"
		return res, nil
	}
	res.Compatible = true
	total := bA.Dx() * bA.Dy()
	res.TotalPixels = total
	var diffImg *image.RGBA
	if diffOut != "" {
		diffImg = image.NewRGBA(bA)
	}
	res.MismatchedPixels = diffPixels(imgA, imgB, bA, tolerance, diffImg)
	if total > 0 {
		res.MismatchPct = 100.0 * float64(res.MismatchedPixels) / float64(total)
	}
	res.ExceedsTolerance = res.MismatchPct > pctThreshold
	if diffImg != nil {
		out, err := safeOutputPath(diffOut)
		if err != nil {
			return res, err
		}
		f, err := os.Create(out)
		if err != nil {
			return res, err
		}
		defer f.Close()
		if err := png.Encode(f, diffImg); err != nil {
			return res, err
		}
		res.DiffImagePath = out
	}
	return res, nil
}

func readPNG(p string) (image.Image, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	if len(b) < 8 || string(b[:8]) != "\x89PNG\r\n\x1a\n" {
		return nil, errors.New("not a PNG file")
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func absDiff8(a, b uint32) int {
	// RGBA() returns 16-bit values; reduce to 8 bits for tolerance check.
	a8 := int(a >> 8)
	b8 := int(b >> 8)
	if a8 > b8 {
		return a8 - b8
	}
	return b8 - a8
}

// diffPixels compares every pixel in b between imgA and imgB, optionally
// painting diffImg (mismatch in red, match in light grey), and returns the
// mismatch count. It dispatches to a direct pixel-buffer fast path when both
// images share a concrete type that png.Decode commonly produces
// (*image.NRGBA for images with an alpha channel, *image.RGBA for opaque
// ones), falling back to the generic image.Image-interface path otherwise.
func diffPixels(imgA, imgB image.Image, b image.Rectangle, tolerance int, diffImg *image.RGBA) int {
	if a, ok := imgA.(*image.NRGBA); ok {
		if bb, ok := imgB.(*image.NRGBA); ok {
			return diffNRGBA(a, bb, b, tolerance, diffImg)
		}
	}
	if a, ok := imgA.(*image.RGBA); ok {
		if bb, ok := imgB.(*image.RGBA); ok {
			return diffRGBA(a, bb, b, tolerance, diffImg)
		}
	}
	return diffGeneric(imgA, imgB, b, tolerance, diffImg)
}

func diffGeneric(imgA, imgB image.Image, b image.Rectangle, tolerance int, diffImg *image.RGBA) int {
	mismatched := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r1, g1, b1, a1 := imgA.At(x, y).RGBA()
			r2, g2, b2, a2 := imgB.At(x, y).RGBA()
			if pixelMismatch(r1, g1, b1, a1, r2, g2, b2, a2, tolerance) {
				mismatched++
				setDiffPixel(diffImg, x, y, true)
			} else {
				setDiffPixel(diffImg, x, y, false)
			}
		}
	}
	return mismatched
}

// diffRGBA is the fast path for two *image.RGBA sources. RGBA's Pix bytes
// are already premultiplied, so color.RGBA.RGBA() (widen 8-bit to 16-bit via
// v|v<<8, no further scaling) is replicated by widenChannel below.
func diffRGBA(imgA, imgB *image.RGBA, b image.Rectangle, tolerance int, diffImg *image.RGBA) int {
	mismatched := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			oa := imgA.PixOffset(x, y)
			ob := imgB.PixOffset(x, y)
			pa := imgA.Pix[oa : oa+4 : oa+4]
			pb := imgB.Pix[ob : ob+4 : ob+4]
			r1, g1, b1, a1 := widenChannel(pa[0]), widenChannel(pa[1]), widenChannel(pa[2]), widenChannel(pa[3])
			r2, g2, b2, a2 := widenChannel(pb[0]), widenChannel(pb[1]), widenChannel(pb[2]), widenChannel(pb[3])
			if pixelMismatch(r1, g1, b1, a1, r2, g2, b2, a2, tolerance) {
				mismatched++
				setDiffPixel(diffImg, x, y, true)
			} else {
				setDiffPixel(diffImg, x, y, false)
			}
		}
	}
	return mismatched
}

// diffNRGBA is the fast path for two *image.NRGBA sources. NRGBA's Pix bytes
// are non-premultiplied, so color.NRGBA.RGBA()'s widen-then-premultiply
// math is replicated by premultiplyChannel below.
func diffNRGBA(imgA, imgB *image.NRGBA, b image.Rectangle, tolerance int, diffImg *image.RGBA) int {
	mismatched := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			oa := imgA.PixOffset(x, y)
			ob := imgB.PixOffset(x, y)
			pa := imgA.Pix[oa : oa+4 : oa+4]
			pb := imgB.Pix[ob : ob+4 : ob+4]
			aA, aB := pa[3], pb[3]
			r1, g1, b1, a1 := premultiplyChannel(pa[0], aA), premultiplyChannel(pa[1], aA), premultiplyChannel(pa[2], aA), widenChannel(aA)
			r2, g2, b2, a2 := premultiplyChannel(pb[0], aB), premultiplyChannel(pb[1], aB), premultiplyChannel(pb[2], aB), widenChannel(aB)
			if pixelMismatch(r1, g1, b1, a1, r2, g2, b2, a2, tolerance) {
				mismatched++
				setDiffPixel(diffImg, x, y, true)
			} else {
				setDiffPixel(diffImg, x, y, false)
			}
		}
	}
	return mismatched
}

// widenChannel replicates color.RGBA.RGBA()'s per-channel conversion: an
// 8-bit value widened to 16-bit by duplicating it into the low byte.
func widenChannel(v uint8) uint32 {
	v32 := uint32(v)
	return v32 | v32<<8
}

// premultiplyChannel replicates color.NRGBA.RGBA()'s per-channel conversion:
// widen to 16-bit, then scale by alpha (0..255) as the stdlib does —
// multiply before dividing, matching its truncation behavior exactly.
func premultiplyChannel(v, alpha uint8) uint32 {
	v32 := widenChannel(v)
	v32 *= uint32(alpha)
	v32 /= 0xff
	return v32
}

func pixelMismatch(r1, g1, b1, a1, r2, g2, b2, a2 uint32, tolerance int) bool {
	return absDiff8(r1, r2) > tolerance ||
		absDiff8(g1, g2) > tolerance ||
		absDiff8(b1, b2) > tolerance ||
		absDiff8(a1, a2) > tolerance
}

func setDiffPixel(diffImg *image.RGBA, x, y int, mismatched bool) {
	if diffImg == nil {
		return
	}
	off := diffImg.PixOffset(x, y)
	px := diffImg.Pix[off : off+4 : off+4]
	if mismatched {
		px[0], px[1], px[2], px[3] = 255, 0, 0, 255
	} else {
		px[0], px[1], px[2], px[3] = 200, 200, 200, 255
	}
}
