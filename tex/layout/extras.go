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

// layoutSpacingNode emits a kern-width box for a fixed-size spacing node
// (\\quad, \\nobreakspace, \\ ). Widths from KaTeX fixed-spacing rules.
func layoutSpacingNode(text string, opts Options) *Box {
	q := opts.Metrics().Quad
	switch text {
	case `\quad`:
		return NewKern(1.0 * q)
	case `\qquad`:
		return NewKern(2.0 * q)
	case `\enspace`:
		return NewKern(0.5 * q)
	case `\thinspace`:
		return NewKern(3.0 / 18.0 * q)
	case `\medspace`:
		return NewKern(4.0 / 18.0 * q)
	case `\thickspace`:
		return NewKern(5.0 / 18.0 * q)
	case `\negthinspace`:
		return NewKern(-3.0 / 18.0 * q)
	case `\negmedspace`:
		return NewKern(-4.0 / 18.0 * q)
	case `\negthickspace`:
		return NewKern(-5.0 / 18.0 * q)
	case `\nobreakspace`, " ":
		return NewKern(0.25)
	case `\ `, `\\ `:
		return NewKern(0.25)
	}
	return NewEmpty()
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

// layoutOp lays out an Op (\sum, \int, \lim, \sin, …). Symbol ops emit
// the named glyph from Main-Regular (or Size1 for big ops); text ops
// (\lim, \sin, …) emit the function name in upright Main-Regular.
func layoutOp(op *parser.Op, opts Options) *Box {
	if op.Body != nil && len(op.Body) > 0 {
		return layoutExpression(op.Body, opts, true)
	}
	if op.Name == nil {
		return NewEmpty()
	}
	name := *op.Name
	if op.Symbol {
		return layoutSymbol(name, parser.ModeMath, opts)
	}
	// Text op: lay out the function name (drop the leading '\') as a
	// row of upright Main-Regular glyphs.
	if len(name) > 0 && name[0] == '\\' {
		name = name[1:]
	}
	children := make([]*Box, 0, len(name))
	for _, r := range name {
		if cm, ok := fontmetrics.LookupWithFallback(fontmetrics.FontMainRegular, r); ok {
			children = append(children, &Box{
				Width: cm.Width, Height: cm.Height, Depth: cm.Depth, Color: opts.Color,
				Content: Glyph{FontID: fontmetrics.FontMainRegular, CharCode: r},
			})
		}
	}
	return makeHBox(children, opts.Color)
}

func layoutOperatorName(op *parser.OperatorName, opts Options) *Box {
	textOpts := opts
	return layoutTextBody(op.Body, textOpts)
}

// accentChar maps an accent label to the glyph codepoint upstream's
// symbols table records for it. (These differ from the Unicode "modifier
// letter" forms — KaTeX uses the ASCII caret `^` for `\hat`, etc.)
var accentChar = map[string]rune{
	`\hat`:       '^',
	`\widehat`:   '^',
	`\check`:     0x02C7, // ˇ
	`\widecheck`: 0x02C7,
	`\tilde`:     '~',
	`\widetilde`: '~',
	`\acute`:     0x02CA, // ˊ
	`\grave`:     '`',
	`\dot`:       0x02D9,
	`\ddot`:      0x00A8,
	`\bar`:       0x02C9, // ˉ
	`\breve`:     0x02D8,
	`\vec`:       0x20D7,
	`\mathring`:  0x02DA,
}

// layoutAccent stacks a math accent glyph above the base.
func layoutAccent(a *parser.Accent, opts Options) *Box {
	base := layoutNode(a.Base, opts.WithStyle(opts.Style.Cramped()))
	cp, ok := accentChar[a.Label]
	if !ok {
		return base
	}
	cm, ok := fontmetrics.Lookup(fontmetrics.FontMainRegular, cp)
	if !ok {
		return base
	}
	// Skew of the base character — used to shift the accent centre by
	// upstream's accent layout (handle_accent in engine.rs).
	skew := baseSkew(a.Base, opts)
	accentBox := &Box{
		Width: cm.Width, Height: cm.Height, Depth: cm.Depth, Color: opts.Color,
		Content: Glyph{FontID: fontmetrics.FontMainRegular, CharCode: cp},
	}
	// KaTeX caps the accent's visible height contribution at 0.35em
	// (handle_accent_clearance in upstream engine.rs).
	visibleAccent := cm.Height
	if visibleAccent > 0.35 {
		visibleAccent = 0.35
	}
	height := base.Height + visibleAccent
	return &Box{
		Width:  base.Width,
		Height: height,
		Depth:  base.Depth,
		Color:  opts.Color,
		Content: Accent{
			Base:      base,
			AccentBox: accentBox,
			Skew:      skew,
		},
	}
}

// baseSkew returns the skew metric of the symbol that forms the accent's
// base, or 0 for non-symbol bases. Used by accent centring.
func baseSkew(n parser.Node, opts Options) float64 {
	var text string
	var mode parser.Mode
	switch v := n.(type) {
	case *parser.MathOrd:
		text, mode = v.Text, parser.ModeMath
	case *parser.OrdGroup:
		// Single-symbol ord group inherits the symbol's skew.
		if len(v.Body) == 1 {
			return baseSkew(v.Body[0], opts)
		}
		return 0
	default:
		return 0
	}
	r, _ := decodeFirstRune(text)
	resolved := resolveCodepoint(text, r, mode)
	font := selectFont(text, resolved, mode)
	if cm, ok := fontmetrics.LookupWithFallback(font, resolved); ok {
		return cm.Skew
	}
	return 0
}

func layoutAccentUnder(a *parser.AccentUnder, opts Options) *Box {
	base := layoutNode(a.Base, opts)
	return base
}

// layoutSqrt lays out `\sqrt[index]{body}`. Mirrors the simplified version
// of upstream's layout_radical for the no-stretch case (small bodies).
//
// For tall bodies upstream uses Size1..Size4 stacked surd glyphs; we use
// the Main-Regular U+221A glyph with the body height set to the inner
// height + clearance.
func layoutSqrt(s *parser.Sqrt, opts Options) *Box {
	body := layoutNode(s.Body, opts.WithStyle(opts.Style.Cramped()))
	rt := opts.Metrics().DefaultRuleThickness
	innerH := body.Height
	if innerH == 0 {
		innerH = opts.Metrics().XHeight
	}
	// Surd glyph metrics (U+221A in Main-Regular).
	surdM, ok := fontmetrics.Lookup(fontmetrics.FontMainRegular, 0x221A)
	surdW := 0.83334
	if ok {
		surdW = surdM.Width
	}
	w := surdW + body.Width
	h := innerH + 4*rt
	d := body.Depth
	return &Box{
		Width: w, Height: h, Depth: d, Color: opts.Color,
		Content: Radical{
			Body:          body,
			RuleThickness: rt,
			InnerHeight:   innerH,
			IndexOffset:   surdW,
		},
	}
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
