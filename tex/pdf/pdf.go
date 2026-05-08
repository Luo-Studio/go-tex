// Package pdf renders a [layout.DisplayList] to a PDF document via
// codeberg.org/go-pdf/fpdf, with embedded KaTeX TTFs from tex/fonts.
//
// Two entry points:
//
//   - [Render] / [RenderTo] produce a standalone single-page PDF whose
//     page is sized exactly to the math + Padding. Suitable for
//     emitting a math expression as its own document.
//
//   - [DrawInto] draws onto an existing *fpdf.Fpdf at a caller-chosen
//     (x, y). Use this when embedding math into a larger PDF you are
//     already constructing — fonts are registered into the supplied
//     document on first use and reused thereafter.
//
// Coordinate conventions: fpdf and the layout DisplayList both use a
// top-left origin with Y growing downward, so coordinates pass through
// in the same orientation. Sizes are in points (1pt = 1/72 inch) by
// default.
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-tex/tex/fonts"
	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/path"
)

// Options configure the rendered PDF.
type Options struct {
	// FontSize is the points-per-em conversion; with FontSize=12 a
	// glyph rendered at scale 1.0 prints at 12pt. fpdf's SetFont
	// always interprets size as points regardless of the document's
	// unit, so this is independent of doc units.
	FontSize float64
	// Padding is added on every side of the bounding box, in the
	// document's units (points by default; mm if the embedding doc
	// was created with UnitStr="mm"). For DrawInto it shifts the math
	// inward from the supplied (x, y); set to 0 for flush placement.
	Padding float64
	// StrokeWidth is the line width for un-filled paths, in document units.
	StrokeWidth float64
}

// DefaultOptions returns FontSize=12, Padding=4, StrokeWidth=0.5 —
// reasonable defaults for math-as-its-own-document. For embedding,
// see [EmbedDefaults].
func DefaultOptions() Options {
	return Options{FontSize: 12, Padding: 4, StrokeWidth: 0.5}
}

// EmbedDefaults returns FontSize=12, Padding=0, StrokeWidth=0.5 —
// suitable for embedding into existing PDF documents where the caller
// controls the surrounding whitespace through (x, y) positioning.
func EmbedDefaults() Options {
	return Options{FontSize: 12, Padding: 0, StrokeWidth: 0.5}
}

// Size returns the rendered math's outer dimensions in points,
// assuming the embedding document also uses points (the case for
// [Render] / [RenderTo]). Padding is added on all four sides.
//
// For embedding into a non-point document (e.g. one created with
// UnitStr="mm"), use [SizeForDoc] instead — the conversion ratio
// between em-units and doc-units is only known at draw time.
func Size(dl layout.DisplayList, opts Options) (w, h float64) {
	return sizeWithEm(dl, opts, opts.FontSize)
}

// SizeForDoc returns the rendered math's outer dimensions in pdf's
// native unit (points, mm, ...). It accounts for the document's
// unit/point conversion ratio so that a 12pt expression embedded in
// an mm document reports its size in mm.
func SizeForDoc(pdf *fpdf.Fpdf, dl layout.DisplayList, opts Options) (w, h float64) {
	if pdf == nil {
		return Size(dl, opts)
	}
	return sizeWithEm(dl, opts, emInDocUnits(pdf, opts))
}

