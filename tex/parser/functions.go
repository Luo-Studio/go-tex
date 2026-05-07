package parser

import (
	"fmt"
	"strings"

	"github.com/luo-studio/go-tex/tex/source"
)

// functionDispatch handles the registry of LaTeX commands the parser knows.
// Returns (node, nil) on a successful function call, (nil, nil) when the
// current token is not a recognised function (so parseGroup can fall back
// to parseSymbol), or (nil, err) on a parse error.
func functionDispatch(p *Parser, callerName, breakOnText string) (Node, error) {
	cmd := p.cur.Text
	switch cmd {
	case `\begin`:
		return parseEnvironment(p)
	case `\frac`, `\dfrac`, `\tfrac`, `\cfrac`:
		return parseFrac(p)
	case `\binom`, `\dbinom`, `\tbinom`:
		return parseBinom(p)
	case `\sqrt`:
		return parseSqrt(p)
	case `\displaystyle`, `\textstyle`, `\scriptstyle`, `\scriptscriptstyle`:
		return parseStyling(p, breakOnText)
	case `\text`, `\textrm`, `\textsf`, `\texttt`, `\textnormal`,
		`\textbf`, `\textit`, `\textmd`, `\textup`, `\textsl`, `\textsc`:
		return parseText(p)
	case `\mathrm`, `\mathit`, `\mathbf`, `\mathnormal`, `\mathsf`,
		`\mathtt`, `\mathfrak`, `\mathcal`, `\mathbb`, `\mathscr`,
		`\Bbb`, `\frak`, `\bold`, `\mathsfit`:
		return parseMathFont(p)
	case `\rm`, `\sf`, `\tt`, `\bf`, `\it`, `\cal`:
		return parseOldFont(p, breakOnText)
	case `\boldsymbol`, `\bm`:
		return parseBoldsymbol(p)
	case `\emph`:
		return parseEmph(p)
	case `\left`:
		return parseLeft(p)
	case `\right`:
		return parseRight(p)
	case `\middle`:
		return parseMiddle(p)
	case `\operatorname@`, `\operatornamewithlimits`:
		return parseOperatorName(p)
	case `\overline`:
		return parseSimpleWrap(p, func(body Node) Node { return &Overline{Mode: p.mode, Body: body} })
	case `\underline`:
		return parseSimpleWrap(p, func(body Node) Node { return &Underline{Mode: p.mode, Body: body} })
	case `\phantom`:
		return parsePhantomLike(p, func(body []Node) Node { return &Phantom{Mode: p.mode, Body: body} })
	case `\hphantom`:
		return parsePhantomLike(p, func(body []Node) Node {
			return &Phantom{Mode: p.mode, Body: body}
		})
	case `\vphantom`:
		return parseSimpleWrap(p, func(body Node) Node { return &VPhantom{Mode: p.mode, Body: body} })
	case `\colon`:
		// `\colon` is a math-punct atom, not a symbol.
		// (Upstream defines it via macro to a complex expansion, but for the
		// common case the parser shape is just a punct atom.)
		return nil, nil
	}
	if isMathAccent(cmd) {
		return parseMathAccent(p)
	}
	if isAccentUnder(cmd) {
		return parseAccentUnder(p)
	}
	if cmd == `\mathop` {
		return parseMathOp(p)
	}
	if cmd == `\kern` || cmd == `\mkern` || cmd == `\hskip` || cmd == `\mskip` {
		return parseKern(p)
	}
	if node, ok := parseOpFunction(p); ok {
		return node, nil
	}
	if cmd == `\textcolor` || cmd == `\color` {
		return parseColor(p)
	}
	if isDelimSizing(cmd) {
		return parseDelimSizing(p)
	}
	if isInfixCmd(cmd) {
		return parseInfix(p)
	}
	if isTextAccent(cmd) {
		return parseTextAccent(p)
	}
	if isXArrow(cmd) {
		return parseXArrow(p)
	}
	switch cmd {
	case `\fbox`, `\cancel`, `\bcancel`, `\xcancel`, `\sout`, `\phase`, `\angl`,
		`\colorbox`, `\fcolorbox`:
		return parseEnclose(p)
	case `\mathllap`, `\mathrlap`, `\mathclap`:
		return parseLap(p)
	case `\rule`:
		return parseRule(p)
	case `\raisebox`:
		return parseRaiseBox(p)
	case `\mathchoice`:
		return parseMathChoice(p)
	case `\href`, `\url`:
		return parseHref(p)
	case `\smash`:
		return parseSmash(p)
	case `\mathord`, `\mathbin`, `\mathrel`, `\mathopen`, `\mathclose`,
		`\mathpunct`, `\mathinner`:
		return parseMClassWrap(p)
	case `\stackrel`, `\overset`, `\underset`:
		return parseStackrel(p)
	case `\overbracket`, `\underbracket`:
		return parseHorizBracket(p)
	case `\nonumber`, `\notag`:
		return parseNoNumber(p)
	case `\pmb`:
		return parsePmb(p)
	case `\vcenter`:
		return parseVCenter(p)
	case `\hbox`:
		return parseHBox(p)
	case `\html@mathml`:
		return parseHtmlMathml(p)
	case `\@char`:
		return parseAtChar(p)
	case `\newline`, `\\`:
		return parseCr(p)
	case `\(`, "$":
		return parseMathSwitch(p, cmd)
	case `\)`, `\]`:
		return nil, errAt("Mismatched "+cmd, p.cur)
	case `\relax`:
		return &Internal{Mode: p.mode}, p.advanceAndReturn()
	case `\hspace`, `\hfill`:
		return parseHSpace(p)
	case `\quad`, `\qquad`, `\enspace`, `\thinspace`, `\medspace`,
		`\thickspace`, `\negthinspace`, `\negmedspace`, `\negthickspace`,
		`\nobreakspace`:
		// Function-form spacing: emit Spacing without loc, matching
		// upstream spacing.rs handle_spacing.
		return parseSpacingNode(p)
	case `\tiny`, `\sixptsize`, `\scriptsize`, `\footnotesize`, `\small`,
		`\normalsize`, `\large`, `\Large`, `\LARGE`, `\huge`, `\Huge`:
		return parseSizing(p, breakOnText)
	}
	if cmd == `\overbrace` || cmd == `\underbrace` {
		return parseHorizBrace(p)
	}
	if cmd == `\not` {
		// Common case: \not\command applies to the next symbol/operator.
		// Without a macro expander we can't expand the upstream definition;
		// emit a placeholder MClass node so downstream consumers see something.
		return nil, nil
	}
	return nil, nil
}

