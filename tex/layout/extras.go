package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/parser"
	"github.com/luo-studio/go-tex/tex/path"
	"github.com/luo-studio/go-tex/tex/symbols"
)

// layoutTextBody lays out the body of a `\text{...}` node as a horizontal
// concatenation of glyph boxes in text mode (no atom-class glue).
//
// If opts.InterGlyphKernEm > 0, inserts a kern between adjacent glyph
// children (matches upstream layout_with_font's tracking adjustment for
// \href / \url monospace alignment).
func layoutTextBody(nodes []parser.Node, opts Options) *Box {
	if len(nodes) == 0 {
		return NewEmpty()
	}
	textOpts := opts
	children := make([]*Box, 0, len(nodes))
	kern := opts.InterGlyphKernEm
	addKern := func() {
		if kern > 0 && len(children) > 0 {
			children = append(children, NewKern(kern))
		}
	}
	for _, n := range nodes {
		switch v := n.(type) {
		case *parser.MathOrd:
			addKern()
			children = append(children, layoutSymbol(v.Text, parser.ModeText, textOpts))
		case *parser.TextOrd:
			addKern()
			children = append(children, layoutSymbol(v.Text, parser.ModeText, textOpts))
		case *parser.Atom:
			addKern()
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
		return m.Number * opts.Metrics().XHeight
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
		body := layoutExpression(op.Body, opts, true)
		// User-defined \mathop{content} (e.g. \vcentcolon): center the
		// content on the math axis via a RaiseBox so the glyph
		// physically moves up. Mirrors upstream layout_op.
		axis := opts.Metrics().AxisHeight
		delta := (body.Height-body.Depth)/2 - axis
		if delta > 0.001 || delta < -0.001 {
			raise := -delta
			h := body.Height + raise
			if h < 0 {
				h = 0
			}
			d := body.Depth - raise
			if d < 0 {
				d = 0
			}
			return &Box{
				Width: body.Width, Height: h, Depth: d, Color: opts.Color,
				Content: RaiseBox{Body: body, Shift: raise},
			}
		}
		return body
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
	// Box width = base.Width (upstream layout_accent uses body_box.width
	// for the box; the 0.5em floor only applies to internal SVG path
	// computation).
	width := base.Width
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
		// Multi-symbol ord group: take the LAST child's skew (mirrors
		// upstream glyph_skew on an HBox).
		if len(v.Body) == 0 {
			return 0
		}
		return baseSkew(v.Body[len(v.Body)-1], opts)
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
// of upstream's layout_radical.
func layoutSqrt(s *parser.Sqrt, opts Options) *Box {
	body := layoutNode(s.Body, opts.WithStyle(opts.Style.Cramped()))
	rt := opts.Metrics().DefaultRuleThickness
	innerH := body.Height
	if innerH == 0 {
		innerH = opts.Metrics().XHeight
	}
	surdM, ok := fontmetrics.Lookup(fontmetrics.FontMainRegular, 0x221A)
	surdW := 0.83334
	if ok {
		surdW = surdM.Width
	}
	var index *Box
	indexScale := 0.5 // scriptscript style
	if s.Index != nil {
		idxOpts := opts.WithStyle(6) // ScriptScript
		index = layoutNode(s.Index, idxOpts)
	}
	w := surdW + body.Width
	if index != nil {
		// Index sits above-left and may push out the surd start; for
		// matching dim parity, reserve max(0, index.Width*scale - 0.5em).
		offset := index.Width*indexScale - surdW + 0.3
		if offset > 0 {
			w += offset
		}
	}
	h := innerH + 4*rt
	d := body.Depth
	return &Box{
		Width: w, Height: h, Depth: d, Color: opts.Color,
		Content: Radical{
			Body:          body,
			Index:         index,
			RuleThickness: rt,
			InnerHeight:   innerH,
			IndexOffset:   surdW,
			IndexScale:    indexScale,
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
	// Upstream: height = body.height + 3*rule.
	return &Box{Width: body.Width, Height: body.Height + 3*rt, Depth: body.Depth, Color: opts.Color,
		Content: Overline{Body: body, RuleThickness: rt}}
}

func layoutUnderline(o *parser.Underline, opts Options) *Box {
	body := layoutNode(o.Body, opts)
	rt := opts.Metrics().DefaultRuleThickness
	return &Box{Width: body.Width, Height: body.Height, Depth: body.Depth + 3*rt, Color: opts.Color,
		Content: Underline{Body: body, RuleThickness: rt}}
}

// layoutHref handles \href{url}{body} and \url{addr}: render body in
// blue with slight tracking, wrapped in an underline.
func layoutHref(body []parser.Node, opts Options) *Box {
	linkColor := Color{0, 0, 255, 255}
	bodyOpts := opts.WithColor(linkColor)
	bodyOpts.InterGlyphKernEm = 0.024
	inner := layoutExpression(body, bodyOpts, true)
	rt := opts.Metrics().DefaultRuleThickness
	return &Box{
		Width:  inner.Width,
		Height: inner.Height,
		Depth:  inner.Depth + 3*rt,
		Color:  linkColor,
		Content: Underline{
			Body:          inner,
			RuleThickness: rt,
		},
	}
}

// genfracDelimHeight returns the target stretchy delim height for
// \binom/\brace/\brack/\dbinom (KaTeX HTML Rule 15e). Uses delim1 in
// display, delim2 in text/script — and Script's delim2 in scriptscript.
func genfracDelimHeight(opts Options) float64 {
	m := opts.Metrics()
	if opts.Style.IsDisplay() {
		return m.Delim1
	}
	// scriptscript falls back to script's delim2.
	// (We don't enumerate ScriptScript here because all metrics tables
	// share the same delim values up to the size index — this matches
	// upstream's logic of switching style metrics rather than the
	// already-applied size.)
	return m.Delim2
}

// layoutFramed handles \fbox / \fcolorbox / \colorbox: padded body
// with optional bg fill + 4-rule border. Mirrors upstream layout_enclose.
func layoutFramed(e *parser.Enclose, opts Options) *Box {
	m := opts.Metrics()
	padding := 3.0 / m.PtPerEm
	borderThick := 0.4 / m.PtPerEm
	hasBorder := e.Label == `\fbox` || e.Label == `\fcolorbox`

	inner := layoutNode(e.Body, opts)
	outerPad := padding
	if hasBorder {
		outerPad += borderThick
	}
	w := inner.Width + 2*outerPad
	h := inner.Height + outerPad
	d := inner.Depth + outerPad

	bg := parseColorPtr(e.BackgroundColor)
	bc := Black
	if e.BorderColor != nil {
		if c, ok := parseColor(*e.BorderColor); ok {
			bc = c
		}
	}
	return &Box{
		Width: w, Height: h, Depth: d, Color: opts.Color,
		Content: Framed{
			Body:            inner,
			Padding:         padding,
			BorderThickness: borderThick,
			HasBorder:       hasBorder,
			BgColor:         bg,
			BorderColor:     bc,
		},
	}
}

// parseColor / parseColorPtr resolve a CSS-style color name or hex
// string. Currently a stub: only handles a few names + #RGB / #RRGGBB.
func parseColor(s string) (Color, bool) {
	if s == "" {
		return Black, false
	}
	switch s {
	case "black":
		return Color{0, 0, 0, 255}, true
	case "white":
		return Color{255, 255, 255, 255}, true
	case "red":
		return Color{255, 0, 0, 255}, true
	case "green":
		return Color{0, 128, 0, 255}, true
	case "blue":
		return Color{0, 0, 255, 255}, true
	case "yellow":
		return Color{255, 255, 0, 255}, true
	case "cyan", "aqua":
		return Color{0, 255, 255, 255}, true
	case "magenta", "fuchsia":
		return Color{255, 0, 255, 255}, true
	case "orange":
		return Color{255, 165, 0, 255}, true
	case "purple":
		return Color{128, 0, 128, 255}, true
	case "gray", "grey":
		return Color{128, 128, 128, 255}, true
	case "lime":
		return Color{0, 255, 0, 255}, true
	case "navy":
		return Color{0, 0, 128, 255}, true
	case "teal":
		return Color{0, 128, 128, 255}, true
	case "olive":
		return Color{128, 128, 0, 255}, true
	case "maroon":
		return Color{128, 0, 0, 255}, true
	case "silver":
		return Color{192, 192, 192, 255}, true
	}
	if len(s) >= 4 && s[0] == '#' {
		hex := s[1:]
		parseHex := func(c byte) (uint8, bool) {
			switch {
			case c >= '0' && c <= '9':
				return c - '0', true
			case c >= 'a' && c <= 'f':
				return c - 'a' + 10, true
			case c >= 'A' && c <= 'F':
				return c - 'A' + 10, true
			}
			return 0, false
		}
		var rgb [3]uint8
		ok := true
		switch len(hex) {
		case 3:
			for i := 0; i < 3; i++ {
				v, vok := parseHex(hex[i])
				if !vok {
					ok = false
					break
				}
				rgb[i] = v*16 + v
			}
		case 6:
			for i := 0; i < 3; i++ {
				hi, ok1 := parseHex(hex[2*i])
				lo, ok2 := parseHex(hex[2*i+1])
				if !ok1 || !ok2 {
					ok = false
					break
				}
				rgb[i] = hi*16 + lo
			}
		default:
			ok = false
		}
		if ok {
			return Color{rgb[0], rgb[1], rgb[2], 255}, true
		}
	}
	return Black, false
}

func parseColorPtr(s *string) *Color {
	if s == nil {
		return nil
	}
	c, ok := parseColor(*s)
	if !ok {
		return nil
	}
	return &c
}

// sizingMultipliers maps KaTeX sizing keyword index (1-11) to a font
// scale ratio. Mirrors upstream layout_sizing.
var sizingMultipliers = []float64{
	1.0,   // 0 (unused)
	0.5,   // \tiny
	0.6,   // \sixptsize
	0.7,   // \scriptsize
	0.8,   // \footnotesize
	0.9,   // \small
	1.0,   // \normalsize
	1.2,   // \large
	1.44,  // \Large
	1.728, // \LARGE
	2.074, // \huge
	2.488, // \Huge
}

// layoutSizing handles \tiny, \scriptsize, ..., \Huge — sizing
// keywords. Builds inner at textstyle and wraps in a Scaled box if the
// ratio differs from current.
func layoutSizing(s *parser.Sizing, opts Options) *Box {
	if int(s.Size) <= 0 || int(s.Size) >= len(sizingMultipliers) {
		return layoutExpression(s.Body, opts, true)
	}
	mult := sizingMultipliers[s.Size]
	innerOpts := opts.WithStyle(opts.Style.Text())
	inner := layoutExpression(s.Body, innerOpts, true)
	ratio := mult / opts.SizeMultiplier()
	if ratio > 0.999 && ratio < 1.001 {
		return inner
	}
	return &Box{
		Width:  inner.Width * ratio,
		Height: inner.Height * ratio,
		Depth:  inner.Depth * ratio,
		Color:  opts.Color,
		Content: Scaled{
			Body:       inner,
			ChildScale: ratio,
		},
	}
}

// isSingleCharBody mirrors KaTeX isCharacterBox: single Atom/MathOrd/
// TextOrd, possibly wrapped in a 1-element OrdGroup or Styling.
func isSingleCharBody(n parser.Node) bool {
	switch v := n.(type) {
	case *parser.OrdGroup:
		return len(v.Body) == 1 && isSingleCharBody(v.Body[0])
	case *parser.Styling:
		return len(v.Body) == 1 && isSingleCharBody(v.Body[0])
	case *parser.Atom, *parser.MathOrd, *parser.TextOrd:
		return true
	}
	return false
}

// layoutCancel handles \cancel{body} (`/`), \bcancel{body} (`\`),
// \xcancel{body} (`X`), \sout{body} (`-`). Mirrors upstream layout_cancel.
//
// Single-character bodies use vertical padding (line corner-to-corner of
// h+d+0.4 box); multi-character bodies use horizontal padding (line
// extends 0.2em past each edge). \sout draws a horizontal line at
// -0.5*xHeight regardless.
func layoutCancel(label string, node parser.Node, opts Options) *Box {
	inner := layoutNode(node, opts)
	w := inner.Width
	if w < 0.01 {
		w = 0.01
	}
	h, d := inner.Height, inner.Depth

	single := isSingleCharBody(node)
	var vPad, hPad float64
	switch {
	case label == `\sout`:
		// no padding
	case single:
		vPad = 0.2
	default:
		hPad = 0.2
	}

	var cmds []path.Command
	switch label {
	case `\cancel`:
		cmds = []path.Command{
			path.MoveTo(-hPad, d+vPad),
			path.LineTo(w+hPad, -h-vPad),
		}
	case `\bcancel`:
		cmds = []path.Command{
			path.MoveTo(-hPad, -h-vPad),
			path.LineTo(w+hPad, d+vPad),
		}
	case `\xcancel`:
		cmds = []path.Command{
			path.MoveTo(-hPad, d+vPad),
			path.LineTo(w+hPad, -h-vPad),
			path.MoveTo(-hPad, -h-vPad),
			path.LineTo(w+hPad, d+vPad),
		}
	case `\sout`:
		midY := -0.5 * opts.Metrics().XHeight
		cmds = []path.Command{
			path.MoveTo(0, midY),
			path.LineTo(w, midY),
		}
	}

	lineW := w + 2*hPad
	lineH := h + vPad
	lineD := d + vPad
	lineBox := &Box{
		Width: lineW, Height: lineH, Depth: lineD, Color: opts.Color,
		Content: SvgPath{Commands: cmds, Fill: false},
	}
	// Body kerned back to overlap the line box: kern = -(line_w - h_pad).
	kern := -(lineW - hPad)
	bodyShifted := makeHBox([]*Box{NewKern(kern), inner}, opts.Color)
	return &Box{
		Width: w, Height: h, Depth: d, Color: opts.Color,
		Content: HBox{Children: []*Box{lineBox, bodyShifted}},
	}
}

// layoutAngl is `\angl{body}` — actuarial angle: a horizontal roof
// above the body and a vertical bar on the right (KaTeX style).
//
// Mirrors upstream layout_angl: roof at body.height + 0.1 above
// baseline, right bar runs from roof down to body.depth + 0.3 below
// baseline.
func layoutAngl(node parser.Node, opts Options) *Box {
	inner := layoutNode(node, opts)
	w := inner.Width
	if w < 0.3 {
		w = 0.3
	}
	clearance := 0.1
	arcH := inner.Height + clearance
	cmds := []path.Command{
		path.MoveTo(0, -arcH),
		path.LineTo(w, -arcH),
		path.LineTo(w, inner.Depth+0.3),
	}
	return &Box{
		Width:  w,
		Height: arcH,
		Depth:  inner.Depth,
		Color:  opts.Color,
		Content: Angl{
			PathCommands: cmds,
			Body:         inner,
		},
	}
}
