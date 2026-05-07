// Package svg serializes a layout DisplayList to SVG. Mirrors upstream's
// ratex-svg crate. The default mode emits `<text>` glyph elements with
// KaTeX CSS font-family names; path-glyph mode (embedding TTF outlines)
// is gated on the font-loading port.
package svg

import (
	"fmt"
	"strings"

	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/path"
)

// Options controls the SVG output dimensions.
type Options struct {
	FontSize     float64 // user units per em
	Padding      float64 // around the bbox
	StrokeWidth  float64 // for unfilled paths
	EmbedGlyphs  bool    // requires TTF font loading; not yet supported
	FontDir      string  // path to KaTeX TTFs (only honoured when EmbedGlyphs is true)
}

// DefaultOptions returns FontSize=40, Padding=10, StrokeWidth=1.5 — the
// upstream ratex-svg defaults.
func DefaultOptions() Options {
	return Options{FontSize: 40, Padding: 10, StrokeWidth: 1.5}
}

// Render serialises a DisplayList to a self-contained SVG document string.
func Render(dl layout.DisplayList, opts Options) string {
	em := opts.FontSize
	pad := opts.Padding
	totalH := dl.Height + dl.Depth
	vbW := dl.Width*em + 2*pad
	vbH := totalH*em + 2*pad

	var body strings.Builder
	for _, item := range dl.Items {
		switch v := item.(type) {
		case layout.GlyphPath:
			emitGlyphText(&body, v, opts)
		case layout.Line:
			emitLine(&body, v, opts)
		case layout.Rect:
			emitRect(&body, v, opts)
		case layout.PathItem:
			emitPath(&body, v, opts)
		}
	}
	return wrapSVG(vbW, vbH, body.String())
}

func wrapSVG(vbW, vbH float64, body string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ` +
		fmtNum(vbW) + " " + fmtNum(vbH) + `" width="` +
		fmtNum(vbW) + `" height="` + fmtNum(vbH) + `">` + body + `</svg>`
}

func tx(em float64, opts Options) float64 { return em*opts.FontSize + opts.Padding }

func ty(em float64, opts Options) float64 { return em*opts.FontSize + opts.Padding }

func emitGlyphText(out *strings.Builder, g layout.GlyphPath, opts Options) {
	ch := string(g.CharCode)
	text := xmlEscape(ch)
	family, weight, style := katexFace(g.Font)
	fs := g.Scale * opts.FontSize
	if fs == 0 {
		fs = opts.FontSize
	}
	fmt.Fprintf(out,
		`<text x="%s" y="%s" font-family="%s" font-size="%s" font-weight="%s" font-style="%s" fill="%s" dominant-baseline="alphabetic">%s</text>`,
		fmtNum(tx(g.X, opts)), fmtNum(ty(g.Y, opts)),
		family, fmtNum(fs), weight, style, colorRGBA(g.Color), text,
	)
}

func emitLine(out *strings.Builder, l layout.Line, opts Options) {
	em := opts.FontSize
	x0 := tx(l.X, opts)
	yc := ty(l.Y, opts)
	t := l.Thickness * em
	if t < 1e-6 {
		t = 1e-6
	}
	w := l.Width * em
	stroke := colorRGBA(l.Color)
	if l.Dashed {
		fmt.Fprintf(out,
			`<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s" stroke-dasharray="%s %s"/>`,
			fmtNum(x0), fmtNum(yc), fmtNum(x0+w), fmtNum(yc),
			stroke, fmtNum(t), fmtNum(t*3), fmtNum(t*3),
		)
		return
	}
	y0 := yc - t/2
	fmt.Fprintf(out,
		`<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`,
		fmtNum(x0), fmtNum(y0), fmtNum(w), fmtNum(t), stroke,
	)
}

func emitRect(out *strings.Builder, r layout.Rect, opts Options) {
	em := opts.FontSize
	x0 := tx(r.X, opts)
	y0 := ty(r.Y, opts)
	w := r.Width * em
	h := r.Height * em
	fmt.Fprintf(out,
		`<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`,
		fmtNum(x0), fmtNum(y0), fmtNum(w), fmtNum(h), colorRGBA(r.Color),
	)
}

