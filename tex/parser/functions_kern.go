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
		dim = &Measurement{Number: 0, Unit: "em"}
	}
	return &Kern{Mode: p.mode, Dimension: *dim}, nil
}

// parseHSpace handles \hspace{size} and \hfill (which produces a special
// fill kern). Both emit Kern nodes.
func parseHSpace(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	if cmd == `\hfill` {
		// \hfill is treated as a SpacingNode in upstream's spacing.rs.
		return &Spacing{Mode: p.mode, Text: cmd}, nil
	}
	dim, err := parseSizeGroup(p, cmd)
	if err != nil {
		return nil, err
	}
	if dim == nil {
		dim = &Measurement{Number: 0, Unit: "em"}
	}
	return &Kern{Mode: p.mode, Dimension: *dim}, nil
}

// parseSpacingNode handles plain spacing commands that emit a SpacingNode
// (\quad, \qquad, etc.) plus a few control-sequence names that the upstream
// passes through as SpacingNode (\space, \nobreakspace).
//
// Note: the function form does not carry a loc — upstream sets loc:None on
// these. (The symbol-table form, used for catcode-active characters like
// `\,` resolved via parseSymbol, does carry loc.)
func parseSpacingNode(p *Parser) (Node, error) {
	cmd := p.cur.Text
	p.advance()
	return &Spacing{Mode: p.mode, Text: cmd}, nil
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
	// Unbraced: read digits/sign/dot first, then AT MOST 2 trailing
	// letters as the unit. Mirrors upstream's
	// `^[-+]? *(?:$|\d+|\d+\.\d*|\.\d*) *[a-z]{0,2} *$` regex which
	// caps the unit at 2 chars so e.g. `\kern-.125emX` consumes `em`
	// and leaves `X` for the next token.
	var raw strings.Builder
	for !p.cur.IsEOF() {
		t := p.cur.Text
		if isAllDigitsOrSign(t) || t == "." {
			raw.WriteString(t)
			p.advance()
			continue
		}
		break
	}
	// Now consume at most 2 letters for the unit.
	letters := 0
	for !p.cur.IsEOF() && letters < 2 {
		t := p.cur.Text
		if !isLetterRun(t) {
			break
		}
		// If a single token has multiple letters, take only enough to
		// reach 2 total.
		take := 2 - letters
		if len(t) <= take {
			raw.WriteString(t)
			letters += len(t)
			p.advance()
		} else {
			raw.WriteString(t[:take])
			// Replace the current token with the remainder so it stays
			// in the stream (gullet will re-emit on next advance).
			p.cur.Text = t[take:]
			letters = 2
		}
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
