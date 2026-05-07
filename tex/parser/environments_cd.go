package parser

import (
	"github.com/luo-studio/go-tex/tex/lexer"
)

// handleCDEnv parses `\begin{CD} ... \end{CD}`. The CD environment uses an
// extra arrow syntax: `@>label>>` etc. We collect raw tokens until the
// terminating `\end`, split on row-breaks (`\\` / `\cr`), and walk each row
// dispatching on the leading `@` to build CdArrow cells.
func handleCDEnv(p *Parser) (Node, error) {
	raw := []lexer.Token{}
	for !p.cur.IsEOF() && p.cur.Text != `\end` {
		raw = append(raw, p.cur)
		p.advance()
	}
	rows := cdSplitRows(raw)

	body := [][]Node{}
	rowGaps := []*Measurement{}
	hlines := [][]bool{{}}
	for _, rowToks := range rows {
		if cdRowAllSpace(rowToks) {
			continue
		}
		cells, err := cdParseRow(p, rowToks)
		if err != nil {
			return nil, err
		}
		if len(cells) == 0 {
			continue
		}
		body = append(body, cells)
		rowGaps = append(rowGaps, nil)
		hlines = append(hlines, []bool{})
	}
	if len(body) == 0 {
		body = append(body, []Node{})
		hlines = append(hlines, []bool{})
	}
	colSep := "CD"
	isCD := true
	hskip := false
	return &Array{
		Mode:              p.mode,
		Body:              body,
		RowGaps:           rowGaps,
		HLinesBeforeRow:   hlines,
		ColSeparationType: &colSep,
		HskipBeforeAndAft: &hskip,
		ArrayStretch:      1.0,
		IsCD:              &isCD,
	}, nil
}

func cdRowAllSpace(toks []lexer.Token) bool {
	for _, t := range toks {
		if t.Text != " " {
			return false
		}
	}
	return true
}

// cdSplitRows splits a flat token list at `\\` / `\cr` boundaries.
func cdSplitRows(toks []lexer.Token) [][]lexer.Token {
	rows := [][]lexer.Token{}
	cur := []lexer.Token{}
	for _, t := range toks {
		if t.Text == `\\` || t.Text == `\cr` {
			rows = append(rows, cur)
			cur = nil
			continue
		}
		cur = append(cur, t)
	}
	if len(cur) > 0 {
		rows = append(rows, cur)
	}
	return rows
}

// cdCollectUntil walks tokens[start..] until it hits a token whose text equals
// delim. Returns (collected, count_of_tokens_consumed_including_delim).
func cdCollectUntil(toks []lexer.Token, start int, delim string) ([]lexer.Token, int) {
	out := []lexer.Token{}
	i := start
	for i < len(toks) {
		if toks[i].Text == delim {
			i++ // consume delim
			break
		}
		out = append(out, toks[i])
		i++
	}
	return out, i - start
}

// cdCollectUntilAt walks tokens[start..] until the next "@".
func cdCollectUntilAt(toks []lexer.Token, start int) ([]lexer.Token, int) {
	out := []lexer.Token{}
	i := start
	for i < len(toks) && toks[i].Text != "@" {
		out = append(out, toks[i])
		i++
	}
	return out, i - start
}

