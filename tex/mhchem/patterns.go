package mhchem

import (
	"fmt"
	"sync"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

// MatchToken is the result of matching one mhchem transition pattern.
// Either a single string (S) or an ordered tuple of substrings (A).
type MatchToken struct {
	S string
	A []string
	// IsArray distinguishes the two variants. Using a bool keeps the
	// type comparable and lets callers branch without a tagged union.
	IsArray bool
}

func tokS(s string) MatchToken         { return MatchToken{S: s} }
func tokA(parts ...string) MatchToken  { return MatchToken{A: parts, IsArray: true} }

// PatternHit is one successful pattern match: the captured token plus
// whatever input remained after it.
type PatternHit struct {
	Token     MatchToken
	Remainder string
}

// beg / end are simple sum-types: either a literal string or a regex.
// Empty string == "no anchor".
type beg struct {
	s  string
	re *regexp2.Regexp
}
type end = beg

func bStr(s string) beg            { return beg{s: s} }
func bRe(re *regexp2.Regexp) beg   { return beg{re: re} }
func eStr(s string) end            { return end{s: s} }
func eRe(re *regexp2.Regexp) end   { return end{re: re} }

// prefixLen returns (len, true) if input starts with `b`. For Beg::Re
// only the leading anchored match matters.
func prefixLen(input string, b beg) (int, bool) {
	if b.re != nil {
		m, _ := b.re.FindStringMatch(input)
		if m == nil || m.Index != 0 {
			return 0, false
		}
		return m.Length, true
	}
	if b.s == "" {
		return 0, true
	}
	if len(input) >= len(b.s) && input[:len(b.s)] == b.s {
		return len(b.s), true
	}
	return 0, false
}

// scanEnd advances through `input` from `start`, looking for the first
// position where `e` matches at depth-zero (i.e. with all { } balanced).
// Returns (matchStart, matchEnd) on success, or (0,0,false).
func scanEnd(input string, start int, e end) (int, int, bool, error) {
	i := start
	braces := 0
	for i < len(input) {
		rest := input[i:]
		var matchLen int
		var matched bool
		if e.re != nil {
			m, _ := e.re.FindStringMatch(rest)
			if m != nil && m.Index == 0 {
				matchLen = m.Length
				matched = true
			}
		} else if e.s != "" {
			if len(rest) >= len(e.s) && rest[:len(e.s)] == e.s {
				matchLen = len(e.s)
				matched = true
			}
		}
		if matched && braces == 0 {
			return i, i + matchLen, true, nil
		}
		r, w := utf8.DecodeRuneInString(rest)
		if r == utf8.RuneError && w <= 1 {
			return 0, 0, false, errMsg("invalid utf-8 in scanEnd")
		}
		if r == '{' {
			braces++
		} else if r == '}' {
			if braces == 0 {
				return 0, 0, false, errExtraClose
			}
			braces--
		}
		i += w
	}
	return 0, 0, false, nil
}

// findObserveGroups is the KaTeX `findObserveGroups` helper. See the
// pattern table in matchPattern for how the two-part variant is used.
func findObserveGroups(
	input, begExcl string,
	begIncl beg,
	endIncl string,
	endExcl end,
	part2 *fogPart2,
	combine bool,
) (*PatternHit, error) {
	ex := 0
	if begExcl != "" {
		n, ok := prefixLen(input, bStr(begExcl))
		if !ok {
			return nil, nil
		}
		ex = n
	}
	rest1 := input[ex:]
	open, ok := prefixLen(rest1, begIncl)
	if !ok {
		return nil, nil
	}
	endMain := endExcl
	if endIncl != "" {
		endMain = eStr(endIncl)
	}
	matchStart, matchEnd, found, err := scanEnd(rest1, open, endMain)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	useIncl := endIncl != ""
	cut := matchStart
	if useIncl {
		cut = matchEnd
	}
	m1 := rest1[:cut]
	after := rest1[matchEnd:]

	if part2 != nil {
		hit, err := findObserveGroups(after, part2.begExcl, part2.begIncl, part2.endIncl, part2.endExcl, nil, false)
		if err != nil {
			return nil, err
		}
		if hit == nil {
			return nil, nil
		}
		if hit.Token.IsArray {
			return nil, errMsg("nested Fog pair")
		}
		m2 := hit.Token.S
		var tok MatchToken
		if combine {
			tok = tokS(m1 + m2)
		} else {
			tok = tokA(m1, m2)
		}
		return &PatternHit{Token: tok, Remainder: hit.Remainder}, nil
	}
	return &PatternHit{Token: tokS(m1), Remainder: after}, nil
}

type fogPart2 struct {
	begExcl string
	begIncl beg
	endIncl string
	endExcl end
}

// regexMatchToken runs the given regex against input. If the pattern
// has capture group 2 (and it's non-empty), returns an A-token of
// (group1, group2). Otherwise returns S of group1 (or full match if
// no group 1).
func regexMatchToken(re *regexp2.Regexp, input string) (MatchToken, int, bool) {
	m, _ := re.FindStringMatch(input)
	if m == nil {
		return MatchToken{}, 0, false
	}
	end := m.Index + m.Length
	g := m.Groups()
	if len(g) > 2 && g[2].String() != "" {
		g1 := ""
		if len(g) > 1 {
			g1 = g[1].String()
		}
		return tokA(g1, g[2].String()), end, true
	}
	if len(g) > 1 && g[1].Length > 0 {
		return tokS(g[1].String()), end, true
	}
	return tokS(m.String()), end, true
}

// Compile-once cache of the inline regex literals that some patterns
// embed (KaTeX uses static lazy_locks for these).
var (
	reBuildOnce sync.Once
	reAggOpen          *regexp2.Regexp
	reCmdBrace         *regexp2.Regexp
	reBeforeBrace      *regexp2.Regexp
	reNegPow           *regexp2.Regexp
	reSci              *regexp2.Regexp
	reSoaRemainder     *regexp2.Regexp
	reSoaAlt           *regexp2.Regexp
	reAmountMain       *regexp2.Regexp
	reAmountDollar     *regexp2.Regexp
	reFormulaParenOnly *regexp2.Regexp
	reFormulaMain      *regexp2.Regexp
)

func mustCompile(src string) *regexp2.Regexp {
	r, err := regexp2.Compile(src, regexp2.ECMAScript)
	if err != nil {
		panic(fmt.Sprintf("mhchem: pattern %q: %v", src, err))
	}
	return r
}

func buildRegexes() {
	reBuildOnce.Do(func() {
		reAggOpen = mustCompile(`^\([a-z]{1,3}(?=[\),])`)
		reCmdBrace = mustCompile(`^\\[a-zA-Z]+\{`)
		reBeforeBrace = mustCompile(`^(?=\{)`)
		reNegPow = mustCompile(`^(\+\-|\+\/\-|\+|\-|\\pm\s?)?([0-9]+(?:[,.][0-9]+)?|[0-9]*(?:\.[0-9]+)?)\^([+\-]?[0-9]+|\{[+\-]?[0-9]+\})`)
		reSci = mustCompile(`^(\+\-|\+\/\-|\+|\-|\\pm\s?)?([0-9]+(?:[,.][0-9]+)?|[0-9]*(?:\.[0-9]+))?(\((?:[0-9]+(?:[,.][0-9]+)?|[0-9]*(?:\.[0-9]+))\))?(?:([eE]|\s*(\*|x|\\times|×)\s*10\^)([+\-]?[0-9]+|\{[+\-]?[0-9]+\}))?`)
		reSoaRemainder = mustCompile(`^($|[\s,;\)\]\}])`)
		reSoaAlt = mustCompile(`^(?:\((?:\\ca\s?)?\$[amothc]\$\))`)
		reAmountMain = mustCompile(`^(?:(?:(?:\([+\-]?[0-9]+\/[0-9]+\)|[+\-]?(?:[0-9]+|\$[a-z]\$|[a-z])\/[0-9]+|[+\-]?[0-9]+[.,][0-9]+|[+\-]?\.[0-9]+|[+\-]?[0-9]+)(?:[a-z](?=\s*[A-Z]))?)|[+\-]?[a-z](?=\s*[A-Z])|\+(?!\s))`)
		reAmountDollar = mustCompile(`^\$(?:\(?[+\-]?(?:[0-9]*[a-z]?[+\-])?[0-9]*[a-z](?:[+\-][0-9]*[a-z]?)?\)?|\+|-)\$$`)
		reFormulaParenOnly = mustCompile(`^\([a-z]+\)$`)
		reFormulaMain = mustCompile(`^(?:[a-z]|(?:[0-9\ \+\-\,\.\(\)]+[a-z])+[0-9\ \+\-\,\.\(\)]*|(?:[a-z][0-9\ \+\-\,\.\(\)]+)+[a-z]?)`)
	})
}

func patternNegPow(input string) (*PatternHit, error) {
	buildRegexes()
	m, _ := reNegPow.FindStringMatch(input)
	if m == nil {
		return nil, nil
	}
	g := m.Groups()
	v := make([]string, 0, len(g)-1)
	for i := 1; i < len(g); i++ {
		v = append(v, g[i].String())
	}
	end := m.Index + m.Length
	return &PatternHit{Token: tokA(v...), Remainder: input[end:]}, nil
}

func patternSci(input string) (*PatternHit, error) {
	buildRegexes()
	m, _ := reSci.FindStringMatch(input)
	if m == nil {
		return nil, nil
	}
	if m.String() == "" {
		return nil, nil
	}
	g := m.Groups()
	v := make([]string, 0, len(g)-1)
	for i := 1; i < len(g); i++ {
		v = append(v, g[i].String())
	}
	end := m.Index + m.Length
	return &PatternHit{Token: tokA(v...), Remainder: input[end:]}, nil
}

func patternStateOfAgg(input string) (*PatternHit, error) {
	buildRegexes()
	hit, err := findObserveGroups(input, "", bRe(reAggOpen), ")", eStr(""), nil, false)
	if err != nil {
		return nil, err
	}
	if hit != nil {
		m, _ := reSoaRemainder.FindStringMatch(hit.Remainder)
		if m != nil && m.Index == 0 {
			return hit, nil
		}
	}
	m, _ := reSoaAlt.FindStringMatch(input)
	if m != nil {
		end := m.Index + m.Length
		return &PatternHit{Token: tokS(m.String()), Remainder: input[end:]}, nil
	}
	return nil, nil
}

func patternAmount(input string) (*PatternHit, error) {
	buildRegexes()
	if m, _ := reAmountMain.FindStringMatch(input); m != nil {
		end := m.Index + m.Length
		return &PatternHit{Token: tokS(m.String()), Remainder: input[end:]}, nil
	}
	hit, err := findObserveGroups(input, "", bStr("$"), "$", eStr(""), nil, false)
	if err != nil || hit == nil {
		return nil, err
	}
	if hit.Token.IsArray {
		return nil, nil
	}
	if m, _ := reAmountDollar.FindStringMatch(hit.Token.S); m != nil {
		return &PatternHit{Token: tokS(hit.Token.S), Remainder: hit.Remainder}, nil
	}
	return nil, nil
}

func patternFormula(input string) (*PatternHit, error) {
	buildRegexes()
	if m, _ := reFormulaParenOnly.FindStringMatch(input); m != nil {
		return nil, nil
	}
	m, _ := reFormulaMain.FindStringMatch(input)
	if m == nil {
		return nil, nil
	}
	end := m.Index + m.Length
	return &PatternHit{Token: tokS(m.String()), Remainder: input[end:]}, nil
}

// matchPattern runs the named pattern against input. Mirrors upstream's
// match_pattern dispatcher. Patterns with bespoke logic are listed
// explicitly; everything else falls through to the precompiled regexp2
// table loaded from patterns_regex.json.
func matchPattern(data *Data, name, input string) (*PatternHit, error) {
	buildRegexes()
	switch name {
	case "(-)(9)^(-9)":
		return patternNegPow(input)
	case "(-)(9.,9)(e)(99)":
		return patternSci(input)
	case "state of aggregation $":
		return patternStateOfAgg(input)
	case "amount", "amount2":
		return patternAmount(input)
	case "formula$":
		return patternFormula(input)
	case "^{(...)}":
		return findObserveGroups(input, "^{", bStr(""), "", eStr("}"), nil, false)
	case "^($...$)":
		return findObserveGroups(input, "^", bStr("$"), "$", eStr(""), nil, false)
	case "^\\x{}{}":
		return findObserveGroups(input, "^", bRe(reCmdBrace), "}", eStr(""),
			&fogPart2{"", bStr("{"), "}", eStr("")}, true)
	case "^\\x{}":
		return findObserveGroups(input, "^", bRe(reCmdBrace), "}", eStr(""), nil, false)
	case "_{(...)}":
		return findObserveGroups(input, "_{", bStr(""), "", eStr("}"), nil, false)
	case "_($...$)":
		return findObserveGroups(input, "_", bStr("$"), "$", eStr(""), nil, false)
	case "_\\x{}{}":
		return findObserveGroups(input, "_", bRe(reCmdBrace), "}", eStr(""),
			&fogPart2{"", bStr("{"), "}", eStr("")}, true)
	case "_\\x{}":
		return findObserveGroups(input, "_", bRe(reCmdBrace), "}", eStr(""), nil, false)
	case "{...}":
		return findObserveGroups(input, "", bStr("{"), "}", eStr(""), nil, false)
	case "{(...)}":
		return findObserveGroups(input, "{", bStr(""), "", eStr("}"), nil, false)
	case "$...$":
		return findObserveGroups(input, "", bStr("$"), "$", eStr(""), nil, false)
	case "${(...)}$":
		return findObserveGroups(input, "${", bStr(""), "", eStr("}$"), nil, false)
	case "$(...)$":
		return findObserveGroups(input, "$", bStr(""), "", eStr("$"), nil, false)
	case `\bond{(...)}`:
		return findObserveGroups(input, `\bond{`, bStr(""), "", eStr("}"), nil, false)
	case "[(...)]":
		return findObserveGroups(input, "[", bStr(""), "", eStr("]"), nil, false)
	case `\x{}{}`:
		return findObserveGroups(input, "", bRe(reCmdBrace), "}", eStr(""),
			&fogPart2{"", bStr("{"), "}", eStr("")}, true)
	case `\x{}`:
		return findObserveGroups(input, "", bRe(reCmdBrace), "}", eStr(""), nil, false)
	case `\frac{(...)}`:
		return findObserveGroups(input, `\frac{`, bStr(""), "", eStr("}"),
			&fogPart2{"{", bStr(""), "", eStr("}")}, false)
	case `\overset{(...)}`:
		return findObserveGroups(input, `\overset{`, bStr(""), "", eStr("}"),
			&fogPart2{"{", bStr(""), "", eStr("}")}, false)
	case `\underset{(...)}`:
		return findObserveGroups(input, `\underset{`, bStr(""), "", eStr("}"),
			&fogPart2{"{", bStr(""), "", eStr("}")}, false)
	case `\underbrace{(...)}`:
		return findObserveGroups(input, `\underbrace{`, bStr(""), "", eStr("}_"),
			&fogPart2{"{", bStr(""), "", eStr("}")}, false)
	case `\color{(...)}0`:
		return findObserveGroups(input, `\color{`, bStr(""), "", eStr("}"), nil, false)
	case `\color{(...)}{(...)}1`:
		return findObserveGroups(input, `\color{`, bStr(""), "", eStr("}"),
			&fogPart2{"{", bStr(""), "", eStr("}")}, false)
	case `\color(...){(...)}2`:
		return findObserveGroups(input, `\color`, bStr(`\`), "", eRe(reBeforeBrace),
			&fogPart2{"{", bStr(""), "", eStr("}")}, false)
	case `\ce{(...)}`:
		return findObserveGroups(input, `\ce{`, bStr(""), "", eStr("}"), nil, false)
	}
	re, ok := data.Regexes[name]
	if !ok {
		return nil, errMsg("mhchem bug P: unknown pattern (%s)", name)
	}
	tok, end, ok := regexMatchToken(re, input)
	if !ok {
		return nil, nil
	}
	return &PatternHit{Token: tok, Remainder: input[end:]}, nil
}
