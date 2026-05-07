package parser

import (
	"fmt"
	"strings"
)

// =============================================================================
// \begin{...} ... \end{...}
//
// This is a starting port of the upstream environments registry. It handles
// the matrix family (matrix / pmatrix / bmatrix / Bmatrix / vmatrix /
// Vmatrix), cases / dcases / rcases / drcases, gathered, and a stripped-
// down array. Other environments fall through to "unknown environment".
// =============================================================================

func parseEnvironment(p *Parser) (Node, error) {
	p.advance() // consume \begin
	name, err := readEnvName(p, `\begin`)
	if err != nil {
		return nil, err
	}
	body, err := envHandler(p, name)
	if err != nil {
		return nil, err
	}
	// After the env body finishes parsing, the parser must see \end{name}.
	if p.cur.Text != `\end` {
		return nil, errAt(fmt.Sprintf("Expected \\end{%s}, got %q", name, p.cur.Text), p.cur)
	}
	p.advance()
	endName, err := readEnvName(p, `\end`)
	if err != nil {
		return nil, err
	}
	if endName != name {
		return nil, errAt(fmt.Sprintf("Mismatched \\begin{%s} and \\end{%s}", name, endName), p.cur)
	}
	return body, nil
}

// readEnvName parses a `{name}` group following \begin or \end and returns
// the inner name string.
func readEnvName(p *Parser, ctx string) (string, error) {
	p.consumeSpaces()
	if p.cur.Text != "{" {
		return "", errAt(fmt.Sprintf("Expected {name} after %s", ctx), p.cur)
	}
	p.advance()
	var b strings.Builder
	for !p.cur.IsEOF() && p.cur.Text != "}" {
		b.WriteString(p.cur.Text)
		p.advance()
	}
	if p.cur.Text != "}" {
		return "", errAt(fmt.Sprintf("Unterminated %s name", ctx), p.cur)
	}
	p.advance()
	return b.String(), nil
}

// envHandler dispatches on the environment name. Returns the environment
// body node (without consuming \end).
func envHandler(p *Parser, name string) (Node, error) {
	base := strings.TrimSuffix(name, "*")
	switch base {
	case "matrix", "pmatrix", "bmatrix", "Bmatrix", "vmatrix", "Vmatrix":
		return handleMatrixEnv(p, name, base)
	case "smallmatrix":
		return handleSmallMatrixEnv(p, name)
	case "cases", "dcases", "rcases", "drcases":
		return handleCasesEnv(p, name)
	case "array", "darray":
		return handleArrayEnv(p, name, base)
	case "align", "aligned", "split", "alignat", "alignedat",
		"equation", "gather", "gathered":
		return handleAlignedEnv(p, name)
	case "subarray":
		return handleSubarrayEnv(p, name)
	case "CD":
		return handleCDEnv(p)
	}
	return nil, errAt(fmt.Sprintf("No such environment: %s", name), p.cur)
}

func handleSmallMatrixEnv(p *Parser, name string) (Node, error) {
	rows, rowGaps, hlines, err := parseArrayBody(p, StyleScript)
	if err != nil {
		return nil, err
	}
	smallSep := "small"
	return &Array{
		Mode:              p.mode,
		Body:              rows,
		RowGaps:           rowGaps,
		HLinesBeforeRow:   hlines,
		ColSeparationType: &smallSep,
		ArrayStretch:      0.5,
	}, nil
}

