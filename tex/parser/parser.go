package parser

import (
	"fmt"

	"github.com/luo-studio/go-tex/tex/lexer"
	"github.com/luo-studio/go-tex/tex/macroexp"
	"github.com/luo-studio/go-tex/tex/source"
	"github.com/luo-studio/go-tex/tex/symbols"
)

// Error is a parse error with optional source location.
type Error struct {
	Message string
	Loc     *source.Location
}

func (e *Error) Error() string {
	if e.Loc != nil {
		return fmt.Sprintf("ParseError at position %d: %s", e.Loc.Start, e.Message)
	}
	return "ParseError: " + e.Message
}

func errAt(msg string, t lexer.Token) *Error {
	loc := t.Loc
	return &Error{Message: msg, Loc: &loc}
}

func errMsg(msg string) *Error { return &Error{Message: msg} }

// endOfExpression are the token texts that terminate an expression list.
var endOfExpression = map[string]struct{}{
	"}":           {},
	`\endgroup`:   {},
	`\end`:        {},
	`\right`:      {},
	"&":           {},
}

// implicitCommands are control sequences that are valid even though no
// function or symbol is registered for them — they are handled inline
// (sup/sub, \limits, \nolimits).
var implicitCommands = map[string]struct{}{
	"^":          {},
	"_":          {},
	`\limits`:    {},
	`\nolimits`:  {},
}

// Parse parses a LaTeX math input string into the top-level body. It is the
// public entry point matching upstream `parse(&str)`.
func Parse(input string) ([]Node, error) {
	p := newParser(input)
	body, err := p.parseExpression(false, "")
	if err != nil {
		return nil, err
	}
	if !p.cur.IsEOF() {
		return nil, errAt(fmt.Sprintf("Expected EOF, got %q", p.cur.Text), p.cur)
	}
	return body, nil
}

// Parser holds the recursive-descent state. Construct with newParser.
type Parser struct {
	gullet *macroexp.Expander
	cur    lexer.Token
	mode   Mode
}

func newParser(input string) *Parser {
	p := &Parser{gullet: macroexp.New(input, macroexp.ModeMath), mode: ModeMath}
	p.advance()
	return p
}

func (p *Parser) advance() {
	t, err := p.gullet.Next()
	if err != nil {
		// Macro expansion errors surface here as a synthetic EOF; the
		// resulting parse will fail with a clear error in the caller.
		t = lexer.EOF(p.gullet.Pos())
	}
	p.cur = t
}

// switchMode keeps the gullet's view of the current mode in sync.
func (p *Parser) switchMode(m Mode) {
	p.mode = m
	if m == ModeText {
		p.gullet.SetMode(macroexp.ModeText)
	} else {
		p.gullet.SetMode(macroexp.ModeMath)
	}
}

func (p *Parser) consumeSpaces() {
	for p.cur.IsSpace() {
		p.advance()
	}
}

func (p *Parser) expect(text string) error {
	if p.cur.Text != text {
		return errAt(fmt.Sprintf("Expected '%s', got '%s'", text, p.cur.Text), p.cur)
	}
	p.advance()
	return nil
}

// parseExpression parses a list of atoms until an end-of-expression token.
func (p *Parser) parseExpression(breakOnInfix bool, breakOnText string) ([]Node, error) {
	body := []Node{}
	for {
		if p.mode == ModeMath {
			p.consumeSpaces()
		}
		if p.cur.IsEOF() {
			break
		}
		if _, isEnd := endOfExpression[p.cur.Text]; isEnd {
			break
		}
		if breakOnText != "" && p.cur.Text == breakOnText {
			break
		}
		atom, err := p.parseAtom(breakOnText)
		if err != nil {
			return nil, err
		}
		if atom == nil {
			break
		}
		if _, isInternal := atom.(*Internal); isInternal {
			continue
		}
		body = append(body, atom)
		if breakOnInfix {
			if _, isInfix := atom.(*Infix); isInfix {
				break
			}
		}
	}
	return handleInfixNodes(body, p.mode)
}

