package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
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
	mismatched := 0
	var diffImg *image.RGBA
	if diffOut != "" {
		diffImg = image.NewRGBA(bA)
	}
	for y := bA.Min.Y; y < bA.Max.Y; y++ {
		for x := bA.Min.X; x < bA.Max.X; x++ {
			r1, g1, b1, a1 := imgA.At(x, y).RGBA()
			r2, g2, b2, a2 := imgB.At(x, y).RGBA()
			if absDiff8(r1, r2) > tolerance ||
				absDiff8(g1, g2) > tolerance ||
				absDiff8(b1, b2) > tolerance ||
				absDiff8(a1, a2) > tolerance {
				mismatched++
				if diffImg != nil {
					diffImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
				}
			} else if diffImg != nil {
				diffImg.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			}
		}
	}
	res.MismatchedPixels = mismatched
	if total > 0 {
		res.MismatchPct = 100.0 * float64(mismatched) / float64(total)
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
