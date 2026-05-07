package parser

import (
	"fmt"
	"strings"
)

// =============================================================================
// \kern, \mkern, \hskip, \mskip — fixed dimensional kerns.
// =============================================================================

func parseKern(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	dim, err := parseSizeGroup(p, cmd)
	if err != nil {
		return nil, err
	}
	if dim == nil {
		// Failed to parse — fall back to 0em like upstream.
		dim = &Measurement{Number: 0, Unit: "em"}
	}
	return &Kern{Mode: p.mode, Dimension: *dim}, nil
}

// parseSizeGroup reads a size argument in either of two forms:
//
//	{3pt}, {1.5em}, ...   braced
//	3pt, 1.5em, ...       unbraced (no leading {)
//
// Returns nil for an empty group, or an error for malformed input.
func parseSizeGroup(p *Parser, cmd string) (*Measurement, error) {
	p.consumeSpaces()
	if p.cur.Text == "{" {
		p.advance()
		var raw strings.Builder
		for !p.cur.IsEOF() && p.cur.Text != "}" {
			raw.WriteString(p.cur.Text)
			p.advance()
		}
		if p.cur.Text != "}" {
			return nil, errAt(fmt.Sprintf("Unterminated size argument for %s", cmd), p.cur)
		}
		p.advance()
		s := strings.TrimSpace(raw.String())
		if s == "" {
			return &Measurement{Number: 0, Unit: "pt"}, nil
		}
		num, unit, ok := splitSize(s)
		if !ok {
			return nil, errAt(fmt.Sprintf("Invalid size '%s' after %s", s, cmd), p.cur)
		}
		return &Measurement{Number: num, Unit: unit}, nil
	}
	// Unbraced: read digits and trailing unit letters from a single (or
	// adjacent) tokens until a non-digit/non-letter is seen.
	var raw strings.Builder
	for !p.cur.IsEOF() {
		t := p.cur.Text
		if isAllDigitsOrSign(t) || isLetterRun(t) || t == "." {
			raw.WriteString(t)
			p.advance()
			continue
		}
		break
	}
	s := strings.TrimSpace(raw.String())
	if s == "" {
		return nil, errAt(fmt.Sprintf("Expected size after %s", cmd), p.cur)
	}
	num, unit, ok := splitSize(s)
	if !ok {
		return nil, errAt(fmt.Sprintf("Invalid size '%s' after %s", s, cmd), p.cur)
	}
	return &Measurement{Number: num, Unit: unit}, nil
}

func isAllDigitsOrSign(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		switch {
		case c >= '0' && c <= '9':
			continue
		case (c == '+' || c == '-') && i == 0:
			continue
		default:
			return false
		}
	}
	return true
}

func isLetterRun(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}
