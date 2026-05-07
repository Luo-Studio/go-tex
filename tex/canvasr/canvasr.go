// Package canvasr renders a [layout.DisplayList] onto a
// [github.com/tdewolff/canvas] Canvas, using the embedded KaTeX TTFs
// for glyph outlines. The resulting Canvas can be rasterised to PNG /
// JPEG / etc. via canvas/renderers/rasterizer or written to SVG / PS /
// EPS via the corresponding renderer.
//
// Coordinate systems: the layout package uses SVG-style top-left origin
// with Y growing downward, in em units; canvas uses bottom-left origin
// with Y growing upward, in millimetres. We pass a Matrix that flips
// Y and scales em → mm to each draw call, so callers don't have to
// think about the conversion.
//
// PDF output for go-tex flows through fpdf in a separate package
// (tex/pdf), not through canvas, so this package targets the
// rasterizer + SVG renderers exclusively.
package canvasr

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/luo-studio/go-tex/tex/fonts"
	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/path"

	"github.com/tdewolff/canvas"
)

// Options configure the rendered output.
type Options struct {
	// FontSize is the user-units-per-em conversion. With Padding=10 and
	// FontSize=40 (the SVG defaults), the output dimensions match the
	// existing tex/svg renderer's pixel-equivalent layout.
	FontSize float64
	// Padding is added on every side of the bounding box.
	Padding float64
	// StrokeWidth is the SVG-equivalent stroke width for un-filled
	// path items (e.g. \angl, \cancel diagonals).
	StrokeWidth float64
}

// DefaultOptions returns the same FontSize/Padding/StrokeWidth combo
// as tex/svg.DefaultOptions, so a 1:1 swap produces equivalent output.
func DefaultOptions() Options {
	return Options{FontSize: 40, Padding: 10, StrokeWidth: 1.5}
}

// Render walks the DisplayList and paints it onto a fresh
// [canvas.Canvas]. The caller can then rasterise / write to whatever
// format. The returned canvas's dimensions are in user units (px-like)
// matching tex/svg's viewBox so callers can pass DPMM(1) to get a 1:1
// pixel image.
func Render(dl layout.DisplayList, opts Options) *canvas.Canvas {
	totalW := dl.Width*opts.FontSize + 2*opts.Padding
	totalH := (dl.Height+dl.Depth)*opts.FontSize + 2*opts.Padding
	if totalW < 1 {
		totalW = 1
	}
	if totalH < 1 {
		totalH = 1
	}
	c := canvas.New(totalW, totalH)
	ctx := canvas.NewContext(c)

	// Flip Y: layout uses top-left origin / Y-down; canvas uses
	// bottom-left origin / Y-up. Translate to top, then negate Y.
	ctx.SetCoordSystem(canvas.CartesianIV) // (0,0) at top-left, Y down

	for _, item := range dl.Items {
		switch v := item.(type) {
		case layout.GlyphPath:
			drawGlyph(ctx, v, opts)
		case layout.Rect:
			drawRect(ctx, v, opts)
		case layout.Line:
			drawLine(ctx, v, opts)
		case layout.PathItem:
			drawPath(ctx, v, opts)
		}
	}
	return c
}

// canvasFonts caches FontFamily objects keyed by layout font ID so we
// only LoadFont once per font-size/family combination.
var (
	familyMu sync.Mutex
	families = map[string]*canvas.FontFamily{}
)

func familyFor(fontID string) (*canvas.FontFamily, error) {
	familyMu.Lock()
	defer familyMu.Unlock()
	if fam, ok := families[fontID]; ok {
		return fam, nil
	}
	b, err := fonts.TTF(fontID)
	if err != nil {
		return nil, err
	}
	fam := canvas.NewFontFamily(fontID)
	if err := fam.LoadFont(b, 0, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("canvasr: load %s: %w", fontID, err)
	}
	families[fontID] = fam
	return fam, nil
}

