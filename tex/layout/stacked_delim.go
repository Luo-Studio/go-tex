package layout

import (
	"fmt"

	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/path"
)

// stacked_delim.go: Tall stretchy delimiters built from KaTeX SVG paths.
// Mirrors upstream stacked_delim.rs.

const stackedLap = 0.008

type stackDelimKind int

const (
	skBraceOpen stackDelimKind = iota
	skBraceClose
	skBracketOpen
	skBracketClose
	skParenOpen
	skParenClose
	skLFloor
	skRFloor
	skLCeil
	skRCeil
	skGroupOpen
	skGroupClose
	skNone
)

func classifyStackable(delim string) stackDelimKind {
	d := delim
	switch d {
	case "{":
		d = `\{`
	case "}":
		d = `\}`
	}
	switch d {
	case `\{`, `\lbrace`:
		return skBraceOpen
	case `\}`, `\rbrace`:
		return skBraceClose
	case "[", `\lbrack`:
		return skBracketOpen
	case "]", `\rbrack`:
		return skBracketClose
	case "(", `\lparen`:
		return skParenOpen
	case ")", `\rparen`:
		return skParenClose
	case `\lfloor`, "⌊":
		return skLFloor
	case `\rfloor`, "⌋":
		return skRFloor
	case `\lceil`, "⌈":
		return skLCeil
	case `\rceil`, "⌉":
		return skRCeil
	case `\lgroup`, "⟮":
		return skGroupOpen
	case `\rgroup`, "⟯":
		return skGroupClose
	}
	return skNone
}

func isStackNeverDelim(delim string) bool {
	switch delim {
	case "<", ">", `\langle`, `\rangle`, "/", `\backslash`, `\lt`, `\gt`,
		"⟨", "⟩":
		return true
	}
	return false
}

func tallDelimSVGPath(label string, midTh int64) string {
	switch label {
	case "lbrack":
		return fmt.Sprintf(
			"M403 1759 V84 H666 V0 H319 V1759 v%d v1759 h347 v-84 H403z M403 1759 V0 H319 V1759 v%d v1759 h84z", midTh, midTh)
	case "rbrack":
		return fmt.Sprintf(
			"M347 1759 V0 H0 V84 H263 V1759 v%d v1759 H0 v84 H347z M347 1759 V0 H263 V1759 v%d v1759 h84z", midTh, midTh)
	case "lfloor":
		return fmt.Sprintf(
			"M319 602 V0 H403 V602 v%d v1715 h263 v84 H319z M319 602 V0 H403 V602 v%d v1715 H319z", midTh, midTh)
	case "rfloor":
		return fmt.Sprintf(
			"M319 602 V0 H403 V602 v%d v1799 H0 v-84 H319z M319 602 V0 H403 V602 v%d v1715 H319z", midTh, midTh)
	case "lceil":
		return fmt.Sprintf(
			"M403 1759 V84 H666 V0 H319 V1759 v%d v602 h84z M403 1759 V0 H319 V1759 v%d v602 h84z", midTh, midTh)
	case "rceil":
		return fmt.Sprintf(
			"M347 1759 V0 H0 V84 H263 V1759 v%d v602 h84z M347 1759 V0 h-84 V1759 v%d v602 h84z", midTh, midTh)
	case "lparen":
		m84 := midTh + 84
		m92 := midTh + 92
		return fmt.Sprintf(
			"M863,9c0,-2,-2,-5,-6,-9c0,0,-17,0,-17,0c-12.7,0,-19.3,0.3,-20,1c-5.3,5.3,-10.3,11,-15,17c-242.7,294.7,-395.3,682,-458,1162c-21.3,163.3,-33.3,349,-36,557 l0,%dc0.2,6,0,26,0,60c2,159.3,10,310.7,24,454c53.3,528,210,949.7,470,1265c4.7,6,9.7,11.7,15,17c0.7,0.7,7,1,19,1c0,0,18,0,18,0c4,-4,6,-7,6,-9c0,-2.7,-3.3,-8.7,-10,-18c-135.3,-192.7,-235.5,-414.3,-300.5,-665c-65,-250.7,-102.5,-544.7,-112.5,-882c-2,-104,-3,-167,-3,-189 l0,-%dc0,-162.7,5.7,-314,17,-454c20.7,-272,63.7,-513,129,-723c65.3,-210,155.3,-396.3,270,-559c6.7,-9.3,10,-15.3,10,-18z",
			m84, m92)
	case "rparen":
		m9 := midTh + 9
		m144 := midTh + 144
		return fmt.Sprintf(
			"M76,0c-16.7,0,-25,3,-25,9c0,2,2,6.3,6,13c21.3,28.7,42.3,60.3,63,95c96.7,156.7,172.8,332.5,228.5,527.5c55.7,195,92.8,416.5,111.5,664.5c11.3,139.3,17,290.7,17,454c0,28,1.7,43,3.3,45l0,%dc-3,4,-3.3,16.7,-3.3,38c0,162,-5.7,313.7,-17,455c-18.7,248,-55.8,469.3,-111.5,664c-55.7,194.7,-131.8,370.3,-228.5,527c-20.7,34.7,-41.7,66.3,-63,95c-2,3.3,-4,7,-6,11c0,7.3,5.7,11,17,11c0,0,11,0,11,0c9.3,0,14.3,-0.3,15,-1c5.3,-5.3,10.3,-11,15,-17c242.7,-294.7,395.3,-681.7,458,-1161c21.3,-164.7,33.3,-350.7,36,-558 l0,-%dc-2,-159.3,-10,-310.7,-24,-454c-53.3,-528,-210,-949.7,-470,-1265c-4.7,-6,-9.7,-11.7,-15,-17c-0.7,-0.7,-6.7,-1,-18,-1z",
			m9, m144)
	}
	return ""
}

