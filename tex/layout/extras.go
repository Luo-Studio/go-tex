package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/parser"
	"github.com/luo-studio/go-tex/tex/symbols"
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
			// Spacing in text mode: emit a kern (no glyph) — upstream's
			// SVG output drops the <text> element for whitespace.
			cm, ok := fontmetrics.LookupWithFallback(fontmetrics.FontMainRegular, ' ')
			if !ok {
				children = append(children, NewEmpty())
				continue
			}
			children = append(children, NewKern(cm.Width))
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
// the named glyph from Size1/Size2 (Size2 in displaystyle); text ops
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
		return layoutOpSymbol(name, opts)
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

// sizeToMaxHeight gives the fixed total height for \\big/\\Big/\\bigg/\\Bigg.
var sizeToMaxHeight = []float64{0, 1.2, 1.8, 2.4, 3.0}

// layoutDelimSizing handles \\big(, \\Bigl{, \\Bigm|, etc.
func layoutDelimSizing(d *parser.DelimSizing, opts Options) *Box {
	if d.Delim == "." || d.Delim == "" {
		return NewKern(0)
	}
	if _, ok := delimChar(d.Delim); !ok {
		return NewKern(0)
	}
	totalH := sizeToMaxHeight[1]
	if int(d.Size) >= 0 && int(d.Size) < len(sizeToMaxHeight) {
		totalH = sizeToMaxHeight[d.Size]
	}
	return makeStretchyDelim(d.Delim, totalH, opts)
}

// layoutOpSymbol lays out a symbol op (e.g. \\sum, \\int) using the
// Size1/Size2 font for the glyph (Size2 in displaystyle).
func layoutOpSymbol(name string, opts Options) *Box {
	info, ok := symbols.Lookup(name, symbols.ModeMath)
	cp := rune(0)
	if ok {
		cp = info.Codepoint
	}
	if cp == 0 {
		return layoutSymbol(name, parser.ModeMath, opts)
	}
	font := fontmetrics.FontSize1Regular
	if opts.Style.IsDisplay() {
		font = fontmetrics.FontSize2Regular
	}
	cm, found := fontmetrics.Lookup(font, cp)
	if !found {
		cm, found = fontmetrics.Lookup(fontmetrics.FontSize1Regular, cp)
		if found {
			font = fontmetrics.FontSize1Regular
		}
	}
	if !found {
		return layoutSymbol(name, parser.ModeMath, opts)
	}
	return &Box{
		Width: cm.Width + cm.Italic, Height: cm.Height, Depth: cm.Depth, Color: opts.Color,
		Content: Glyph{FontID: font, CharCode: cp},
	}
}