// parseAtom parses one atom, possibly followed by ^/_ for super/subscripts
// (math mode) and `'` (prime) shortcuts.
func (p *Parser) parseAtom(breakOnText string) (Node, error) {
	base, err := p.parseGroup("atom", breakOnText)
	if err != nil {
		return nil, err
	}
	if base != nil {
		if _, ok := base.(*Internal); ok {
			return base, nil
		}
	}
	if p.mode == ModeText {
		return base, nil
	}

	var sup, sub Node
	for {
		p.consumeSpaces()
		t := p.cur
		switch {
		case t.Text == `\limits` || t.Text == `\nolimits`:
			// \limits / \nolimits attach to operator nodes (Op / OperatorName).
			// Without an op base the upstream silently consumes the directive.
			p.advance()
			// TODO: when Op/OperatorName are produced, mutate base.limits.
		case t.Text == "^":
			if sup != nil {
				return nil, errAt("Double superscript", t)
			}
			s, err := p.parseSupSub("superscript")
			if err != nil {
				return nil, err
			}
			sup = s
		case t.Text == "_":
			if sub != nil {
				return nil, errAt("Double subscript", t)
			}
			s, err := p.parseSupSub("subscript")
			if err != nil {
				return nil, err
			}
			sub = s
		case t.Text == "'":
			if sup != nil {
				return nil, errAt("Double superscript", t)
			}
			prime := &TextOrd{Mode: p.mode, Text: `\prime`}
			primes := []Node{prime}
			p.advance()
			for p.cur.Text == "'" {
				primes = append(primes, &TextOrd{Mode: p.mode, Text: `\prime`})
				p.advance()
			}
			if p.cur.Text == "^" {
				s, err := p.parseSupSub("superscript")
				if err != nil {
					return nil, err
				}
				primes = append(primes, s)
			}
			sup = &OrdGroup{Mode: p.mode, Body: primes}
		default:
			goto done
		}
	}
done:
	if sup != nil || sub != nil {
		return &SupSub{Mode: p.mode, Base: base, Sup: sup, Sub: sub}, nil
	}
	return base, nil
}

func (p *Parser) parseSupSub(name string) (Node, error) {
	tok := p.cur
	p.advance()
	p.consumeSpaces()
	g, err := p.parseGroup(name, "")
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errAt(fmt.Sprintf("Expected group after '%s'", tok.Text), tok)
	}
	if _, ok := g.(*Internal); ok {
		g2, err := p.parseGroup(name, "")
		if err != nil {
			return nil, err
		}
		if g2 == nil {
			return nil, errAt(fmt.Sprintf("Expected group after '%s'", tok.Text), tok)
		}
		return g2, nil
	}
	return g, nil
}

// parseGroup parses one of:
//   - `{...}`
//   - a function call (TODO: requires the function registry)
//   - a single symbol
func (p *Parser) parseGroup(name, breakOnText string) (Node, error) {
	first := p.cur
	if first.Text == "{" || first.Text == `\begingroup` {
		p.advance()
		end := "}"
		semisimple := (*bool)(nil)
		if first.Text == `\begingroup` {
			end = `\endgroup`
			tt := true
			semisimple = &tt
		}
		body, err := p.parseExpression(false, end)
		if err != nil {
			return nil, err
		}
		last := p.cur
		if err := p.expect(end); err != nil {
			return nil, err
		}
		loc := source.Range(first.Loc, last.Loc)
		return &OrdGroup{Mode: p.mode, Body: body, Semisimple: semisimple, Loc: &loc}, nil
	}
	// Function dispatch is not yet implemented; fall through to parseSymbol.
	if node, err := p.parseFunction(name, breakOnText); err != nil {
		return nil, err
	} else if node != nil {
		return node, nil
	}
	node, err := p.parseSymbol()
	if err != nil {
		return nil, err
	}
	if node == nil && len(first.Text) > 0 && first.Text[0] == '\\' {
		if _, ok := implicitCommands[first.Text]; !ok {
			return nil, errAt(fmt.Sprintf("Undefined control sequence: %s", first.Text), first)
		}
	}
	return node, nil
}