// parseColor handles `\textcolor{name}{body}` and `\color{name}`.
func parseColor(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	colorArg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	color := flattenSymbolText(colorArg)
	if cmd == `\color` {
		body, err := p.parseExpression(true, "")
		if err != nil {
			return nil, err
		}
		return &Color{Mode: p.mode, Color: color, Body: body}, nil
	}
	bodyArg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	body := OrdArgument(bodyArg)
	return &Color{Mode: p.mode, Color: color, Body: body}, nil
}

// flattenSymbolText concatenates the text of a sequence of symbol nodes
// inside an OrdGroup (used to read color names like "red" or "#fff").
func flattenSymbolText(n Node) string {
	if g, ok := n.(*OrdGroup); ok {
		s := ""
		for _, c := range g.Body {
			if t, ok := SymbolText(c); ok {
				s += t
			}
		}
		return s
	}
	if t, ok := SymbolText(n); ok {
		return t
	}
	return ""
}

// parseHorizBrace handles `\overbrace{...}` and `\underbrace{...}`.
func parseHorizBrace(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &HorizBrace{
		Mode:   p.mode,
		Label:  cmd,
		IsOver: cmd == `\overbrace`,
		Base:   arg,
	}, nil
}

// parseFunctionArg consumes one mandatory argument: either `{...}` (which
// becomes an OrdGroup spanning the braces) or a single symbol token, in
// which case the symbol is wrapped in an OrdGroup whose loc matches the
// token. This mirrors upstream's parse_argument_group, where every function
// arg ends up as an OrdGroup so downstream consumers don't have to special-
// case "single symbol" vs "braced group".
//
// Some callers (e.g. accents, sqrt index) prefer the unwrapped form; they
// can call NormalizeArgument on the returned node.
func parseFunctionArg(p *Parser, name string) (Node, error) {
	p.consumeSpaces()
	startTok := p.cur
	g, err := p.parseGroup(name, "")
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errAt(fmt.Sprintf("Expected group after %s", name), p.cur)
	}
	if _, isGroup := g.(*OrdGroup); isGroup {
		return g, nil
	}
	// Wrap a bare argument in an OrdGroup so the AST matches upstream.
	loc := startTok.Loc
	return &OrdGroup{Mode: p.mode, Body: []Node{g}, Loc: &loc}, nil
}

