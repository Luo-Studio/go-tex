package parser

// =============================================================================
// Op-style commands: \sum, \int, \lim, \sin, ...
// =============================================================================

// opSymbolLimits: big operators that take limits, rendered as a symbol
// (e.g. \sum, \prod, \bigvee).
var opSymbolLimits = map[string]struct{}{
	`\coprod`: {}, `\bigvee`: {}, `\bigwedge`: {}, `\biguplus`: {},
	`\bigcap`: {}, `\bigcup`: {}, `\intop`: {}, `\prod`: {}, `\sum`: {},
	`\bigotimes`: {}, `\bigoplus`: {}, `\bigodot`: {}, `\bigsqcup`: {},
	`\smallint`: {},
	"∏":   {}, "∐": {}, "∑": {}, "⋀": {}, "⋁": {},
	"⋂":   {}, "⋃": {}, "⨀": {}, "⨁": {}, "⨂": {},
	"⨄":   {}, "⨆": {},
}

// opSymbolNolimits: big operators that don't take limits, rendered as a
// symbol (integrals).
var opSymbolNolimits = map[string]struct{}{
	`\int`: {}, `\iint`: {}, `\iiint`: {}, `\oint`: {}, `\oiint`: {}, `\oiiint`: {},
}

// opTextNolimits: textual operators without limits (\sin, \log, \exp, …).
var opTextNolimits = map[string]struct{}{
	`\arcsin`: {}, `\arccos`: {}, `\arctan`: {}, `\arctg`: {}, `\arcctg`: {},
	`\arg`: {}, `\ch`: {}, `\cos`: {}, `\cosec`: {}, `\cosh`: {}, `\cot`: {},
	`\cotg`: {}, `\coth`: {}, `\csc`: {}, `\ctg`: {}, `\cth`: {}, `\deg`: {},
	`\dim`: {}, `\exp`: {}, `\hom`: {}, `\ker`: {}, `\lg`: {}, `\ln`: {},
	`\log`: {}, `\sec`: {}, `\sin`: {}, `\sinh`: {}, `\sh`: {}, `\tan`: {},
	`\tanh`: {}, `\tg`: {}, `\th`: {},
}

// opTextLimits: textual operators with limits (\det, \lim, \max, …).
var opTextLimits = map[string]struct{}{
	`\det`: {}, `\gcd`: {}, `\inf`: {}, `\lim`: {}, `\max`: {},
	`\min`: {}, `\Pr`: {}, `\sup`: {},
}

// singleCharBigOp maps Unicode big-operator characters to their TeX names.
var singleCharBigOp = map[string]string{
	"∏": `\prod`, "∐": `\coprod`, "∑": `\sum`,
	"⋀": `\bigwedge`, "⋁": `\bigvee`, "⋂": `\bigcap`,
	"⋃": `\bigcup`, "⨀": `\bigodot`, "⨁": `\bigoplus`,
	"⨂": `\bigotimes`, "⨄": `\biguplus`, "⨆": `\bigsqcup`,
}

func parseOpFunction(p *Parser) (Node, bool) {
	cmd := p.cur.Text
	switch {
	case isIn(opSymbolLimits, cmd):
		p.advance()
		name := cmd
		if mapped, ok := singleCharBigOp[cmd]; ok {
			name = mapped
		}
		return &Op{
			Mode:           p.mode,
			Limits:         true,
			ParentIsSupSub: false,
			Symbol:         true,
			Name:           &name,
		}, true
	case isIn(opSymbolNolimits, cmd):
		p.advance()
		name := cmd
		return &Op{
			Mode:           p.mode,
			Limits:         false,
			ParentIsSupSub: false,
			Symbol:         true,
			Name:           &name,
		}, true
	case isIn(opTextNolimits, cmd):
		p.advance()
		name := cmd
		return &Op{
			Mode:           p.mode,
			Limits:         false,
			ParentIsSupSub: false,
			Symbol:         false,
			Name:           &name,
		}, true
	case isIn(opTextLimits, cmd):
		p.advance()
		name := cmd
		return &Op{
			Mode:           p.mode,
			Limits:         true,
			ParentIsSupSub: false,
			Symbol:         false,
			Name:           &name,
		}, true
	}
	return nil, false
}

// parseMathOp handles `\mathop{...}`.
func parseMathOp(p *Parser) (Node, error) {
	p.advance()
	arg, err := parseFunctionArg(p, `\mathop`)
	if err != nil {
		return nil, err
	}
	body := OrdArgument(arg)
	return &Op{
		Mode:           p.mode,
		Limits:         false,
		ParentIsSupSub: false,
		Symbol:         false,
		Body:           body,
	}, nil
}

func isIn(m map[string]struct{}, k string) bool { _, ok := m[k]; return ok }