func svgLabelForStack(kind stackDelimKind) string {
	switch kind {
	case skBracketOpen:
		return "lbrack"
	case skBracketClose:
		return "rbrack"
	case skParenOpen:
		return "lparen"
	case skParenClose:
		return "rparen"
	case skLFloor:
		return "lfloor"
	case skRFloor:
		return "rfloor"
	case skLCeil:
		return "lceil"
	case skRCeil:
		return "rceil"
	}
	return ""
}

func widthEmForStackLabel(label string) float64 {
	switch label {
	case "lparen", "rparen":
		return 0.875
	}
	return 0.667
}

func shiftPathY(cmds []path.Command, dy float64) []path.Command {
	out := make([]path.Command, len(cmds))
	for i, c := range cmds {
		out[i] = c
		switch c.Kind {
		case path.KindMoveTo, path.KindLineTo:
			out[i].Y = c.Y + dy
		case path.KindCubicTo:
			out[i].Y1 = c.Y1 + dy
			out[i].Y2 = c.Y2 + dy
			out[i].Y = c.Y + dy
		case path.KindQuadTo:
			out[i].Y1 = c.Y1 + dy
			out[i].Y = c.Y + dy
		}
	}
	return out
}

// axisCenterStackedVBox wraps a body in a RaiseBox so its visual centre
// aligns with the math axis.
func axisCenterStackedVBox(body *Box, opts Options) *Box {
	hVis := body.Height + body.Depth
	axis := opts.Metrics().AxisHeight * opts.SizeMultiplier()
	shift := axis - hVis/2
	depth := hVis/2 - axis
	if depth < 0 {
		depth = 0
	}
	return &Box{
		Width:  body.Width,
		Height: hVis/2 + axis,
		Depth:  depth,
		Color:  opts.Color,
		Content: RaiseBox{
			Body:  body,
			Shift: shift,
		},
	}
}

func makeTallSVGDelim(kind stackDelimKind, totalH float64, opts Options) *Box {
	label := svgLabelForStack(kind)
	if label == "" {
		return nil
	}
	var topC, botC rune
	switch kind {
	case skBracketOpen:
		topC, botC = 0x23a1, 0x23a3
	case skBracketClose:
		topC, botC = 0x23a4, 0x23a6
	case skParenOpen:
		topC, botC = 0x239b, 0x239d
	case skParenClose:
		topC, botC = 0x239e, 0x23a0
	case skLFloor:
		topC, botC = 0x23a2, 0x23a3
	case skRFloor:
		topC, botC = 0x23a5, 0x23a6
	case skLCeil:
		topC, botC = 0x23a1, 0x23a2
	case skRCeil:
		topC, botC = 0x23a4, 0x23a5
	default:
		return nil
	}
	topM, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, topC)
	botM, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, botC)
	minH := (topM.Height + topM.Depth) + (botM.Height + botM.Depth)
	midEm := totalH - minH
	if midEm < 0 {
		midEm = 0
	}
	midTh := int64(midEm*1000 + 0.5)
	d := tallDelimSVGPath(label, midTh)
	if d == "" {
		return nil
	}
	raw := parseSVGPath(d)
	if len(raw) == 0 {
		return nil
	}
	cmds := scaleSVGPathThousandths(raw)
	cmds = shiftPathY(cmds, -totalH)
	w := widthEmForStackLabel(label)
	inner := &Box{
		Width: w, Height: totalH, Depth: 0, Color: opts.Color,
		Content: SvgPath{Commands: cmds, Fill: true},
	}
	return axisCenterStackedVBox(inner, opts)
}