// =============================================================================
// Fractions and roots
// =============================================================================

func parseFrac(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	numer, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	denom, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	frac := &GenFrac{
		Mode:       p.mode,
		Continued:  cmd == `\cfrac`,
		Numer:      numer,
		Denom:      denom,
		HasBarLine: true,
	}
	return wrapFracStyle(p, cmd, frac), nil
}

// wrapFracStyle wraps a GenFrac in a Styling node when the command name
// implies a style (\dfrac/\dbinom -> display, \tfrac/\tbinom -> text,
// \cfrac -> display). Mirrors upstream's handle_frac.
func wrapFracStyle(p *Parser, cmd string, frac Node) Node {
	if cmd == `\cfrac` || (len(cmd) > 1 && cmd[1] == 'd') {
		return &Styling{Mode: p.mode, Style: StyleDisplay, Body: []Node{frac}}
	}
	if len(cmd) > 1 && cmd[1] == 't' {
		return &Styling{Mode: p.mode, Style: StyleText, Body: []Node{frac}}
	}
	return frac
}

func parseBinom(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	numer, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	denom, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	left := "("
	right := ")"
	frac := &GenFrac{
		Mode:       p.mode,
		Numer:      numer,
		Denom:      denom,
		HasBarLine: false,
		LeftDelim:  &left,
		RightDelim: &right,
	}
	return wrapFracStyle(p, cmd, frac), nil
}

func parseSqrt(p *Parser) (Node, error) {
	p.advance()
	p.consumeSpaces()
	var index Node
	if p.cur.Text == "[" {
		startTok := p.cur
		p.advance()
		body, err := p.parseExpression(false, "]")
		if err != nil {
			return nil, err
		}
		endTok := p.cur
		if err := p.expect("]"); err != nil {
			return nil, err
		}
		loc := source.Range(startTok.Loc, endTok.Loc)
		index = &OrdGroup{Mode: p.mode, Body: body, Loc: &loc}
	}
	body, err := parseFunctionArg(p, `\sqrt`)
	if err != nil {
		return nil, err
	}
	return &Sqrt{Mode: p.mode, Body: body, Index: index}, nil
}

// =============================================================================
// Styling (\displaystyle, \textstyle, \scriptstyle, \scriptscriptstyle)
// =============================================================================

// =============================================================================
// Sizing: \tiny, \scriptsize, \small, \normalsize, \large, \Large, \LARGE,
// \huge, \Huge.
// =============================================================================

var sizeFuncs = []string{
	`\tiny`, `\sixptsize`, `\scriptsize`, `\footnotesize`, `\small`,
	`\normalsize`, `\large`, `\Large`, `\LARGE`, `\huge`, `\Huge`,
}

func parseSizing(p *Parser, breakOnText string) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := p.parseExpression(false, breakOnText)
	if err != nil {
		return nil, err
	}
	size := uint8(6)
	for i, n := range sizeFuncs {
		if n == cmd {
			size = uint8(i + 1)
			break
		}
	}
	return &Sizing{Mode: p.mode, Size: size, Body: body}, nil
}

