package parser

import (
	"fmt"
	"strings"
)

// =============================================================================
// Enclose family: \fbox, \cancel, \bcancel, \xcancel, \sout, \phase, \angl,
// \colorbox, \fcolorbox.
// =============================================================================

func parseEnclose(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	switch cmd {
	case `\fbox`, `\angl`:
		// HBox-arg: text mode + Styling[text] wrap.
		body, err := parseTextModeArg(p, cmd)
		if err != nil {
			return nil, err
		}
		styled := &Styling{Mode: ModeText, Style: StyleText, Body: []Node{body}}
		return &Enclose{Mode: p.mode, Label: cmd, Body: styled}, nil
	case `\cancel`, `\bcancel`, `\xcancel`, `\sout`, `\phase`:
		body, err := parseFunctionArg(p, cmd)
		if err != nil {
			return nil, err
		}
		return &Enclose{Mode: p.mode, Label: cmd, Body: body}, nil
	case `\colorbox`:
		colorArg, err := parseFunctionArg(p, cmd)
		if err != nil {
			return nil, err
		}
		body, err := parseTextModeArg(p, cmd)
		if err != nil {
			return nil, err
		}
		c := flattenSymbolText(colorArg)
		return &Enclose{Mode: p.mode, Label: cmd, BackgroundColor: &c, Body: body}, nil
	case `\fcolorbox`:
		borderArg, err := parseFunctionArg(p, cmd)
		if err != nil {
			return nil, err
		}
		bgArg, err := parseFunctionArg(p, cmd)
		if err != nil {
			return nil, err
		}
		body, err := parseTextModeArg(p, cmd)
		if err != nil {
			return nil, err
		}
		bc := flattenSymbolText(borderArg)
		bg := flattenSymbolText(bgArg)
		return &Enclose{Mode: p.mode, Label: cmd, BorderColor: &bc, BackgroundColor: &bg, Body: body}, nil
	}
	return nil, errAt("unknown enclose function "+cmd, p.cur)
}

// parseTextModeArg parses a function arg in text mode regardless of the
// outer mode. Used by commands like \colorbox/\fcolorbox where the body
// is rendered as text per upstream's ArgType::Text.
func parseTextModeArg(p *Parser, name string) (Node, error) {
	prev := p.mode
	p.switchMode(ModeText)
	arg, err := parseFunctionArg(p, name)
	p.switchMode(prev)
	if err != nil {
		return nil, err
	}
	return arg, nil
}

// =============================================================================
// \mathllap, \mathrlap, \mathclap → Lap
// =============================================================================

func parseLap(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	alignment := strings.TrimPrefix(cmd, `\math`) // "llap" / "rlap" / "clap"
	return &Lap{Mode: p.mode, Alignment: alignment, Body: body}, nil
}

// =============================================================================
// \rule[shift]{width}{height}
// =============================================================================

func parseRule(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	p.consumeSpaces()
	var shift *Measurement
	if p.cur.Text == "[" {
		p.advance()
		var raw strings.Builder
		for !p.cur.IsEOF() && p.cur.Text != "]" {
			raw.WriteString(p.cur.Text)
			p.advance()
		}
		if p.cur.Text != "]" {
			return nil, errAt("Unterminated [shift] in \\rule", p.cur)
		}
		p.advance()
		s := strings.TrimSpace(raw.String())
		if s != "" {
			n, u, ok := splitSize(s)
			if ok {
				shift = &Measurement{Number: n, Unit: u}
			}
		}
	}
	width, err := parseSizeGroup(p, cmd)
	if err != nil {
		return nil, err
	}
	height, err := parseSizeGroup(p, cmd)
	if err != nil {
		return nil, err
	}
	return &Rule{Mode: p.mode, Shift: shift, Width: *width, Height: *height}, nil
}

// =============================================================================
// \raisebox{dy}{body}
// =============================================================================

func parseRaiseBox(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	dy, err := parseSizeGroup(p, cmd)
	if err != nil {
		return nil, err
	}
	// Body is parsed in text mode and wrapped in Styling text — matches
	// upstream's ArgType::HBox + Styling{text} wrap.
	body, err := parseTextModeArg(p, cmd)
	if err != nil {
		return nil, err
	}
	styled := &Styling{Mode: ModeText, Style: StyleText, Body: []Node{body}}
	return &RaiseBox{Mode: p.mode, Dy: *dy, Body: styled}, nil
}

// =============================================================================
// \mathchoice{D}{T}{S}{SS}
// =============================================================================

