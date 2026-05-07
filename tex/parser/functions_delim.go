package parser

import "fmt"

// =============================================================================
// \big, \Big, \bigl, \Bigr, \biggm, ... — fixed-size delimiters.
// =============================================================================

type delimSizeInfo struct {
	mclass string
	size   uint8
}

var delimSizeTable = map[string]delimSizeInfo{
	`\bigl`: {"mopen", 1}, `\Bigl`: {"mopen", 2}, `\biggl`: {"mopen", 3}, `\Biggl`: {"mopen", 4},
	`\bigr`: {"mclose", 1}, `\Bigr`: {"mclose", 2}, `\biggr`: {"mclose", 3}, `\Biggr`: {"mclose", 4},
	`\bigm`: {"mrel", 1}, `\Bigm`: {"mrel", 2}, `\biggm`: {"mrel", 3}, `\Biggm`: {"mrel", 4},
	`\big`: {"mord", 1}, `\Big`: {"mord", 2}, `\bigg`: {"mord", 3}, `\Bigg`: {"mord", 4},
}

func parseDelimSizing(p *Parser) (Node, error) {
	cmd := p.cur.Text
	info := delimSizeTable[cmd]
	p.advance()
	p.consumeSpaces()
	delimTok := p.cur
	if delimTok.IsEOF() {
		return nil, errAt(fmt.Sprintf("Invalid delimiter type 'EOF' after '%s'", cmd), delimTok)
	}
	if !isValidDelimiter(delimTok.Text) {
		return nil, errAt(fmt.Sprintf("Invalid delimiter '%s' after '%s'", delimTok.Text, cmd), delimTok)
	}
	p.advance()
	return &DelimSizing{
		Mode:   p.mode,
		Size:   info.size,
		Mclass: info.mclass,
		Delim:  delimTok.Text,
	}, nil
}

func isValidDelimiter(t string) bool {
	switch t {
	case "(", `\lparen`, ")", `\rparen`,
		"[", `\lbrack`, "]", `\rbrack`,
		`\{`, `\lbrace`, `\}`, `\rbrace`,
		`\lfloor`, `\rfloor`,
		`\lceil`, `\rceil`,
		"<", ">", `\langle`, `\rangle`, `\lt`, `\gt`,
		`\lvert`, `\rvert`, `\lVert`, `\rVert`,
		`\lgroup`, `\rgroup`,
		`\lmoustache`, `\rmoustache`,
		"/", `\backslash`,
		"|", `\vert`, `\|`, `\Vert`,
		`\uparrow`, `\Uparrow`,
		`\downarrow`, `\Downarrow`,
		`\updownarrow`, `\Updownarrow`,
		".":
		return true
	}
	return false
}

func isDelimSizing(cmd string) bool {
	_, ok := delimSizeTable[cmd]
	return ok
}