// parseSymbol parses one symbol token into a leaf node, or returns (nil, nil)
// if the current token doesn't classify as a symbol.
func (p *Parser) parseSymbol() (Node, error) {
	nucleus := p.cur
	text := nucleus.Text

	// `\verb...` is a single token from the lexer; decompose into a Verb node.
	if len(text) >= 5 && text[:5] == `\verb` {
		p.advance()
		body := text[5:]
		star := false
		if len(body) > 0 && body[0] == '*' {
			star = true
			body = body[1:]
		}
		if len(body) < 2 {
			return nil, errAt(`\verb assertion failed`, nucleus)
		}
		// Strip leading and trailing delimiter characters.
		body = body[1 : len(body)-1]
		loc := nucleus.Loc
		return &Verb{Mode: ModeText, Body: body, Star: star, Loc: &loc}, nil
	}

	// `^` and `_` are not symbols — handled by parseAtom.
	if text == "^" || text == "_" {
		return nil, nil
	}
	// Bare backslash from an unterminated control sequence.
	if text == `\` {
		return nil, nil
	}

	mode := symbols.ModeMath
	if p.mode == ModeText {
		mode = symbols.ModeText
	}
	if info, ok := symbols.Lookup(text, mode); ok {
		loc := source.Range(nucleus.Loc, nucleus.Loc)
		var node Node
		if info.Group.IsAtom() {
			node = &Atom{Mode: p.mode, Family: groupToFamily(info.Group), Text: text, Loc: &loc}
		} else {
			switch info.Group {
			case symbols.GroupMathOrd:
				node = &MathOrd{Mode: p.mode, Text: text, Loc: &loc}
			case symbols.GroupTextOrd:
				node = &TextOrd{Mode: p.mode, Text: text, Loc: &loc}
			case symbols.GroupOpToken:
				node = &OpToken{Mode: p.mode, Text: text, Loc: &loc}
			case symbols.GroupAccentToken:
				node = &AccentToken{Mode: p.mode, Text: text, Loc: &loc}
			case symbols.GroupSpacing:
				node = &Spacing{Mode: p.mode, Text: text, Loc: &loc}
			default:
				node = &MathOrd{Mode: p.mode, Text: text, Loc: &loc}
			}
		}
		p.advance()
		return node, nil
	}

	// Non-ASCII characters with no symbol entry → treat as TextOrd in text
	// mode (matches KaTeX's fallthrough for unknown CJK / Latin extras).
	if len(text) > 0 {
		// Decode first rune.
		var first rune
		for _, r := range text {
			first = r
			break
		}
		if first >= 0x80 {
			loc := source.Range(nucleus.Loc, nucleus.Loc)
			node := &TextOrd{Mode: ModeText, Text: text, Loc: &loc}
			p.advance()
			return node, nil
		}
	}
	return nil, nil
}

func groupToFamily(g symbols.Group) AtomFamily {
	switch g {
	case symbols.GroupBin:
		return FamilyBin
	case symbols.GroupClose:
		return FamilyClose
	case symbols.GroupInner:
		return FamilyInner
	case symbols.GroupOpen:
		return FamilyOpen
	case symbols.GroupPunct:
		return FamilyPunct
	case symbols.GroupRel:
		return FamilyRel
	}
	return FamilyBin
}

// parseFunction is a stub for now. Returns (nil, nil) so parseGroup falls
// through to parseSymbol. Function support is added one command at a time.
func (p *Parser) parseFunction(name, breakOnText string) (Node, error) {
	return functionDispatch(p, name, breakOnText)
}
