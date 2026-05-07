// Package render rasterises an SVG document to a PNG image. Uses
// github.com/srwiley/oksvg for SVG parsing and rasterx for the actual
// vector raster. Pure-Go, no native deps.
//
// Pixel parity against upstream RaTeX's ratex-render is best-effort:
// oksvg / rasterx anti-aliasing differs slightly from the upstream
// `ab_glyph` rasteriser, so byte-identical PNGs are unlikely without a
// custom raster matching ab_glyph exactly. Where SVG inputs match,
// dimensions and overall ink placement match; AA edges may differ.
package render

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"regexp"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

var rgbaRE = regexp.MustCompile(`rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*[\d.]+\s*\)`)

// rgbaToRGB rewrites `rgba(r,g,b,a)` color literals to `rgb(r,g,b)` so
// oksvg can parse the SVG. The alpha channel is dropped.
func rgbaToRGB(s string) string {
	return rgbaRE.ReplaceAllString(s, "rgb($1,$2,$3)")
}

// PNG converts an SVG document string to a PNG byte slice. width/height
// are the output pixel dimensions; if zero, the SVG's intrinsic
// dimensions are used. The input SVG's `rgba()` color literals are
// rewritten to `rgb()` so oksvg accepts them.
func PNG(svg string, width, height int) ([]byte, error) {
	svg = rgbaToRGB(svg)
	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return nil, fmt.Errorf("render: parse svg: %w", err)
	}
	w := width
	h := height
	if w == 0 {
		w = int(icon.ViewBox.W)
	}
	if h == 0 {
		h = int(icon.ViewBox.H)
	}
	if w == 0 {
		w = 1
	}
	if h == 0 {
		h = 1
	}
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1.0)
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("render: encode png: %w", err)
	}
	return buf.Bytes(), nil
}