func emitPath(out *strings.Builder, p layout.PathItem, opts Options) {
	em := opts.FontSize
	x0 := tx(p.X, opts)
	y0 := ty(p.Y, opts)
	d := pathToD(x0, y0, em, p.Commands)
	if p.Fill {
		fmt.Fprintf(out, `<path d="%s" fill="%s" fill-rule="nonzero" stroke="none"/>`,
			d, colorRGBA(p.Color))
	} else {
		fmt.Fprintf(out, `<path d="%s" stroke="%s" stroke-width="%s" fill="none"/>`,
			d, colorRGBA(p.Color), fmtNum(opts.StrokeWidth))
	}
}

func pathToD(originX, originY, em float64, cmds []path.Command) string {
	var s strings.Builder
	for _, c := range cmds {
		switch c.Kind {
		case path.KindMoveTo:
			fmt.Fprintf(&s, "M%s %s", fmtNum(originX+c.X*em), fmtNum(originY+c.Y*em))
		case path.KindLineTo:
			fmt.Fprintf(&s, "L%s %s", fmtNum(originX+c.X*em), fmtNum(originY+c.Y*em))
		case path.KindCubicTo:
			fmt.Fprintf(&s, "C%s %s %s %s %s %s",
				fmtNum(originX+c.X1*em), fmtNum(originY+c.Y1*em),
				fmtNum(originX+c.X2*em), fmtNum(originY+c.Y2*em),
				fmtNum(originX+c.X*em), fmtNum(originY+c.Y*em))
		case path.KindQuadTo:
			fmt.Fprintf(&s, "Q%s %s %s %s",
				fmtNum(originX+c.X1*em), fmtNum(originY+c.Y1*em),
				fmtNum(originX+c.X*em), fmtNum(originY+c.Y*em))
		case path.KindClose:
			s.WriteByte('Z')
		}
	}
	return s.String()
}

// katexFace maps an internal font id to (family, weight, style) for
// SVG <text> elements, matching upstream's katex_face.
func katexFace(font string) (family, weight, style string) {
	switch font {
	case "Main-Regular":
		return "KaTeX_Main", "normal", "normal"
	case "Main-Bold":
		return "KaTeX_Main", "bold", "normal"
	case "Main-Italic":
		return "KaTeX_Main", "normal", "italic"
	case "Main-BoldItalic":
		return "KaTeX_Main", "bold", "italic"
	case "Math-Italic":
		return "KaTeX_Math", "normal", "italic"
	case "Math-BoldItalic":
		return "KaTeX_Math", "bold", "italic"
	case "AMS-Regular":
		return "KaTeX_AMS", "normal", "normal"
	case "Caligraphic-Regular":
		return "KaTeX_Caligraphic", "normal", "normal"
	case "Fraktur-Regular":
		return "KaTeX_Fraktur", "normal", "normal"
	case "Fraktur-Bold":
		return "KaTeX_Fraktur", "bold", "normal"
	case "SansSerif-Regular":
		return "KaTeX_SansSerif", "normal", "normal"
	case "SansSerif-Bold":
		return "KaTeX_SansSerif", "bold", "normal"
	case "SansSerif-Italic":
		return "KaTeX_SansSerif", "normal", "italic"
	case "Script-Regular":
		return "KaTeX_Script", "normal", "normal"
	case "Typewriter-Regular":
		return "KaTeX_Typewriter", "normal", "normal"
	case "Size1-Regular":
		return "KaTeX_Size1", "normal", "normal"
	case "Size2-Regular":
		return "KaTeX_Size2", "normal", "normal"
	case "Size3-Regular":
		return "KaTeX_Size3", "normal", "normal"
	case "Size4-Regular":
		return "KaTeX_Size4", "normal", "normal"
	case "CJK-Regular", "CJK-Fallback":
		return "sans-serif", "normal", "normal"
	case "Emoji-Fallback":
		return `Apple Color Emoji, "Segoe UI Emoji", "Noto Color Emoji", sans-serif`, "normal", "normal"
	}
	return "KaTeX_Main", "normal", "normal"
}

func colorRGBA(c layout.Color) string {
	a := float64(c.A) / 255.0
	return fmt.Sprintf("rgba(%d,%d,%d,%s)", c.R, c.G, c.B, fmtNum(a))
}

// fmtNum prints a float with up to 6 decimal places, trimming trailing
// zeros and trailing '.'. Matches upstream's fmt_num.
func fmtNum(n float64) string {
	s := fmt.Sprintf("%.6f", n)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