// cdParseTokens parses a forward-order token slice as a math expression and
// wraps the result in an OrdGroup. An all-space slice yields an empty
// OrdGroup, matching upstream's filter-then-emit shape.
func cdParseTokens(p *Parser, toks []lexer.Token) (Node, error) {
	hasContent := false
	for _, t := range toks {
		if t.Text != " " {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return &OrdGroup{Mode: p.mode, Body: []Node{}}, nil
	}
	body, err := subparse(p, toks)
	if err != nil {
		return nil, err
	}
	return &OrdGroup{Mode: p.mode, Body: body}, nil
}

// subparse runs the parser over a fixed token slice and returns the parsed
// expression. Tokens are pushed onto the existing parser's gullet stack
// (with a synthetic EOF terminator pushed first so parseExpression breaks
// at the end of the slice rather than falling through to the underlying
// lexer). Both the parser's lookahead and the lexer state are restored on
// return so the outer parse continues seamlessly.
func subparse(p *Parser, toks []lexer.Token) ([]Node, error) {
	savedCur := p.cur
	// Push synthetic EOF first (it will be consumed last).
	p.gullet.Push(lexer.EOF(0))
	// Push tokens in reverse so toks[0] is consumed first.
	for i := len(toks) - 1; i >= 0; i-- {
		p.gullet.Push(toks[i])
	}
	p.advance()
	body, err := p.parseExpression(false, "")
	if err != nil {
		p.cur = savedCur
		return nil, err
	}
	p.cur = savedCur
	return body, nil
}

// cdParseRow walks a row token slice, building CdArrow cells for `@dir...`
// segments and OrdGroup objects between them.
func cdParseRow(p *Parser, row []lexer.Token) ([]Node, error) {
	cells := []Node{}
	i := 0
	for i < len(row) {
		for i < len(row) && row[i].Text == " " {
			i++
		}
		if i >= len(row) {
			break
		}
		if row[i].Text == "@" {
			i++
			if i >= len(row) {
				return nil, errMsg("Unexpected end of CD row after @")
			}
			dir := row[i].Text
			i++
			cell, consumed, err := cdBuildArrow(p, dir, row, i)
			if err != nil {
				return nil, err
			}
			i += consumed
			cells = append(cells, cell)
		} else {
			objToks, consumed := cdCollectUntilAt(row, i)
			i += consumed
			obj, err := cdParseTokens(p, objToks)
			if err != nil {
				return nil, err
			}
			cells = append(cells, obj)
		}
	}
	return cdStructureRow(cells, p.mode), nil
}

func cdBuildArrow(p *Parser, dir string, row []lexer.Token, i int) (Node, int, error) {
	switch dir {
	case ">", "<":
		above, c1 := cdCollectUntil(row, i, dir)
		below, c2 := cdCollectUntil(row, i+c1, dir)
		la, err := cdParseTokens(p, above)
		if err != nil {
			return nil, 0, err
		}
		lb, err := cdParseTokens(p, below)
		if err != nil {
			return nil, 0, err
		}
		direction := "right"
		if dir == "<" {
			direction = "left"
		}
		return &CdArrow{
			Mode:       p.mode,
			Direction:  direction,
			LabelAbove: la,
			LabelBelow: lb,
		}, c1 + c2, nil
	case "V", "A":
		left, c1 := cdCollectUntil(row, i, dir)
		right, c2 := cdCollectUntil(row, i+c1, dir)
		la, err := cdParseTokens(p, left)
		if err != nil {
			return nil, 0, err
		}
		lb, err := cdParseTokens(p, right)
		if err != nil {
			return nil, 0, err
		}
		direction := "down"
		if dir == "A" {
			direction = "up"
		}
		return &CdArrow{
			Mode:       p.mode,
			Direction:  direction,
			LabelAbove: la,
			LabelBelow: lb,
		}, c1 + c2, nil
	case "=":
		return &CdArrow{Mode: p.mode, Direction: "horiz_eq"}, 0, nil
	case "|":
		return &CdArrow{Mode: p.mode, Direction: "vert_eq"}, 0, nil
	case ".":
		return &CdArrow{Mode: p.mode, Direction: "none"}, 0, nil
	}
	return nil, 0, errMsg("Unknown CD directive: @" + dir)
}

// cdStructureRow inserts empty OrdGroup fillers between consecutive arrow
// cells in an arrow-only row, matching upstream's grid shape.
func cdStructureRow(cells []Node, mode Mode) []Node {
	isArrowRow := false
	hasArrow := false
	allArrowOrEmptyOG := true
	for _, c := range cells {
		switch v := c.(type) {
		case *CdArrow:
			hasArrow = true
		case *OrdGroup:
			if len(v.Body) != 0 {
				allArrowOrEmptyOG = false
			}
		default:
			allArrowOrEmptyOG = false
		}
	}
	if hasArrow && allArrowOrEmptyOG {
		isArrowRow = true
	}
	if !isArrowRow {
		return cells
	}
	arrows := []Node{}
	for _, c := range cells {
		if _, ok := c.(*CdArrow); ok {
			arrows = append(arrows, c)
		}
	}
	if len(arrows) == 0 {
		return []Node{}
	}
	out := make([]Node, 0, len(arrows)*2-1)
	for i, a := range arrows {
		if i > 0 {
			out = append(out, &OrdGroup{Mode: mode, Body: []Node{}})
		}
		out = append(out, a)
	}
	return out
}