func handleAlignedEnv(p *Parser, name string) (Node, error) {
	base := strings.TrimSuffix(name, "*")
	isAlignat := strings.Contains(base, "at")
	isGather := base == "gather" || base == "gathered"
	isEquation := base == "equation"

	// alignat / alignedat take a {N} argument for column count. We read it
	// (and currently ignore it; it's only used to validate column counts).
	if isAlignat {
		p.consumeSpaces()
		if p.cur.Text == "{" {
			depth := 1
			p.advance()
			for !p.cur.IsEOF() && depth > 0 {
				if p.cur.Text == "{" {
					depth++
				} else if p.cur.Text == "}" {
					depth--
				}
				if depth > 0 {
					p.advance()
				}
			}
			if p.cur.Text == "}" {
				p.advance()
			}
		}
	}

	rows, rowGaps, hlines, err := parseArrayBody(p, StyleDisplay)
	if err != nil {
		return nil, err
	}

	if isEquation {
		var tags []ArrayTag
		if !strings.HasSuffix(name, "*") {
			tags = make([]ArrayTag, len(rows))
			for i := range tags {
				p.equationCounter++
				num := fmt.Sprintf("%d", p.equationCounter)
				tags[i] = ArrayTag{Explicit: []Node{
					&MathOrd{Mode: ModeMath, Text: "("},
					&MathOrd{Mode: ModeMath, Text: num},
					&MathOrd{Mode: ModeMath, Text: ")"},
				}}
			}
		}
		return &Array{
			Mode:            p.mode,
			Body:            rows,
			RowGaps:         rowGaps,
			HLinesBeforeRow: hlines,
			ArrayStretch:    1.0,
			Tags:            tags,
		}, nil
	}
	if isGather {
		c := "c"
		cols := []AlignSpec{{Type: AlignTypeAlign, Align: &c}}
		gatherSep := "gather"
		return &Array{
			Mode:              p.mode,
			Body:              rows,
			RowGaps:           rowGaps,
			HLinesBeforeRow:   hlines,
			Cols:              cols,
			AddJot:            boolPtr(true),
			ColSeparationType: &gatherSep,
			ArrayStretch:      1.0,
		}, nil
	}

	// align / aligned / split / alignat / alignedat:
	// upstream prepends an empty ordgroup at the start of every even-indexed
	// cell (2nd, 4th, …) and computes alternating r-l columns.
	for _, row := range rows {
		for i := 1; i < len(row); i += 2 {
			if styled, ok := row[i].(*Styling); ok && len(styled.Body) > 0 {
				if og, ok2 := styled.Body[0].(*OrdGroup); ok2 {
					og.Body = append([]Node{&OrdGroup{Mode: p.mode}}, og.Body...)
				}
			}
		}
	}
	numCols := rowMaxCols(rows)
	cols := make([]AlignSpec, numCols)
	zero, one := 0.0, 1.0
	isAligned := !isAlignat
	for i := 0; i < numCols; i++ {
		var align string
		var pregap *float64
		if i%2 == 1 {
			align = "l"
			pregap = &zero
		} else if i > 0 && isAligned {
			align = "r"
			pregap = &one
		} else {
			align = "r"
			pregap = &zero
		}
		al := align
		cols[i] = AlignSpec{Type: AlignTypeAlign, Align: &al, Pregap: pregap, Postgap: &zero}
	}
	sepType := "align"
	if isAlignat {
		sepType = "alignat"
	}
	return &Array{
		Mode:              p.mode,
		Body:              rows,
		RowGaps:           rowGaps,
		HLinesBeforeRow:   hlines,
		AddJot:            boolPtr(true),
		Cols:              cols,
		ColSeparationType: &sepType,
		ArrayStretch:      1.0,
	}, nil
}

func handleSubarrayEnv(p *Parser, name string) (Node, error) {
	// Read column spec.
	p.consumeSpaces()
	if p.cur.Text != "{" {
		return nil, errAt("Expected column spec after \\begin{subarray}", p.cur)
	}
	p.advance()
	var spec strings.Builder
	for !p.cur.IsEOF() && p.cur.Text != "}" {
		spec.WriteString(p.cur.Text)
		p.advance()
	}
	if p.cur.Text == "}" {
		p.advance()
	}
	cols := parseColumnSpec(spec.String())
	rows, rowGaps, hlines, err := parseArrayBody(p, StyleScript)
	if err != nil {
		return nil, err
	}
	return &Array{
		Mode:            p.mode,
		Body:            rows,
		RowGaps:         rowGaps,
		HLinesBeforeRow: hlines,
		Cols:            cols,
		ArrayStretch:    0.5,
	}, nil
}

// dCellStyle returns "display" if env name starts with 'd', else "text".
func dCellStyle(name string) StyleStr {
	if len(name) > 0 && name[0] == 'd' {
		return StyleDisplay
	}
	return StyleText
}