// makeBraceRepeatSvg builds a thin vertical rectangle joining two brace pieces.
func makeBraceRepeatSvg(innerH float64, opts Options) *Box {
	m, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, 0x23aa)
	x0 := 384.0 / 1000.0
	x1 := 504.0 / 1000.0
	cmds := []path.Command{
		path.MoveTo(x0, -innerH),
		path.LineTo(x1, -innerH),
		path.LineTo(x1, 0),
		path.LineTo(x0, 0),
		{Kind: path.KindClose},
	}
	return &Box{
		Width: m.Width, Height: innerH, Depth: 0, Color: opts.Color,
		Content: SvgPath{Commands: cmds, Fill: true},
	}
}

func size4Glyph(charCode rune, opts Options) *Box {
	m, ok := fontmetrics.Lookup(fontmetrics.FontSize4Regular, charCode)
	if !ok {
		return NewEmpty()
	}
	return &Box{
		Width: m.Width, Height: m.Height, Depth: m.Depth, Color: opts.Color,
		Content: Glyph{FontID: fontmetrics.FontSize4Regular, CharCode: charCode},
	}
}

func makeStackedVBox(children []VBoxChild, opts Options) *Box {
	// Total height = sum of all box heights+depths and kerns.
	totalH := 0.0
	for _, c := range children {
		switch k := c.Kind.(type) {
		case VBoxBox:
			totalH += k.Box.Height + k.Box.Depth
		case VBoxKern:
			totalH += k.Em
		}
	}
	w := 0.0
	for _, c := range children {
		if k, ok := c.Kind.(VBoxBox); ok {
			if k.Box.Width > w {
				w = k.Box.Width
			}
		}
	}
	return &Box{
		Width: w, Height: totalH, Depth: 0, Color: opts.Color,
		Content: VBox{Children: children},
	}
}

func makeGlyphStackDelim(kind stackDelimKind, totalH float64, opts Options) *Box {
	bx := func(b *Box) VBoxChild { return VBoxChild{Kind: VBoxBox{Box: b}} }
	kern := func(k float64) VBoxChild { return VBoxChild{Kind: VBoxKern{Em: -k}} }
	switch kind {
	case skBraceOpen, skBraceClose:
		var topC, midC, botC rune
		if kind == skBraceOpen {
			topC, midC, botC = 0x23a7, 0x23a8, 0x23a9
		} else {
			topC, midC, botC = 0x23ab, 0x23ac, 0x23ad
		}
		top, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, topC)
		mid, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, midC)
		bot, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, botC)
		minH := (top.Height + top.Depth) + (mid.Height + mid.Depth) + (bot.Height + bot.Depth)
		innerH := (totalH-minH)/2 + 2*stackedLap
		if innerH < 2*stackedLap {
			innerH = 2 * stackedLap
		}
		children := []VBoxChild{
			bx(size4Glyph(topC, opts)),
			kern(stackedLap),
			bx(makeBraceRepeatSvg(innerH, opts)),
			kern(stackedLap),
			bx(size4Glyph(midC, opts)),
			kern(stackedLap),
			bx(makeBraceRepeatSvg(innerH, opts)),
			kern(stackedLap),
			bx(size4Glyph(botC, opts)),
		}
		return axisCenterStackedVBox(makeStackedVBox(children, opts), opts)
	case skGroupOpen, skGroupClose:
		var topC, botC rune
		if kind == skGroupOpen {
			topC, botC = 0x23a7, 0x23a9
		} else {
			topC, botC = 0x23ab, 0x23ad
		}
		top, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, topC)
		bot, _ := fontmetrics.Lookup(fontmetrics.FontSize4Regular, botC)
		minH := (top.Height + top.Depth) + (bot.Height + bot.Depth)
		innerH := totalH - minH + 2*stackedLap
		if innerH < 2*stackedLap {
			innerH = 2 * stackedLap
		}
		children := []VBoxChild{
			bx(size4Glyph(topC, opts)),
			kern(stackedLap),
			bx(makeBraceRepeatSvg(innerH, opts)),
			kern(stackedLap),
			bx(size4Glyph(botC, opts)),
		}
		return axisCenterStackedVBox(makeStackedVBox(children, opts), opts)
	}
	return nil
}

// makeStackedDelimIfNeeded returns a stacked / SVG delimiter when the
// best plain glyph cannot reach total_height.
func makeStackedDelimIfNeeded(delim string, totalH, bestGlyphTotal float64, opts Options) *Box {
	if isStackNeverDelim(delim) {
		return nil
	}
	if bestGlyphTotal+1e-6 >= totalH {
		return nil
	}
	kind := classifyStackable(delim)
	if kind == skNone {
		return nil
	}
	switch kind {
	case skBraceOpen, skBraceClose, skGroupOpen, skGroupClose:
		return makeGlyphStackDelim(kind, totalH, opts)
	default:
		return makeTallSVGDelim(kind, totalH, opts)
	}
}

