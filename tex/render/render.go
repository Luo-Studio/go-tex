// Package render rasterises a [layout.DisplayList] to a PNG image,
// using tex/canvasr (DisplayList → tdewolff/canvas) and the embedded
// KaTeX TTFs from tex/fonts. Pure-Go, no native deps.
//
// Pixel parity against upstream RaTeX's ratex-render is best-effort:
// canvas's anti-aliasing differs from the upstream `ab_glyph`
// rasteriser, so byte-identical PNGs are unlikely. Visual output IS
// faithful — actual KaTeX glyphs from KaTeX_Main / KaTeX_Math / Size1-4
// are drawn (the previous oksvg+rasterx pipeline could not render
// `<text>` elements at all and produced glyph-less PNGs).
package render

import (
	"bytes"
	"fmt"
	"image/png"
	"io"

	"github.com/luo-studio/go-tex/tex/canvasr"
	"github.com/luo-studio/go-tex/tex/layout"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// PNG renders a DisplayList as a PNG byte slice using
// canvasr.DefaultOptions and 1px-per-user-unit resolution. width and
// height are not used (kept for backwards-compat with the previous
// SVG-based signature) — the canvas size derives from the DisplayList
// dimensions × FontSize + 2 × Padding.
func PNG(dl layout.DisplayList, opts ...Option) ([]byte, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	c := canvasr.Render(dl, cfg.canvasOpts)
	img := rasterizer.Draw(c, canvas.DPMM(cfg.scale), canvas.DefaultColorSpace)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("render: encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// PNGTo writes the PNG bytes to w.
func PNGTo(w io.Writer, dl layout.DisplayList, opts ...Option) error {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	c := canvasr.Render(dl, cfg.canvasOpts)
	img := rasterizer.Draw(c, canvas.DPMM(cfg.scale), canvas.DefaultColorSpace)
	return png.Encode(w, img)
}

// Option configures rendering. See [WithFontSize], [WithPadding],
// [WithStrokeWidth], and [WithScale].
type Option func(*config)

type config struct {
	canvasOpts canvasr.Options
	scale      float64
}

func defaultConfig() config {
	return config{
		canvasOpts: canvasr.DefaultOptions(),
		scale:      1,
	}
}

// WithFontSize overrides the user-units-per-em conversion (default 40,
// matching tex/svg's pixel-equivalent layout).
func WithFontSize(em float64) Option {
	return func(c *config) { c.canvasOpts.FontSize = em }
}

// WithPadding overrides the padding around the bounding box (default 10).
func WithPadding(pad float64) Option {
	return func(c *config) { c.canvasOpts.Padding = pad }
}

// WithStrokeWidth sets the stroke width for un-filled paths (default 1.5).
func WithStrokeWidth(w float64) Option {
	return func(c *config) { c.canvasOpts.StrokeWidth = w }
}

// WithScale multiplies the rasterisation resolution (default 1.0).
// Use 2.0 for 2× output (HiDPI), 0.5 for half-resolution thumbnails.
func WithScale(s float64) Option {
	return func(c *config) { c.scale = s }
}