func parseMathChoice(p *Parser) (Node, error) {
	p.advance()
	d, err := parseFunctionArg(p, `\mathchoice`)
	if err != nil {
		return nil, err
	}
	tx, err := parseFunctionArg(p, `\mathchoice`)
	if err != nil {
		return nil, err
	}
	s, err := parseFunctionArg(p, `\mathchoice`)
	if err != nil {
		return nil, err
	}
	ss, err := parseFunctionArg(p, `\mathchoice`)
	if err != nil {
		return nil, err
	}
	return &MathChoice{
		Mode:         p.mode,
		Display:      OrdArgument(d),
		Text:         OrdArgument(tx),
		Script:       OrdArgument(s),
		ScriptScript: OrdArgument(ss),
	}, nil
}

// =============================================================================
// \href{url}{body}, \url{addr}
// =============================================================================

func parseHref(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	urlText, err := parseURLArg(p, cmd)
	if err != nil {
		return nil, err
	}
	if cmd == `\url` {
		// \url{addr} renders the URL itself as typewriter text wrapped in Href.
		chars := make([]Node, 0, len(urlText))
		for _, r := range urlText {
			chars = append(chars, &TextOrd{Mode: ModeText, Text: string(r)})
		}
		font := `\texttt`
		body := []Node{&Text{Mode: p.mode, Body: chars, Font: &font}}
		return &Href{Mode: p.mode, Href: urlText, Body: body}, nil
	}
	body, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &Href{Mode: p.mode, Href: urlText, Body: OrdArgument(body)}, nil
}

// parseURLArg reads `{url}` collecting the raw text inside without macro
// expansion (URLs may contain `\`, `#`, `%`, etc.). For now we approximate
// by reading raw token text inside braces.
func parseURLArg(p *Parser, cmd string) (string, error) {
	p.consumeSpaces()
	if p.cur.Text != "{" {
		return "", errAt(fmt.Sprintf("Expected URL group after %s", cmd), p.cur)
	}
	p.advance()
	var b strings.Builder
	depth := 1
	for !p.cur.IsEOF() && depth > 0 {
		if p.cur.Text == "{" {
			depth++
		} else if p.cur.Text == "}" {
			depth--
			if depth == 0 {
				break
			}
		}
		b.WriteString(p.cur.Text)
		p.advance()
	}
	if p.cur.Text != "}" {
		return "", errAt("Unterminated URL group", p.cur)
	}
	p.advance()
	return b.String(), nil
}

// =============================================================================
// \smash[t|b]{body}
// =============================================================================

func parseSmash(p *Parser) (Node, error) {
	p.advance()
	p.consumeSpaces()
	smashHeight, smashDepth := true, true
	if p.cur.Text == "[" {
		p.advance()
		var raw strings.Builder
		for !p.cur.IsEOF() && p.cur.Text != "]" {
			raw.WriteString(p.cur.Text)
			p.advance()
		}
		if p.cur.Text == "]" {
			p.advance()
		}
		smashHeight, smashDepth = false, false
		ok := true
		for _, r := range raw.String() {
			switch r {
			case 't':
				smashHeight = true
			case 'b':
				smashDepth = true
			case ' ':
			default:
				ok = false
			}
		}
		if !ok {
			smashHeight, smashDepth = false, false
		}
	}
	body, err := parseFunctionArg(p, `\smash`)
	if err != nil {
		return nil, err
	}
	return &Smash{Mode: p.mode, Body: body, SmashHeight: smashHeight, SmashDepth: smashDepth}, nil
}

// =============================================================================
// \mathord/\mathbin/\mathrel/\mathopen/\mathclose/\mathpunct/\mathinner
// =============================================================================

func parseMClassWrap(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	body, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	mclass := "m" + strings.TrimPrefix(cmd, `\math`)
	bodyNodes := OrdArgument(body)
	isCharBox := len(bodyNodes) == 1 && IsSymbol(bodyNodes[0])
	return &MClass{
		Mode:           p.mode,
		Mclass:         mclass,
		Body:           bodyNodes,
		IsCharacterBox: isCharBox,
	}, nil
}

// =============================================================================
// \stackrel, \overset, \underset
// =============================================================================

func parseStackrel(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	shifted, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	base, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	mclass := "mrel"
	if cmd != `\stackrel` {
		mclass = binrelClass(base)
	}
	suppressBase := cmd != `\stackrel`
	always := true
	op := &Op{
		Mode:               p.mode,
		Limits:             true,
		AlwaysHandleSupSub: &always,
		SuppressBaseShift:  &suppressBase,
		ParentIsSupSub:     false,
		Symbol:             false,
		Body:               OrdArgument(base),
	}
	var sup, sub Node
	if cmd == `\underset` {
		sub = shifted
	} else {
		sup = shifted
	}
	supsub := &SupSub{Mode: p.mode, Base: op, Sup: sup, Sub: sub}
	return &MClass{
		Mode:           p.mode,
		Mclass:         mclass,
		Body:           []Node{supsub},
		IsCharacterBox: false,
	}, nil
}