// parseArrayBody collects rows of cells until \end is seen, also tracking
// per-row \hline / \hdashline markers. Returns rows, rowGaps, and the
// (rows+1)-element hLinesBeforeRow slice (each entry is a list of bools
// where true means \hdashline, false means \hline).
//
// Each cell is wrapped in `Styling { style: cellStyle, body: [OrdGroup{...}] }`,
// matching upstream's parse_array.
func parseArrayBody(p *Parser, cellStyle StyleStr) ([][]Node, []*Measurement, [][]bool, error) {
	rows := [][]Node{}
	rowGaps := []*Measurement{}
	hlines := [][]bool{}
	hlines = append(hlines, readHlines(p))
	for {
		if p.cur.Text == `\end` || p.cur.IsEOF() {
			break
		}
		row, err := parseArrayRow(p, cellStyle)
		if err != nil {
			return nil, nil, nil, err
		}
		rows = append(rows, row)
		if p.cur.Text == `\\` || p.cur.Text == `\cr` {
			p.advance()
			rowGaps = append(rowGaps, nil)
			hlines = append(hlines, readHlines(p))
			continue
		}
		break
	}
	if len(hlines) == len(rows) {
		hlines = append(hlines, []bool{})
	}
	return rows, rowGaps, hlines, nil
}

// readHlines consumes a run of \hline / \hdashline / \relax tokens and
// returns a slice of bools (true = \hdashline, false = \hline). Mirrors
// upstream's get_hlines.
func readHlines(p *Parser) []bool {
	out := []bool{}
	p.consumeSpaces()
	for p.cur.Text == `\relax` {
		p.advance()
		p.consumeSpaces()
	}
	for p.cur.Text == `\hline` || p.cur.Text == `\hdashline` {
		out = append(out, p.cur.Text == `\hdashline`)
		p.advance()
		p.consumeSpaces()
	}
	return out
}

func parseArrayRow(p *Parser, cellStyle StyleStr) ([]Node, error) {
	row := []Node{}
	for {
		p.consumeSpaces()
		// Parse one cell as an expression that breaks on `\\`, `&`, or
		// `\end`. `&` is part of endOfExpression; `\end` too. We pass `\\`
		// as breakOnText so we can detect it and continue to the next row.
		cellBody, err := p.parseExpression(false, `\\`)
		if err != nil {
			return nil, err
		}
		cellOrd := &OrdGroup{Mode: p.mode, Body: cellBody}
		styled := &Styling{Mode: p.mode, Style: cellStyle, Body: []Node{cellOrd}}
		row = append(row, styled)
		if p.cur.Text == "&" {
			p.advance()
			continue
		}
		break
	}
	return row, nil
}

// handleMatrixEnv handles the matrix family. Wraps the body in a LeftRight
// when the env carries delimiters (pmatrix/bmatrix/etc.).
func handleMatrixEnv(p *Parser, fullName, base string) (Node, error) {
	delims := matrixDelims(base)
	cellStyle := dCellStyle(fullName)

	// Starred variants (matrix*, pmatrix*, ...) accept an optional
	// [l|c|r] alignment argument before the body.
	colAlign := "c"
	if strings.HasSuffix(fullName, "*") {
		p.consumeSpaces()
		if p.cur.Text == "[" {
			p.advance()
			p.consumeSpaces()
			if t := p.cur.Text; t == "l" || t == "c" || t == "r" {
				colAlign = t
				p.advance()
			}
			p.consumeSpaces()
			if p.cur.Text == "]" {
				p.advance()
			}
		}
	}

	rows, rowGaps, hlines, err := parseArrayBody(p, cellStyle)
	if err != nil {
		return nil, err
	}
	numCols := rowMaxCols(rows)
	cols := make([]AlignSpec, numCols)
	for i := range cols {
		c := colAlign
		cols[i] = AlignSpec{Type: AlignTypeAlign, Align: &c}
	}
	arr := &Array{
		Mode:            p.mode,
		Body:            rows,
		RowGaps:         rowGaps,
		HLinesBeforeRow: hlines,
		Cols:            cols,
		HskipBeforeAndAft: boolPtr(false),
		ArrayStretch:    1.0,
	}
	if delims == [2]string{} {
		return arr, nil
	}
	return &LeftRight{
		Mode: p.mode, Body: []Node{arr},
		Left: delims[0], Right: delims[1],
	}, nil
}

func matrixDelims(base string) [2]string {
	switch base {
	case "pmatrix":
		return [2]string{"(", ")"}
	case "bmatrix":
		return [2]string{"[", "]"}
	case "Bmatrix":
		return [2]string{`\{`, `\}`}
	case "vmatrix":
		return [2]string{"|", "|"}
	case "Vmatrix":
		return [2]string{`\Vert`, `\Vert`}
	}
	return [2]string{}
}