func sizeWithEm(dl layout.DisplayList, opts Options, em float64) (w, h float64) {
	w = dl.Width*em + 2*opts.Padding
	h = (dl.Height+dl.Depth)*em + 2*opts.Padding
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// emInDocUnits converts the point-valued FontSize into the embedding
// document's units. fpdf's GetConversionRatio returns user-units per
// point (1.0 for "pt", ~2.835 for "mm", etc.), so dividing point sizes
// by it gives doc-unit-equivalents.
func emInDocUnits(pdf *fpdf.Fpdf, opts Options) float64 {
	k := pdf.GetConversionRatio()
	if k == 0 {
		return opts.FontSize
	}
	return opts.FontSize / k
}

// Render generates a single-page PDF from the DisplayList and returns
// the raw bytes. The page size matches the math's bounding box +
// Padding.
func Render(dl layout.DisplayList, opts Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := RenderTo(&buf, dl, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderTo writes the PDF to w.
func RenderTo(w io.Writer, dl layout.DisplayList, opts Options) error {
	wPt, hPt := Size(dl, opts)
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "pt",
		Size:           fpdf.SizeType{Wd: wPt, Ht: hPt},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	if err := DrawInto(pdf, dl, 0, 0, opts); err != nil {
		return err
	}
	if err := pdf.Output(w); err != nil {
		return fmt.Errorf("pdf: output: %w", err)
	}
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("pdf: %w", err)
	}
	return nil
}

// DrawInto renders the DisplayList onto pdf with its top-left corner
// (after Padding) at page position (x, y) on the document's current
// page. The caller is responsible for constructing the document,
// adding pages, and setting margins. The math's outer extent is
// Size(dl, opts).
//
// On first use with a given *fpdf.Fpdf, the KaTeX TTFs are registered
// into that document's font table; subsequent calls on the same pdf
// reuse them. Call [RegisterFonts] explicitly if you want to control
// the timing.
func DrawInto(pdf *fpdf.Fpdf, dl layout.DisplayList, x, y float64, opts Options) error {
	if err := RegisterFonts(pdf); err != nil {
		return err
	}
	em := emInDocUnits(pdf, opts)
	for _, item := range dl.Items {
		switch v := item.(type) {
		case layout.GlyphPath:
			drawGlyph(pdf, v, x, y, opts, em)
		case layout.Rect:
			drawRect(pdf, v)
		case layout.Line:
			drawLine(pdf, v, x, y, opts, em)
		case layout.PathItem:
			drawPath(pdf, v, x, y, opts, em)
		}
	}
	return pdf.Error()
}

// RegisterFonts registers the KaTeX TTFs into pdf's font table. Safe
// to call multiple times: subsequent calls on the same pdf are no-ops.
func RegisterFonts(pdf *fpdf.Fpdf) error {
	if pdf == nil {
		return fmt.Errorf("pdf: nil *fpdf.Fpdf")
	}
	if _, done := registeredPDFs.LoadOrStore(pdf, true); done {
		return nil
	}
	for _, name := range fonts.Names() {
		b, err := ttfBytes(name)
		if err != nil {
			registeredPDFs.Delete(pdf)
			return err
		}
		// All fonts loaded as the "regular" style; bold / italic /
		// bold-italic variants are separate font IDs in the layout
		// package (e.g. "Main-Bold", "Math-BoldItalic"), so we don't
		// rely on fpdf's style flags.
		pdf.AddUTF8FontFromBytes(name, "", b)
	}
	if err := pdf.Error(); err != nil {
		registeredPDFs.Delete(pdf)
		return err
	}
	return nil
}

// registeredPDFs tracks which fpdf documents have already had the
// KaTeX font set installed. Keyed by *fpdf.Fpdf so multiple unrelated
// documents in the same process each register independently.
var registeredPDFs sync.Map

var (
	ttfCacheMu sync.Mutex
	ttfCache   = map[string][]byte{}
)

func ttfBytes(fontID string) ([]byte, error) {
	ttfCacheMu.Lock()
	defer ttfCacheMu.Unlock()
	if b, ok := ttfCache[fontID]; ok {
		return b, nil
	}
	b, err := fonts.TTF(fontID)
	if err != nil {
		return nil, err
	}
	ttfCache[fontID] = b
	return b, nil
}

func drawGlyph(pdf *fpdf.Fpdf, g layout.GlyphPath, ox, oy float64, opts Options, em float64) {
	if _, err := ttfBytes(g.Font); err != nil {
		// Unknown font (e.g. CJK-Regular) — skip silently rather than
		// failing the entire render.
		return
	}
	pts := opts.FontSize * g.Scale
	x := ox + g.X*em + opts.Padding
	y := oy + g.Y*em + opts.Padding
	pdf.SetFont(g.Font, "", pts)
	pdf.SetTextColor(int(g.Color.R), int(g.Color.G), int(g.Color.B))
	pdf.Text(x, y, string(g.CharCode))
}

func drawRect(pdf *fpdf.Fpdf, r layout.Rect) {
	if r.Width <= 0 || r.Height <= 0 {
		return
	}
	pdf.SetFillColor(int(r.Color.R), int(r.Color.G), int(r.Color.B))
	pdf.Rect(r.X, r.Y, r.Width, r.Height, "F")
}

func drawLine(pdf *fpdf.Fpdf, l layout.Line, ox, oy float64, opts Options, em float64) {
	t := l.Thickness * em
	if t < 1e-6 {
		t = 1e-6
	}
	w := l.Width * em
	x0 := ox + l.X*em + opts.Padding
	yc := oy + l.Y*em + opts.Padding
	if l.Dashed {
		pdf.SetDrawColor(int(l.Color.R), int(l.Color.G), int(l.Color.B))
		pdf.SetLineWidth(t)
		pdf.SetDashPattern([]float64{t * 3, t * 3}, 0)
		pdf.Line(x0, yc, x0+w, yc)
		pdf.SetDashPattern(nil, 0)
		return
	}
	// Solid line: draw as a filled rectangle from yc-t/2 to yc+t/2.
	y0 := yc - t/2
	pdf.SetFillColor(int(l.Color.R), int(l.Color.G), int(l.Color.B))
	pdf.Rect(x0, y0, w, t, "F")
}

func drawPath(pdf *fpdf.Fpdf, p layout.PathItem, ox, oy float64, opts Options, em float64) {
	x0 := ox + p.X*em + opts.Padding
	y0 := oy + p.Y*em + opts.Padding
	first := true
	for _, c := range p.Commands {
		ax := x0 + c.X*em
		ay := y0 + c.Y*em
		switch c.Kind {
		case path.KindMoveTo:
			pdf.MoveTo(ax, ay)
			first = false
		case path.KindLineTo:
			if first {
				pdf.MoveTo(ax, ay)
				first = false
				continue
			}
			pdf.LineTo(ax, ay)
		case path.KindCubicTo:
			cx1 := x0 + c.X1*em
			cy1 := y0 + c.Y1*em
			cx2 := x0 + c.X2*em
			cy2 := y0 + c.Y2*em
			pdf.CurveBezierCubicTo(cx1, cy1, cx2, cy2, ax, ay)
		case path.KindQuadTo:
			cx1 := x0 + c.X1*em
			cy1 := y0 + c.Y1*em
			pdf.CurveTo(cx1, cy1, ax, ay)
		case path.KindClose:
			pdf.ClosePath()
		}
	}
	if p.Fill {
		pdf.SetFillColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
		pdf.DrawPath("F")
		return
	}
	pdf.SetDrawColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
	pdf.SetLineWidth(opts.StrokeWidth)
	pdf.SetLineCapStyle("round")
	pdf.SetLineJoinStyle("round")
	pdf.DrawPath("D")
}