// =============================================================================
// Tall vertical-bar delimiters (\biggm\vert etc.)
// =============================================================================

func vertRepeatPieceHeight(isDouble bool) float64 {
	code := rune(8739)
	if isDouble {
		code = rune(8741)
	}
	if m, ok := fontmetrics.Lookup(fontmetrics.FontSize1Regular, code); ok {
		return m.Height + m.Depth
	}
	return 0.5
}

func katexVertRealHeight(requested float64, isDouble bool) float64 {
	piece := vertRepeatPieceHeight(isDouble)
	minH := 2 * piece
	repeat := (requested - minH) / piece
	if repeat < 0 {
		repeat = 0
	}
	// ceil
	rc := float64(int64(repeat))
	if repeat > rc+1e-9 {
		rc++
	}
	h := minH + rc*piece
	if !isDouble && requested > 2.99 && requested < 3.01 {
		h *= 1.135
	}
	return h
}

func tallVertSVGPathData(midTh int64, isDouble bool) string {
	neg := -midTh
	if !isDouble {
		return fmt.Sprintf(
			"M145 15 v585 v%d v585 c2.667,10,9.667,15,21,15 c10,0,16.667,-5,20,-15 v-585 v%d v-585 c-2.667,-10,-9.667,-15,-21,-15 c-10,0,-16.667,5,-20,15z M188 15 H145 v585 v%d v585 h43z",
			midTh, neg, midTh)
	}
	return fmt.Sprintf(
		"M145 15 v585 v%d v585 c2.667,10,9.667,15,21,15 c10,0,16.667,-5,20,-15 v-585 v%d v-585 c-2.667,-10,-9.667,-15,-21,-15 c-10,0,-16.667,5,-20,15z M188 15 H145 v585 v%d v585 h43z M367 15 v585 v%d v585 c2.667,10,9.667,15,21,15 c10,0,16.667,-5,20,-15 v-585 v%d v-585 c-2.667,-10,-9.667,-15,-21,-15 c-10,0,-16.667,5,-20,15z M410 15 H367 v585 v%d v585 h43z",
		midTh, neg, midTh, midTh, neg, midTh)
}

func mapVertPathYToBaseline(cmds []path.Command, height, depth float64, viewBoxHeight int64) []path.Command {
	spanEm := float64(viewBoxHeight) / 1000.0
	total := height + depth
	scaleY := 1.0
	if spanEm > 0 {
		scaleY = total / spanEm
	}
	out := make([]path.Command, len(cmds))
	mapY := func(y float64) float64 { return -height + y*scaleY }
	for i, c := range cmds {
		out[i] = c
		switch c.Kind {
		case path.KindMoveTo, path.KindLineTo:
			out[i].Y = mapY(c.Y)
		case path.KindCubicTo:
			out[i].Y1 = mapY(c.Y1)
			out[i].Y2 = mapY(c.Y2)
			out[i].Y = mapY(c.Y)
		case path.KindQuadTo:
			out[i].Y1 = mapY(c.Y1)
			out[i].Y = mapY(c.Y)
		}
	}
	return out
}

func makeVertDelimBox(totalH float64, isDouble bool, opts Options) *Box {
	realH := katexVertRealHeight(totalH, isDouble)
	axis := opts.Metrics().AxisHeight
	depth := realH/2 - axis
	if depth < 0 {
		depth = 0
	}
	height := realH - depth
	w := 0.333
	if isDouble {
		w = 0.556
	}
	piece := vertRepeatPieceHeight(isDouble)
	midEm := realH - 2*piece
	if midEm < 0 {
		midEm = 0
	}
	midTh := int64(midEm*1000 + 0.5)
	viewBoxH := int64(realH*1000 + 0.5)
	d := tallVertSVGPathData(midTh, isDouble)
	raw := parseSVGPath(d)
	scaled := scaleSVGPathThousandths(raw)
	cmds := mapVertPathYToBaseline(scaled, height, depth, viewBoxH)
	return &Box{
		Width: w, Height: height, Depth: depth, Color: opts.Color,
		Content: SvgPath{Commands: cmds, Fill: true},
	}
}

func isVertDelim(delim string) bool {
	switch delim {
	case "|", `\vert`, `\lvert`, `\rvert`:
		return true
	}
	return false
}

func isDoubleVertDelim(delim string) bool {
	switch delim {
	case `\|`, `\Vert`, `\lVert`, `\rVert`:
		return true
	}
	return false
}