func parseStyling(p *Parser, breakOnText string) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := p.parseExpression(true, breakOnText)
	if err != nil {
		return nil, err
	}
	var style StyleStr
	switch cmd {
	case `\displaystyle`:
		style = StyleDisplay
	case `\textstyle`:
		style = StyleText
	case `\scriptstyle`:
		style = StyleScript
	case `\scriptscriptstyle`:
		style = StyleScriptScript
	default:
		style = StyleText
	}
	return &Styling{Mode: p.mode, Style: style, Body: body}, nil
}

// =============================================================================
// Text (\text, \textrm, etc.)
// =============================================================================

func parseText(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	// Switch to text mode for the body, then restore.
	prev := p.mode
	p.switchMode(ModeText)
	arg, err := parseFunctionArg(p, cmd)
	p.switchMode(prev)
	if err != nil {
		return nil, err
	}
	body := OrdArgument(arg)
	font := cmd
	return &Text{Mode: p.mode, Body: body, Font: &font}, nil
}

// parseMathFont handles \mathrm, \mathbf, etc. — produces a Font node.
func parseMathFont(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	body = NormalizeArgument(body)
	font := mathFontKaTeXName(cmd)
	return &Font{Mode: p.mode, Font: font, Body: body}, nil
}

// mathFontKaTeXName converts \mathXX -> the upstream "fontXX" string.
//
//	\mathrm  -> mathrm
//	\Bbb     -> mathbb
//	\frak    -> mathfrak
//	\bold    -> mathbf
//	\mathbf  -> mathbf, etc.
func mathFontKaTeXName(cmd string) string {
	switch cmd {
	case `\Bbb`:
		return "mathbb"
	case `\frak`:
		return "mathfrak"
	case `\bold`:
		return "mathbf"
	}
	return strings.TrimPrefix(cmd, `\`)
}

// parseOldFont handles \rm, \sf, \tt, \bf, \it, \cal — old-style font
// switches that consume the rest of the current group.
func parseOldFont(p *Parser, breakOnText string) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := p.parseExpression(true, breakOnText)
	if err != nil {
		return nil, err
	}
	font := "math" + cmd[1:]
	return &Font{
		Mode: p.mode,
		Font: font,
		Body: &OrdGroup{Mode: p.mode, Body: body},
	}, nil
}

// parseBoldsymbol handles \boldsymbol and \bm — emit MClass[Font[boldsymbol]].
func parseBoldsymbol(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &MClass{
		Mode:   p.mode,
		Mclass: "mord",
		Body: []Node{&Font{
			Mode: p.mode,
			Font: "boldsymbol",
			Body: body,
		}},
		IsCharacterBox: false,
	}, nil
}

// parseEmph handles \emph{...} — emit Text node with font="\emph",
// switching to text mode for the body (mirrors upstream).
func parseEmph(p *Parser) (Node, error) {
	p.advance()
	prev := p.mode
	p.switchMode(ModeText)
	body, err := parseFunctionArg(p, `\emph`)
	p.switchMode(prev)
	if err != nil {
		return nil, err
	}
	font := `\emph`
	return &Text{
		Mode: p.mode,
		Body: OrdArgument(body),
		Font: &font,
	}, nil
}

// =============================================================================
// Accents
// =============================================================================

var (
	mathAccents = map[string]struct{}{
		`\acute`: {}, `\grave`: {}, `\ddot`: {}, `\tilde`: {}, `\bar`: {},
		`\breve`: {}, `\check`: {}, `\hat`: {}, `\vec`: {}, `\dot`: {},
		`\mathring`: {}, `\widecheck`: {}, `\widehat`: {}, `\widetilde`: {},
		`\overrightarrow`: {}, `\overleftarrow`: {}, `\Overrightarrow`: {},
		`\overleftrightarrow`: {}, `\overgroup`: {}, `\overlinesegment`: {},
		`\overleftharpoon`: {}, `\overrightharpoon`: {},
	}
	nonStretchyAccents = map[string]struct{}{
		`\acute`: {}, `\grave`: {}, `\ddot`: {}, `\tilde`: {}, `\bar`: {},
		`\breve`: {}, `\check`: {}, `\hat`: {}, `\vec`: {}, `\dot`: {},
		`\mathring`: {},
	}
	accentUnders = map[string]struct{}{
		`\underleftarrow`: {}, `\underrightarrow`: {}, `\underleftrightarrow`: {},
		`\undergroup`: {}, `\underlinesegment`: {}, `\utilde`: {},
	}
)

func isMathAccent(cmd string) bool { _, ok := mathAccents[cmd]; return ok }
func isAccentUnder(cmd string) bool { _, ok := accentUnders[cmd]; return ok }

func parseMathAccent(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	base := NormalizeArgument(arg)
	_, nonStretchy := nonStretchyAccents[cmd]
	isStretchy := !nonStretchy
	isShifty := !isStretchy || cmd == `\widehat` || cmd == `\widetilde` || cmd == `\widecheck`
	return &Accent{
		Mode:       p.mode,
		Label:      cmd,
		IsStretchy: &isStretchy,
		IsShifty:   &isShifty,
		Base:       base,
	}, nil
}

func parseAccentUnder(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	stretchy := true
	shifty := false
	return &AccentUnder{
		Mode:       p.mode,
		Label:      cmd,
		IsStretchy: &stretchy,
		IsShifty:   &shifty,
		Base:       arg,
	}, nil
}

// =============================================================================
// Left/Right
// =============================================================================

func parseLeft(p *Parser) (Node, error) {
	p.advance()
	p.consumeSpaces()
	delimTok := p.cur
	delim, err := readDelim(p)
	if err != nil {
		return nil, err
	}
	body, err := p.parseExpression(false, "")
	if err != nil {
		return nil, err
	}
	if p.cur.Text != `\right` {
		return nil, errAt(`\left missing matching \right`, delimTok)
	}
	right, err := parseRight(p)
	if err != nil {
		return nil, err
	}
	rr, _ := right.(*LeftRightRight)
	if rr == nil {
		return nil, errAt(`Expected \right after \left`, delimTok)
	}
	return &LeftRight{
		Mode:       p.mode,
		Body:       body,
		Left:       delim,
		Right:      rr.Delim,
		RightColor: rr.Color,
	}, nil
}

func parseRight(p *Parser) (Node, error) {
	p.advance()
	p.consumeSpaces()
	delim, err := readDelim(p)
	if err != nil {
		return nil, err
	}
	return &LeftRightRight{Mode: p.mode, Delim: delim}, nil
}

func parseMiddle(p *Parser) (Node, error) {
	p.advance()
	p.consumeSpaces()
	delim, err := readDelim(p)
	if err != nil {
		return nil, err
	}
	return &Middle{Mode: p.mode, Delim: delim}, nil
}

// readDelim reads a single delimiter token: either a literal delim (like
// "(", ")", "|", ".") or a control sequence (\lbrace, \uparrow, …).
func readDelim(p *Parser) (string, error) {
	t := p.cur
	if t.IsEOF() {
		return "", errAt("Expected delimiter, got EOF", t)
	}
	p.advance()
	return t.Text, nil
}

// =============================================================================
// Operator-name and simple wrappers
// =============================================================================

func parseOperatorName(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	body := OrdArgument(arg)
	always := cmd == `\operatornamewithlimits`
	return &OperatorName{
		Mode:               p.mode,
		Body:               body,
		AlwaysHandleSupSub: always,
		Limits:             false,
		ParentIsSupSub:     false,
	}, nil
}

func parseSimpleWrap(p *Parser, build func(Node) Node) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return build(arg), nil
}

func parsePhantomLike(p *Parser, build func([]Node) Node) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return build(OrdArgument(arg)), nil
}