// accentChar maps an accent label to the glyph codepoint upstream's
// symbols table records for it. (These differ from the Unicode "modifier
// letter" forms — KaTeX uses the ASCII caret `^` for `\hat`, etc.)
var accentChar = map[string]rune{
	// Math-mode accents.
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
	// Text-mode accents — codepoints from upstream's symbols table.
	// These differ from the math-mode versions (e.g. \\^ -> U+02C6, while
	// math \\hat -> ASCII ^).
	`\'`:         0x02CA, // ˊ
	"\\`":        0x02CB, // ˋ
	`\^`:         0x02C6, // ˆ
	`\~`:         0x02DC, // ˜
	`\=`:         0x02C9, // ˉ
	`\u`:         0x02D8, // ˘
	`\.`:         0x02D9, // ˙
	`\"`:         0x00A8, // ¨
	`\r`:         0x02DA, // ˚
	`\H`:         0x02DD, // ˝
	`\v`:         0x02C7, // ˇ
	`\c`:         0x00B8, // ¸
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
	// Box width is at least 0.5em (matches upstream layout_accent's
	// `base_w = body_box.width.max(0.5)`).
	width := base.Width
	if width < 0.5 {
		width = 0.5
	}
	// Skew of the base character — used to shift the accent centre by
	// upstream's accent layout (handle_accent in engine.rs).
	skew := baseSkew(a.Base, opts)
	accentBox := &Box{
		Width: cm.Width, Height: cm.Height, Depth: cm.Depth, Color: opts.Color,
		Content: Glyph{FontID: fontmetrics.FontMainRegular, CharCode: cp},
	}
	xHeight := opts.Metrics().XHeight
	visCap := cm.Height
	if visCap > 0.35 {
		visCap = 0.35
	}
	// Mirrors upstream handle_accent_clearance for non-stretchy, non-arrow,
	// non-nested-Accent bodies.
	var clearance float64
	if a.Label == `\bar` || a.Label == `\=` {
		// Macron special case: clearance = body.height (less the
		// 0.12em macron adjustment applied below).
		clearance = base.Height
	} else {
		katexPos := base.Height - xHeight
		if katexPos < 0 {
			katexPos = 0
		}
		correction := cm.Height - visCap
		if correction < 0 {
			correction = 0
		}
		clearance = katexPos + correction
	}
	clearance += cm.Depth
	if a.Label == `\bar` || a.Label == `\=` {
		clearance -= 0.12
		if clearance < 0 {
			clearance = 0
		}
	}
	accentVisualTop := clearance + visCap
	height := 0.0
	switch a.Label {
	case `\hat`, `\bar`, `\=`, `\dot`, `\ddot`:
		const strut = 0.78056
		height = accentVisualTop
		if strut > height {
			height = strut
		}
	default:
		height = base.Height
		if accentVisualTop > height {
			height = accentVisualTop
		}
	}
	return &Box{
		Width:  width,
		Height: height,
		Depth:  base.Depth,
		Color:  opts.Color,
		Content: Accent{
			Base:       base,
			AccentBox:  accentBox,
			Skew:       skew,
			Clearance:  clearance,
			Correction: cm.Height - visCap,
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

// layoutLeftRight: lay out the inner, then pick stretchy delimiters
// from the Size1-Size4 family to wrap it. Mirrors upstream's
// layout_left_right + make_stretchy_delim.
func layoutLeftRight(lr *parser.LeftRight, opts Options) *Box {
	inner := layoutExpression(lr.Body, opts, true)
	totalH := leftRightDelimTotalHeight(inner, opts)
	left := makeStretchyDelim(lr.Left, totalH, opts)
	right := makeStretchyDelim(lr.Right, totalH, opts)
	w := left.Width + inner.Width + right.Width
	h := max64(max64(left.Height, right.Height), inner.Height)
	d := max64(max64(left.Depth, right.Depth), inner.Depth)
	return &Box{
		Width: w, Height: h, Depth: d, Color: opts.Color,
		Content: LeftRight{Left: left, Inner: inner, Right: right},
	}
}

func leftRightDelimTotalHeight(inner *Box, opts Options) float64 {
	m := opts.Metrics()
	axis := m.AxisHeight
	maxDist := inner.Height - axis
	if inner.Depth+axis > maxDist {
		maxDist = inner.Depth + axis
	}
	delimFactor := 901.0
	delimExtend := 5.0 / m.PtPerEm
	fromFormula := maxDist / 500 * delimFactor
	if 2*maxDist-delimExtend > fromFormula {
		fromFormula = 2*maxDist - delimExtend
	}
	if inner.Height+inner.Depth > fromFormula {
		return inner.Height + inner.Depth
	}
	return fromFormula
}

var delimFontSequence = []string{
	fontmetrics.FontMainRegular,
	fontmetrics.FontSize1Regular,
	fontmetrics.FontSize2Regular,
	fontmetrics.FontSize3Regular,
	fontmetrics.FontSize4Regular,
}

// delimChar maps a left/right delimiter spelling to its glyph codepoint.
func delimChar(delim string) (rune, bool) {
	switch delim {
	case "(":
		return '(', true
	case ")":
		return ')', true
	case "[", `\lbrack`:
		return '[', true
	case "]", `\rbrack`:
		return ']', true
	case `\{`, `\lbrace`:
		return '{', true
	case `\}`, `\rbrace`:
		return '}', true
	case "/":
		return '/', true
	case `\backslash`:
		return '\\', true
	case "|", `\vert`, `\lvert`, `\rvert`:
		return '|', true
	case `\|`, `\Vert`, `\lVert`, `\rVert`:
		return 0x2225, true // ‖
	case `\langle`, "<", `\lt`:
		return 0x27E8, true // ⟨
	case `\rangle`, ">", `\gt`:
		return 0x27E9, true
	case `\lfloor`:
		return 0x230A, true
	case `\rfloor`:
		return 0x230B, true
	case `\lceil`:
		return 0x2308, true
	case `\rceil`:
		return 0x2309, true
	case `\uparrow`:
		return 0x2191, true
	case `\downarrow`:
		return 0x2193, true
	case `\updownarrow`:
		return 0x2195, true
	case `\Uparrow`:
		return 0x21D1, true
	case `\Downarrow`:
		return 0x21D3, true
	case `\Updownarrow`:
		return 0x21D5, true
	case ".", "":
		return 0, false
	}
	return 0, false
}

func makeStretchyDelim(delim string, totalH float64, opts Options) *Box {
	if delim == "." || delim == "" {
		return NewKern(0)
	}
	cp, ok := delimChar(delim)
	if !ok {
		return NewKern(0)
	}
	bestFont := fontmetrics.FontMainRegular
	bestW, bestH, bestD := 0.4, 0.7, 0.2
	for _, font := range delimFontSequence {
		if cm, ok := fontmetrics.Lookup(font, cp); ok {
			bestFont = font
			bestW = cm.Width
			bestH = cm.Height
			bestD = cm.Depth
			if bestH+bestD >= totalH {
				break
			}
		}
	}
	return &Box{
		Width: bestW, Height: bestH, Depth: bestD, Color: opts.Color,
		Content: Glyph{FontID: bestFont, CharCode: cp},
	}
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