func handleCasesEnv(p *Parser, name string) (Node, error) {
	cellStyle := dCellStyle(name)
	rows, rowGaps, hlines, err := parseArrayBody(p, cellStyle)
	if err != nil {
		return nil, err
	}
	zero, one := 0.0, 1.0
	cols := []AlignSpec{
		{Type: AlignTypeAlign, Align: strPtr("l"), Pregap: &zero, Postgap: &one},
		{Type: AlignTypeAlign, Align: strPtr("l"), Pregap: &zero, Postgap: &zero},
	}
	left, right := `\{`, `.`
	if strings.Contains(name, "r") {
		left, right = `.`, `\}`
	}
	arr := &Array{
		Mode:            p.mode,
		Body:            rows,
		RowGaps:         rowGaps,
		HLinesBeforeRow: hlines,
		Cols:            cols,
		ArrayStretch:    1.2,
	}
	return &LeftRight{
		Mode: p.mode, Body: []Node{arr},
		Left: left, Right: right,
	}, nil
}

func handleArrayEnv(p *Parser, fullName, base string) (Node, error) {
	p.consumeSpaces()
	if p.cur.Text != "{" {
		return nil, errAt(fmt.Sprintf("Expected column spec after \\begin{%s}", fullName), p.cur)
	}
	p.advance()
	var spec strings.Builder
	for !p.cur.IsEOF() && p.cur.Text != "}" {
		spec.WriteString(p.cur.Text)
		p.advance()
	}
	if p.cur.Text != "}" {
		return nil, errAt("Unterminated column spec", p.cur)
	}
	p.advance()
	cols := parseColumnSpec(spec.String())
	cellStyle := dCellStyle(fullName)
	if base == "darray" {
		cellStyle = StyleDisplay
	}
	rows, rowGaps, hlines, err := parseArrayBody(p, cellStyle)
	if err != nil {
		return nil, err
	}
	return &Array{
		Mode:              p.mode,
		Body:              rows,
		RowGaps:           rowGaps,
		HLinesBeforeRow:   hlines,
		Cols:              cols,
		HskipBeforeAndAft: boolPtr(true),
		ArrayStretch:      arrayStretchFromMacro(p, 1.0),
	}, nil
}

// arrayStretchFromMacro returns the value of the \arraystretch macro if
// defined (parsed as a float), or fallback otherwise. Mirrors upstream's
// `\arraystretch` lookup at array-build time.
func arrayStretchFromMacro(p *Parser, fallback float64) float64 {
	body, ok := p.gullet.MacroBodyText(`\arraystretch`)
	if !ok {
		return fallback
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return fallback
	}
	var n float64
	if _, err := fmt.Sscanf(body, "%f", &n); err == nil && n > 0 {
		return n
	}
	return fallback
}

// parseColumnSpec turns "ccr|l" into a list of AlignSpecs (with separators).
func parseColumnSpec(spec string) []AlignSpec {
	out := []AlignSpec{}
	for _, c := range spec {
		switch c {
		case 'l':
			s := "l"
			out = append(out, AlignSpec{Type: AlignTypeAlign, Align: &s})
		case 'c':
			s := "c"
			out = append(out, AlignSpec{Type: AlignTypeAlign, Align: &s})
		case 'r':
			s := "r"
			out = append(out, AlignSpec{Type: AlignTypeAlign, Align: &s})
		case '|':
			out = append(out, AlignSpec{Type: AlignTypeSeparator, Align: strPtr("|")})
		case ':':
			out = append(out, AlignSpec{Type: AlignTypeSeparator, Align: strPtr(":")})
		}
	}
	return out
}

func arrayCenterCols(n int) []AlignSpec {
	out := make([]AlignSpec, n)
	c := "c"
	for i := range out {
		out[i] = AlignSpec{Type: AlignTypeAlign, Align: &c}
	}
	return out
}

func rowMaxCols(rows [][]Node) int {
	max := 0
	for _, r := range rows {
		if len(r) > max {
			max = len(r)
		}
	}
	return max
}

func buildEmptyHLines(rows int) [][]bool {
	out := make([][]bool, rows+1)
	for i := range out {
		out[i] = []bool{}
	}
	return out
}

func boolPtr(b bool) *bool       { return &b }
func strPtr(s string) *string    { return &s }
