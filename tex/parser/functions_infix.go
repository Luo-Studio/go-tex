package parser

import "fmt"

// =============================================================================
// Infix operators: \over, \atop, \brace, \brack, \choose, \above.
//
// These are emitted as Infix nodes during parseExpression and lowered into
// GenFrac (or whichever target frac variant the upstream maps them to) by
// handleInfixNodes once the expression is complete.
// =============================================================================

// infixReplaceWith maps the infix command to the upstream "replaceWith"
// command name.
var infixReplaceWith = map[string]string{
	`\over`:   `\frac`,
	`\choose`: `\binom`,
	`\atop`:   `\\atopfrac`,
	`\brace`:  `\\bracefrac`,
	`\brack`:  `\\brackfrac`,
}

func isInfixCmd(cmd string) bool {
	_, ok := infixReplaceWith[cmd]
	return ok || cmd == `\above`
}

func parseInfix(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	if cmd == `\above` {
		// `\above {dimen}` consumes a size argument and emits an Infix
		// targeting `\\abovefrac`.
		size, err := parseSizeArg(p, cmd)
		if err != nil {
			return nil, err
		}
		return &Infix{
			Mode:        p.mode,
			ReplaceWith: `\\abovefrac`,
			Size:        size,
		}, nil
	}
	return &Infix{
		Mode:        p.mode,
		ReplaceWith: infixReplaceWith[cmd],
	}, nil
}

// parseSizeArg reads `{<size>}` and returns the parsed measurement.
// Currently a minimal subset — only "<num><unit>" forms inside braces.
func parseSizeArg(p *Parser, cmd string) (*Measurement, error) {
	p.consumeSpaces()
	if p.cur.Text != "{" {
		return nil, errAt(fmt.Sprintf("Expected size group after %s", cmd), p.cur)
	}
	p.advance()
	var raw string
	for !p.cur.IsEOF() && p.cur.Text != "}" {
		raw += p.cur.Text
		p.advance()
	}
	if p.cur.Text != "}" {
		return nil, errAt(fmt.Sprintf("Unterminated size group for %s", cmd), p.cur)
	}
	p.advance()
	num, unit, ok := splitSize(raw)
	if !ok {
		return nil, errAt(fmt.Sprintf("Invalid size '%s' after %s", raw, cmd), p.cur)
	}
	return &Measurement{Number: num, Unit: unit}, nil
}

// splitSize extracts the numeric prefix and unit suffix from a string like
// "2pt", "1.5em", "-3mu".
func splitSize(s string) (float64, string, bool) {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	for i < len(s) && (s[i] == ' ') {
		i++
	}
	startNum := i
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
			i++
		}
	}
	if i == startNum {
		return 0, "", false
	}
	numStr := s[:i]
	unit := s[i:]
	// Strip whitespace around unit.
	for len(unit) > 0 && unit[0] == ' ' {
		unit = unit[1:]
	}
	for len(unit) > 0 && unit[len(unit)-1] == ' ' {
		unit = unit[:len(unit)-1]
	}
	var num float64
	if _, err := fmt.Sscanf(numStr, "%f", &num); err != nil {
		return 0, "", false
	}
	return num, unit, true
}

// handleInfixNodes lowers any Infix node in body into a GenFrac that
// replaces the entire expression. Returns the (possibly rewritten) body.
//
// Mirrors upstream's parser::handle_infix_nodes.
func handleInfixNodes(body []Node, mode Mode) ([]Node, error) {
	idx := -1
	for i, n := range body {
		if _, ok := n.(*Infix); ok {
			if idx >= 0 {
				return nil, errMsg("only one infix operator per group")
			}
			idx = i
		}
	}
	if idx < 0 {
		return body, nil
	}

	infix := body[idx].(*Infix)
	numerBody := body[:idx]
	denomBody := body[idx+1:]

	numer := wrapOrSingle(numerBody, mode)
	denom := wrapOrSingle(denomBody, mode)

	switch infix.ReplaceWith {
	case `\frac`:
		return []Node{&GenFrac{Mode: mode, Numer: numer, Denom: denom, HasBarLine: true}}, nil
	case `\binom`:
		l, r := "(", ")"
		return []Node{&GenFrac{Mode: mode, Numer: numer, Denom: denom, LeftDelim: &l, RightDelim: &r}}, nil
	case `\\atopfrac`:
		return []Node{&GenFrac{Mode: mode, Numer: numer, Denom: denom}}, nil
	case `\\bracefrac`:
		l, r := `\{`, `\}`
		return []Node{&GenFrac{Mode: mode, Numer: numer, Denom: denom, LeftDelim: &l, RightDelim: &r}}, nil
	case `\\brackfrac`:
		l, r := "[", "]"
		return []Node{&GenFrac{Mode: mode, Numer: numer, Denom: denom, LeftDelim: &l, RightDelim: &r}}, nil
	case `\\abovefrac`:
		hasBar := infix.Size != nil && infix.Size.Number > 0
		return []Node{&GenFrac{
			Mode: mode, Numer: numer, Denom: denom,
			HasBarLine: hasBar, BarSize: infix.Size,
		}}, nil
	}
	return []Node{&GenFrac{Mode: mode, Numer: numer, Denom: denom, HasBarLine: true}}, nil
}

// wrapOrSingle returns the single child if body has exactly one OrdGroup,
// otherwise wraps body in a fresh OrdGroup. Mirrors upstream's branch.
func wrapOrSingle(body []Node, mode Mode) Node {
	if len(body) == 1 {
		if g, ok := body[0].(*OrdGroup); ok {
			return g
		}
	}
	return &OrdGroup{Mode: mode, Body: body}
}
