// Package pdf renders a [layout.DisplayList] to a PDF document via
// codeberg.org/go-pdf/fpdf, with embedded KaTeX TTFs from tex/fonts.
//
// Coordinate conventions: fpdf and the layout DisplayList both use a
// top-left origin with Y growing downward, so coordinates pass through
// in the same orientation. Sizes are in points (1pt = 1/72 inch) by
// default; the page extent is sized to the DisplayList's bounding box
// + Padding so the math sits flush against the page edges, suitable
// for embedding into larger documents.
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
	// glyph rendered at scale 1.0 prints at 12pt.
	FontSize float64
	// Padding is added on every side of the bounding box.
	Padding float64
	// StrokeWidth is the line width for un-filled paths.
	StrokeWidth float64
}

// DefaultOptions returns FontSize=12, Padding=4, StrokeWidth=0.5 —
// reasonable defaults for math-in-document embedding.
func DefaultOptions() Options {
	return Options{FontSize: 12, Padding: 4, StrokeWidth: 0.5}
}

// Render generates a single-page PDF from the DisplayList and returns
// the raw bytes. The page size matches the math's bounding box +
// Padding, so the output can be cropped or directly embedded.
func Render(dl layout.DisplayList, opts Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := RenderTo(&buf, dl, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderTo writes the PDF to w.
func RenderTo(w io.Writer, dl layout.DisplayList, opts Options) error {
	wPt := dl.Width*opts.FontSize + 2*opts.Padding
	hPt := (dl.Height+dl.Depth)*opts.FontSize + 2*opts.Padding
	if wPt < 1 {
		wPt = 1
	}
	if hPt < 1 {
		hPt = 1
	}
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "pt",
		Size:           fpdf.SizeType{Wd: wPt, Ht: hPt},
	})
	// No page margins — math sits at (Padding, Padding) flush in the
	// requested page extent.
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	if err := registerFonts(pdf); err != nil {
		return err
	}

	for _, item := range dl.Items {
		switch v := item.(type) {
		case layout.GlyphPath:
			drawGlyph(pdf, v, opts)
		case layout.Rect:
			drawRect(pdf, v, opts)
		case layout.Line:
			drawLine(pdf, v, opts)
		case layout.PathItem:
			drawPath(pdf, v, opts)
		}
	}

	if err := pdf.Output(w); err != nil {
		return fmt.Errorf("pdf: output: %w", err)
	}
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("pdf: %w", err)
	}
	return nil
}

// registeredFonts tracks which TTFs have been loaded into the
// supplied Fpdf instance. fpdf's font registry is per-document, so we
// re-register on each call but only reload bytes from tex/fonts once.
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

func registerFonts(pdf *fpdf.Fpdf) error {
	for _, name := range fonts.Names() {
		b, err := ttfBytes(name)
		if err != nil {
			return err
		}
		// All fonts loaded as the "regular" style; bold / italic /
		// bold-italic variants are separate font IDs in the layout
		// package (e.g. "Main-Bold", "Math-BoldItalic"), so we don't
		// rely on fpdf's style flags.
		pdf.AddUTF8FontFromBytes(name, "", b)
	}
	return pdf.Error()
}

func drawGlyph(pdf *fpdf.Fpdf, g layout.GlyphPath, opts Options) {
	if _, err := ttfBytes(g.Font); err != nil {
		// Unknown font (e.g. CJK-Regular) — skip silently rather than
		// failing the entire render.
		return
	}
	em := opts.FontSize * g.Scale
	x := g.X*opts.FontSize + opts.Padding
	y := g.Y*opts.FontSize + opts.Padding
	pdf.SetFont(g.Font, "", em)
	pdf.SetTextColor(int(g.Color.R), int(g.Color.G), int(g.Color.B))
	pdf.Text(x, y, string(g.CharCode))
}

func drawRect(pdf *fpdf.Fpdf, r layout.Rect, opts Options) {
	if r.Width <= 0 || r.Height <= 0 {
		return
	}
	pdf.SetFillColor(int(r.Color.R), int(r.Color.G), int(r.Color.B))
	pdf.Rect(r.X, r.Y, r.Width, r.Height, "F")
}

func drawLine(pdf *fpdf.Fpdf, l layout.Line, opts Options) {
	em := opts.FontSize
	t := l.Thickness * em
	if t < 1e-6 {
		t = 1e-6
	}
	w := l.Width * em
	x0 := l.X*em + opts.Padding
	yc := l.Y*em + opts.Padding
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

func drawPath(pdf *fpdf.Fpdf, p layout.PathItem, opts Options) {
	em := opts.FontSize
	x0 := p.X*em + opts.Padding
	y0 := p.Y*em + opts.Padding
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