func toCanvasColor(c layout.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

func drawGlyph(ctx *canvas.Context, g layout.GlyphPath, opts Options) {
	fam, err := familyFor(g.Font)
	if err != nil {
		// Unknown font (e.g. CJK-Regular fallback) — skip silently;
		// the rasterizer would otherwise panic.
		return
	}
	em := opts.FontSize * g.Scale
	face := fam.Face(em*ptPerUnit, toCanvasColor(g.Color))
	// Glyph baseline x is g.X*opts.FontSize + Padding, baseline y same.
	x := g.X*opts.FontSize + opts.Padding
	y := g.Y*opts.FontSize + opts.Padding
	t := canvas.NewTextLine(face, string(g.CharCode), canvas.Left)
	ctx.DrawText(x, y, t)
}

// canvas FontFace wants size in points; user units are 1:1 with
// "pixels" in our SVG output, so 1 unit ≈ 1 px ≈ 1pt at 72dpi → mm
// conversion below. Canvas's internal mm scale is irrelevant when we
// rasterise at DPMM(1) — the canvas just stores numbers.
const ptPerUnit = 1.0

func drawRect(ctx *canvas.Context, r layout.Rect, opts Options) {
	x := r.X
	y := r.Y
	w := r.Width
	h := r.Height
	if w <= 0 || h <= 0 {
		return
	}
	ctx.SetFillColor(toCanvasColor(r.Color))
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(x, y, canvas.Rectangle(w, h))
}

func drawLine(ctx *canvas.Context, l layout.Line, opts Options) {
	em := opts.FontSize
	t := l.Thickness * em
	if t < 1e-6 {
		t = 1e-6
	}
	w := l.Width * em
	x0 := l.X*em + opts.Padding
	yc := l.Y*em + opts.Padding
	if l.Dashed {
		ctx.SetStrokeColor(toCanvasColor(l.Color))
		ctx.SetStrokeWidth(t)
		ctx.SetDashes(0, t*3, t*3)
		ctx.MoveTo(x0, yc)
		ctx.LineTo(x0+w, yc)
		ctx.Stroke()
		ctx.SetDashes(0)
		return
	}
	// Solid: draw as a filled rectangle from yc - t/2 to yc + t/2.
	y0 := yc - t/2
	ctx.SetFillColor(toCanvasColor(l.Color))
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(x0, y0, canvas.Rectangle(w, t))
}

func drawPath(ctx *canvas.Context, p layout.PathItem, opts Options) {
	em := opts.FontSize
	x0 := p.X*em + opts.Padding
	y0 := p.Y*em + opts.Padding
	cp := buildCanvasPath(p.Commands, em)
	col := toCanvasColor(p.Color)
	if p.Fill {
		ctx.SetFillColor(col)
		ctx.SetStrokeColor(canvas.Transparent)
		ctx.DrawPath(x0, y0, cp)
		return
	}
	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(col)
	ctx.SetStrokeWidth(opts.StrokeWidth)
	ctx.SetStrokeCapper(canvas.RoundCap)
	ctx.SetStrokeJoiner(canvas.RoundJoin)
	ctx.DrawPath(x0, y0, cp)
}

// buildCanvasPath converts our internal path.Command slice into a
// *canvas.Path. Commands are in body-relative em with Y measured the
// same way as the surrounding DisplayList items (top-left origin,
// Y down).
func buildCanvasPath(cmds []path.Command, em float64) *canvas.Path {
	p := &canvas.Path{}
	for _, c := range cmds {
		switch c.Kind {
		case path.KindMoveTo:
			p.MoveTo(c.X*em, c.Y*em)
		case path.KindLineTo:
			p.LineTo(c.X*em, c.Y*em)
		case path.KindCubicTo:
			p.CubeTo(c.X1*em, c.Y1*em, c.X2*em, c.Y2*em, c.X*em, c.Y*em)
		case path.KindQuadTo:
			p.QuadTo(c.X1*em, c.Y1*em, c.X*em, c.Y*em)
		case path.KindClose:
			p.Close()
		}
	}
	return p
}
