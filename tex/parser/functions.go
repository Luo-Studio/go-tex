package parser

import "fmt"

// functionDispatch handles a small registry of LaTeX commands. Returns
// (node, nil) on a successful function call, (nil, nil) when the current
// token is not a recognised function (so parseGroup can fall back to
// parseSymbol), or (nil, err) on a parse error.
//
// This is the starting point for porting the upstream functions/ subdirectory.
// Add commands here one at a time, incrementally growing parser coverage.
func functionDispatch(p *Parser, callerName, breakOnText string) (Node, error) {
	switch p.cur.Text {
	case `\frac`, `\dfrac`, `\tfrac`, `\cfrac`:
		return parseFrac(p)
	case `\binom`, `\dbinom`, `\tbinom`:
		return parseBinom(p)
	case `\sqrt`:
		return parseSqrt(p)
	}
	return nil, nil
}

// parseFunctionArg consumes one mandatory argument: either `{...}` or a
// single token group. Skips leading whitespace.
func parseFunctionArg(p *Parser, name string) (Node, error) {
	p.consumeSpaces()
	g, err := p.parseGroup(name, "")
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errAt(fmt.Sprintf("Expected group after %s", name), p.cur)
	}
	return g, nil
}

// parseFrac handles \frac{a}{b}, \dfrac, \tfrac, \cfrac.
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
	return &GenFrac{
		Mode:       p.mode,
		Continued:  cmd == `\cfrac`,
		Numer:      numer,
		Denom:      denom,
		HasBarLine: true,
		LeftDelim:  nil,
		RightDelim: nil,
		BarSize:    nil,
	}, nil
}

// parseBinom handles \binom{n}{k} (and \dbinom/\tbinom).
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
	return &GenFrac{
		Mode:       p.mode,
		Continued:  false,
		Numer:      numer,
		Denom:      denom,
		HasBarLine: false,
		LeftDelim:  &left,
		RightDelim: &right,
		BarSize:    nil,
	}, nil
}

// parseSqrt handles \sqrt[index]{body}.
func parseSqrt(p *Parser) (Node, error) {
	p.advance() // consume \sqrt
	p.consumeSpaces()
	var index Node
	if p.cur.Text == "[" {
		p.advance()
		body, err := p.parseExpression(false, "]")
		if err != nil {
			return nil, err
		}
		if err := p.expect("]"); err != nil {
			return nil, err
		}
		index = &OrdGroup{Mode: p.mode, Body: body}
	}
	body, err := parseFunctionArg(p, `\sqrt`)
	if err != nil {
		return nil, err
	}
	return &Sqrt{Mode: p.mode, Body: body, Index: index}, nil
}