func binrelClass(arg Node) string {
	atom := arg
	if og, ok := atom.(*OrdGroup); ok && len(og.Body) > 0 {
		atom = og.Body[0]
	}
	if a, ok := atom.(*Atom); ok {
		switch a.Family {
		case FamilyBin:
			return "mbin"
		case FamilyRel:
			return "mrel"
		}
	}
	return "mord"
}

// =============================================================================
// xArrow family: \xrightarrow[below]{body}, \xleftarrow, ...
// =============================================================================

var xArrowCommands = map[string]struct{}{
	`\xleftarrow`: {}, `\xrightarrow`: {}, `\xLeftarrow`: {}, `\xRightarrow`: {},
	`\xleftrightarrow`: {}, `\xLeftrightarrow`: {}, `\xhookleftarrow`: {},
	`\xhookrightarrow`: {}, `\xmapsto`: {}, `\xrightharpoondown`: {},
	`\xrightharpoonup`: {}, `\xleftharpoondown`: {}, `\xleftharpoonup`: {},
	`\xrightleftharpoons`: {}, `\xleftrightharpoons`: {}, `\xlongequal`: {},
	`\xtwoheadrightarrow`: {}, `\xtwoheadleftarrow`: {}, `\xtofrom`: {},
	`\xrightleftarrows`: {}, `\xrightequilibrium`: {}, `\xleftequilibrium`: {},
}

func isXArrow(cmd string) bool { _, ok := xArrowCommands[cmd]; return ok }

func parseXArrow(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	p.consumeSpaces()
	var below Node
	if p.cur.Text == "[" {
		p.advance()
		body, err := p.parseExpression(false, "]")
		if err != nil {
			return nil, err
		}
		if err := p.expect("]"); err != nil {
			return nil, err
		}
		below = &OrdGroup{Mode: p.mode, Body: body}
	}
	body, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &XArrow{Mode: p.mode, Label: cmd, Body: body, Below: below}, nil
}

// =============================================================================
// \overbracket, \underbracket — same shape as \overbrace/\underbrace.
// =============================================================================

func parseHorizBracket(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &HorizBrace{
		Mode:   p.mode,
		Label:  cmd,
		IsOver: strings.HasPrefix(cmd, `\over`),
		Base:   arg,
	}, nil
}

// =============================================================================
// \nonumber / \notag
// =============================================================================

func parseNoNumber(p *Parser) (Node, error) {
	p.advance()
	return &NoNumber{Mode: p.mode}, nil
}

// =============================================================================
// \pmb{...} (poor man's bold)
// =============================================================================

func parsePmb(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &Pmb{Mode: p.mode, Mclass: "mord", Body: OrdArgument(arg)}, nil
}

// =============================================================================
// \vcenter{...}
// =============================================================================

func parseVCenter(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	return &VCenter{Mode: p.mode, Body: arg}, nil
}

// =============================================================================
// \hbox{...}
// =============================================================================

func parseHBox(p *Parser) (Node, error) {
	p.advance()
	prev := p.mode
	p.switchMode(ModeText)
	arg, err := parseFunctionArg(p, `\hbox`)
	p.switchMode(prev)
	if err != nil {
		return nil, err
	}
	return &HBox{Mode: p.mode, Body: OrdArgument(arg)}, nil
}

// =============================================================================
// \html@mathml{html}{mathml}
// =============================================================================

func parseHtmlMathml(p *Parser) (Node, error) {
	p.advance()
	htmlArg, err := parseFunctionArg(p, `\html@mathml`)
	if err != nil {
		return nil, err
	}
	mathmlArg, err := parseFunctionArg(p, `\html@mathml`)
	if err != nil {
		return nil, err
	}
	return &HtmlMathMl{
		Mode:   p.mode,
		Html:   OrdArgument(htmlArg),
		MathMl: OrdArgument(mathmlArg),
	}, nil
}

// =============================================================================
// Text-mode accents: \', \", \^, \~, \=, \., \u, \v, \H, \r, \c, \textcircled
// =============================================================================

var textAccents = map[string]struct{}{
	`\'`: {}, "\\`": {}, `\^`: {}, `\~`: {}, `\=`: {}, `\u`: {},
	`\.`: {}, `\"`: {}, `\c`: {}, `\r`: {}, `\H`: {}, `\v`: {},
	`\textcircled`: {},
}

func isTextAccent(cmd string) bool { _, ok := textAccents[cmd]; return ok }

func parseTextAccent(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	arg, err := parseFunctionArg(p, cmd)
	if err != nil {
		return nil, err
	}
	stretchy := false
	shifty := true
	return &Accent{
		Mode:       ModeText,
		Label:      cmd,
		IsStretchy: &stretchy,
		IsShifty:   &shifty,
		Base:       arg,
	}, nil
}
