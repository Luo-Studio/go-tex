package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/parser"
)

// layoutTextBody lays out the body of a `\text{...}` node as a horizontal
// concatenation of glyph boxes in text mode (no atom-class glue).
func layoutTextBody(nodes []parser.Node, opts Options) *Box {
	if len(nodes) == 0 {
		return NewEmpty()
	}
	textOpts := opts
	children := make([]*Box, 0, len(nodes))
	for _, n := range nodes {
		switch v := n.(type) {
		case *parser.MathOrd:
			children = append(children, layoutSymbol(v.Text, parser.ModeText, textOpts))
		case *parser.TextOrd:
			children = append(children, layoutSymbol(v.Text, parser.ModeText, textOpts))
		case *parser.Atom:
			children = append(children, layoutSymbol(v.Text, parser.ModeText, textOpts))
		case *parser.Spacing:
			// Use the symbols-table spacing entry (\nobreakspace gives
			// width=0.25em in Main-Regular, etc.). Unknown spacing names
			// produce an empty box.
			cm, ok := fontmetrics.LookupWithFallback(fontmetrics.FontMainRegular, ' ')
			if !ok {
				children = append(children, NewEmpty())
				continue
			}
			children = append(children, &Box{Width: cm.Width, Height: cm.Height, Depth: cm.Depth, Color: opts.Color, Content: Glyph{FontID: fontmetrics.FontMainRegular, CharCode: ' '}})
		default:
			children = append(children, layoutNode(n, textOpts))
		}
	}
	return makeHBox(children, opts.Color)
}

// measurementToEm converts a parser.Measurement to em given the current
// style. Approximate; accurate enough for layout-dim parity.
func measurementToEm(m parser.Measurement, opts Options) float64 {
	switch m.Unit {
	case "em":
		return m.Number
	case "ex":
		return m.Number * 0.43055
	case "mu":
		return muToEm(m.Number, opts.Metrics().Quad)
	case "pt":
		return m.Number / opts.Metrics().PtPerEm
	case "pc":
		return m.Number * 12.0 / opts.Metrics().PtPerEm
	case "in":
		return m.Number * 72.27 / opts.Metrics().PtPerEm
	case "cm":
		return m.Number * 28.4527559 / opts.Metrics().PtPerEm
	case "mm":
		return m.Number * 2.84527559 / opts.Metrics().PtPerEm
	case "bp":
		return m.Number * 72.0 / 72.27 / opts.Metrics().PtPerEm
	}
	return m.Number
}

// layoutOp implements the simplest Op layout: emit the operator glyph
// (e.g. \int, \sum). Limits handling needs sup/sub from the parent
// SupSub, which we don't model here.
func layoutOp(op *parser.Op, opts Options) *Box {
	if op.Body != nil && len(op.Body) > 0 {
		// `\mathop{...}` — body is the content.
		return layoutExpression(op.Body, opts, true)
	}
	if op.Name != nil {
		return layoutSymbol(*op.Name, parser.ModeMath, opts)
	}
	return NewEmpty()
}

func layoutOperatorName(op *parser.OperatorName, opts Options) *Box {
	textOpts := opts
	return layoutTextBody(op.Body, textOpts)
}

// layoutAccent is a placeholder that returns the base unchanged. Real
// accent layout needs glyph widths for the accent character; we'll
// refine once that data is wired up.
func layoutAccent(a *parser.Accent, opts Options) *Box {
	base := layoutNode(a.Base, opts)
	return base
}

func layoutAccentUnder(a *parser.AccentUnder, opts Options) *Box {
	base := layoutNode(a.Base, opts)
	return base
}

// layoutSqrt is a placeholder that lays out the body and adds a small
// surd width on the left.
func layoutSqrt(s *parser.Sqrt, opts Options) *Box {
	body := layoutNode(s.Body, opts.WithStyle(opts.Style.Cramped()))
	// Approximate: extra surd width 0.5em.
	w := body.Width + 0.5
	h := body.Height + opts.Metrics().DefaultRuleThickness*2
	d := body.Depth
	return &Box{Width: w, Height: h, Depth: d, Color: opts.Color,
		Content: Radical{Body: body, RuleThickness: opts.Metrics().DefaultRuleThickness, InnerHeight: body.Height}}
}

// layoutLeftRight is a placeholder: just lay out the inner body. Real
// layout grows the left/right delimiters to the inner height.
func layoutLeftRight(lr *parser.LeftRight, opts Options) *Box {
	return layoutExpression(lr.Body, opts, true)
}

func layoutOverline(o *parser.Overline, opts Options) *Box {
	body := layoutNode(o.Body, opts.WithStyle(opts.Style.Cramped()))
	rt := opts.Metrics().DefaultRuleThickness
	return &Box{Width: body.Width, Height: body.Height + 4*rt, Depth: body.Depth, Color: opts.Color,
		Content: Overline{Body: body, RuleThickness: rt}}
}

func layoutUnderline(o *parser.Underline, opts Options) *Box {
	body := layoutNode(o.Body, opts)
	rt := opts.Metrics().DefaultRuleThickness
	return &Box{Width: body.Width, Height: body.Height, Depth: body.Depth + 4*rt, Color: opts.Color,
		Content: Underline{Body: body, RuleThickness: rt}}
}
