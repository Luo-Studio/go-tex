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
	case "cases", "dcases", "rcases", "drcases":
		return handleCasesEnv(p, name)
	case "array", "darray":
		return handleArrayEnv(p, name, base)
	}
	return nil, errAt(fmt.Sprintf("No such environment: %s", name), p.cur)
}

// dCellStyle returns "display" if env name starts with 'd', else "text".
func dCellStyle(name string) StyleStr {
	if len(name) > 0 && name[0] == 'd' {
		return StyleDisplay
	}
	return StyleText
}

// parseArrayBody collects rows of cells until \end is seen.
//
//	row    := cell ('&' cell)*
//	rows   := row ('\\\\' row)*
//	body   := rows
//
// Each cell is wrapped in `Styling { style: cellStyle, body: [OrdGroup{...}] }`,
// matching upstream's parse_array.
func parseArrayBody(p *Parser, cellStyle StyleStr) ([][]Node, []*Measurement, error) {
	rows := [][]Node{}
	rowGaps := []*Measurement{}
	for {
		p.consumeSpaces()
		for p.cur.Text == `\hline` || p.cur.Text == `\hdashline` || p.cur.Text == `\relax` {
			p.advance()
			p.consumeSpaces()
		}
		if p.cur.Text == `\end` || p.cur.IsEOF() {
			break
		}
		row, err := parseArrayRow(p, cellStyle)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
		if p.cur.Text == `\\` {
			p.advance()
			rowGaps = append(rowGaps, nil)
			continue
		}
		break
	}
	return rows, rowGaps, nil
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
	rows, rowGaps, err := parseArrayBody(p, cellStyle)
	if err != nil {
		return nil, err
	}
	cols := arrayCenterCols(rowMaxCols(rows))
	hlines := buildEmptyHLines(len(rows))
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
	rows, rowGaps, err := parseArrayBody(p, cellStyle)
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
		HLinesBeforeRow: buildEmptyHLines(len(rows)),
		Cols:            cols,
		ArrayStretch:    1.2,
	}
	return &LeftRight{
		Mode: p.mode, Body: []Node{arr},
		Left: left, Right: right,
	}, nil
}

func handleArrayEnv(p *Parser, fullName, base string) (Node, error) {
	// Read the column spec argument.
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
	rows, rowGaps, err := parseArrayBody(p, cellStyle)
	if err != nil {
		return nil, err
	}
	return &Array{
		Mode:            p.mode,
		Body:            rows,
		RowGaps:         rowGaps,
		HLinesBeforeRow: buildEmptyHLines(len(rows)),
		Cols:            cols,
		HskipBeforeAndAft: boolPtr(true),
		ArrayStretch:    1.0,
	}, nil
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
