// Package macroexp is a minimal port of the upstream RaTeX macro expander
// (the "gullet"). It sits between the lexer and parser, expanding text
// macros and a small set of function macros (currently just \TextOrMath).
//
// The full upstream expander supports \def, \newcommand, \char, \@ifstar,
// catcode changes, and grouping. Those are not yet ported; the goal of
// this minimal version is to unlock the score gains from the static
// built-in macros (Greek aliases, spacing kerns via \TextOrMath, …).
package macroexp

import (
	"fmt"
	"strings"

	"github.com/luo-studio/go-tex/tex/lexer"
	"github.com/luo-studio/go-tex/tex/mhchem"
)

// Mode mirrors parser.Mode without an import cycle.
type Mode uint8

const (
	ModeMath Mode = iota
	ModeText
)

// FnMacro is a function-based macro. It receives the expander and returns
// the tokens to push onto the stack (in stack order — last token in the
// returned slice is consumed first).
type FnMacro func(*Expander) ([]lexer.Token, error)

// Definition is one of: a simple text expansion, a pre-tokenised list, or
// a function. Text expansions are lexed lazily on first use.
type Definition struct {
	Text   string
	Tokens []lexer.Token // already in stack order (reversed)
	Fn     FnMacro
	NumArg int // for arg-based macros (not yet honoured)
}

// Expander wraps a Lexer and exposes a stream of tokens with macros expanded.
type Expander struct {
	lex     *lexer.Lexer
	mode    Mode
	stack   []lexer.Token
	macros  map[string]Definition
	expand  int // recursion guard
	maxExp  int
}

// New returns an Expander over input in the given starting mode.
func New(input string, mode Mode) *Expander {
	e := &Expander{
		lex:    lexer.New(input),
		mode:   mode,
		macros: defaultMacros(),
		maxExp: 10000,
	}
	return e
}

// Mode returns the current parser mode (used by \TextOrMath).
func (e *Expander) Mode() Mode { return e.mode }

// SetMode lets the parser update the expander's mode when entering \text{}.
func (e *Expander) SetMode(m Mode) { e.mode = m }

// Pos returns the underlying lexer position. Useful for error locations.
func (e *Expander) Pos() int { return e.lex.Pos() }

// pop returns the next raw token: from the stack first, otherwise from the
// underlying lexer.
func (e *Expander) pop() lexer.Token {
	if n := len(e.stack); n > 0 {
		t := e.stack[n-1]
		e.stack = e.stack[:n-1]
		return t
	}
	return e.lex.Lex()
}

// Push pushes a token back onto the stack so the next pop returns it.
func (e *Expander) Push(t lexer.Token) {
	e.stack = append(e.stack, t)
}

// Future returns the next token without consuming it.
func (e *Expander) Future() lexer.Token {
	t := e.pop()
	e.Push(t)
	return t
}

// Next returns the next non-expandable token.
func (e *Expander) Next() (lexer.Token, error) {
	for {
		t := e.pop()
		if t.IsEOF() {
			return t, nil
		}
		if t.NoExpand {
			return t, nil
		}
		def, ok := e.macros[t.Text]
		if !ok {
			return t, nil
		}
		if e.expand++; e.expand > e.maxExp {
			return t, fmt.Errorf("macro expansion limit reached at %q", t.Text)
		}
		switch {
		case def.Fn != nil:
			toks, err := def.Fn(e)
			if err != nil {
				return t, err
			}
			e.stack = append(e.stack, toks...)
		case def.Tokens != nil:
			toks, err := expandTokens(e, def.Tokens, def.NumArg)
			if err != nil {
				return t, err
			}
			e.stack = append(e.stack, toks...)
		case def.NumArg > 0:
			args, err := e.consumeArgs(def.NumArg)
			if err != nil {
				return t, err
			}
			expansion, err := substituteArgs(def.Text, args)
			if err != nil {
				return t, err
			}
			e.stack = append(e.stack, lexAll(expansion)...)
		default:
			toks := lexAll(def.Text)
			e.stack = append(e.stack, toks...)
		}
	}
}

// expandTokens reads numArgs brace-args (as token slices) and produces the
// macro expansion in stack order: body tokens with #1, #2, … replaced by
// the corresponding arg's tokens. Body tokens preserve their original
// source positions; arg tokens carry the positions from the invocation.
func expandTokens(e *Expander, body []lexer.Token, numArgs int) ([]lexer.Token, error) {
	args := make([][]lexer.Token, numArgs)
	for i := 0; i < numArgs; i++ {
		toks, err := e.consumeArgTokens()
		if err != nil {
			return nil, fmt.Errorf("reading arg %d: %w", i+1, err)
		}
		args[i] = toks
	}
	out := make([]lexer.Token, 0, len(body))
	// Walk body, expanding #N. ## escapes a literal #.
	for i := 0; i < len(body); i++ {
		t := body[i]
		if t.Text == "#" && i+1 < len(body) {
			nx := body[i+1].Text
			if nx == "#" {
				out = append(out, body[i+1])
				i++
				continue
			}
			if len(nx) == 1 && nx[0] >= '1' && nx[0] <= '9' {
				idx := int(nx[0] - '1')
				if idx < len(args) {
					out = append(out, args[idx]...)
				}
				i++
				continue
			}
		}
		out = append(out, t)
	}
	// Reverse for stack order.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// consumeArgTokens reads a single brace-or-single-token argument and
// returns its raw token slice (in source order).
func (e *Expander) consumeArgTokens() ([]lexer.Token, error) {
	for {
		t := e.pop()
		if t.IsEOF() {
			return nil, fmt.Errorf("unexpected EOF in macro argument")
		}
		if t.IsSpace() {
			continue
		}
		if t.Text != "{" {
			return []lexer.Token{t}, nil
		}
		var out []lexer.Token
		depth := 1
		for {
			tt := e.pop()
			if tt.IsEOF() {
				return nil, fmt.Errorf("unterminated macro argument")
			}
			if tt.Text == "{" {
				depth++
			} else if tt.Text == "}" {
				depth--
				if depth == 0 {
					return out, nil
				}
			}
			out = append(out, tt)
		}
	}
}

// consumeArgs reads n brace-or-single-token arguments and returns each
// arg's raw source string (so #1/#2 substitution can be done as text
// replacement before re-lexing).
func (e *Expander) consumeArgs(n int) ([]string, error) {
	args := make([]string, n)
	for i := 0; i < n; i++ {
		// Skip leading spaces.
		for {
			t := e.pop()
			if t.IsEOF() {
				return nil, fmt.Errorf("unexpected EOF reading macro arg %d", i+1)
			}
			if t.IsSpace() {
				continue
			}
			if t.Text != "{" {
				args[i] = t.Text
				break
			}
			// Brace group: collect raw tokens until matching `}`.
			var b []byte
			depth := 1
			for {
				t := e.pop()
				if t.IsEOF() {
					return nil, fmt.Errorf("unterminated macro arg %d", i+1)
				}
				if t.Text == "{" {
					depth++
				} else if t.Text == "}" {
					depth--
					if depth == 0 {
						break
					}
				}
				b = append(b, t.Text...)
			}
			args[i] = string(b)
			break
		}
	}
	return args, nil
}

// substituteArgs replaces #1, #2, ... in template with the corresponding
// element of args. ## escapes a literal #.
func substituteArgs(template string, args []string) (string, error) {
	var out []byte
	i := 0
	for i < len(template) {
		c := template[i]
		if c != '#' {
			out = append(out, c)
			i++
			continue
		}
		if i+1 >= len(template) {
			return "", fmt.Errorf("trailing # in macro template")
		}
		nx := template[i+1]
		if nx == '#' {
			out = append(out, '#')
			i += 2
			continue
		}
		if nx >= '1' && nx <= '9' {
			idx := int(nx - '0' - 1)
			if idx < len(args) {
				out = append(out, args[idx]...)
			}
			i += 2
			continue
		}
		out = append(out, c)
		i++
	}
	return string(out), nil
}

// BeginGroup / EndGroup are no-ops in the minimal expander but kept here
// for API compatibility — \def/\gdef tracking can use them later.
func (e *Expander) BeginGroup() {}
func (e *Expander) EndGroup()   {}

// SetMacro registers (or overwrites) a macro definition. Used by tests.
func (e *Expander) SetMacro(name string, def Definition) {
	e.macros[name] = def
}

// HasMacro reports whether name is a known macro.
func (e *Expander) HasMacro(name string) bool {
	_, ok := e.macros[name]
	return ok
}

// MacroBodyText returns the textual body of a macro definition. For text-
// based macros it returns Definition.Text; for token-based macros it
// concatenates token text. Returns ("", false) if the macro is missing,
// function-based, or has args.
func (e *Expander) MacroBodyText(name string) (string, bool) {
	def, ok := e.macros[name]
	if !ok || def.Fn != nil || def.NumArg > 0 {
		return "", false
	}
	if def.Text != "" {
		return def.Text, true
	}
	var b strings.Builder
	for _, t := range def.Tokens {
		b.WriteString(t.Text)
	}
	return b.String(), true
}

// lexBodySource lexes a macro body in source order (NOT stack order). Used
// at registration time when the body's tokens are spliced via #N substitution
// — the per-call `expandTokens` reverses to stack order at the end.
func lexBodySource(text string) []lexer.Token {
	l := lexer.New(text)
	var toks []lexer.Token
	for {
		t := l.Lex()
		if t.IsEOF() {
			break
		}
		toks = append(toks, t)
	}
	return toks
}

// handleMhchem implements \ce / \pu. The single brace argument is parsed
// through the mhchem state machine; the resulting TeX is re-lexed and
// pushed back onto the macro stack.
//
// Unlike consumeArgs (which concatenates token texts and so loses any
// whitespace between control sequences), this rebuilds the source string
// from each token's Loc, inserting a space wherever the next token starts
// past where the previous one ended. Mirrors upstream
// `mhchem_arg_tokens_to_string`.
func handleMhchem(mode string) FnMacro {
	return func(e *Expander) ([]lexer.Token, error) {
		args, err := e.consumeArgsAsTokens(1)
		if err != nil {
			return nil, err
		}
		src := mhchemSourceFromTokens(args[0])
		tex, err := mhchem.ChemParseStr(src, mode)
		if err != nil {
			return nil, fmt.Errorf("\\%s: %w", mode, err)
		}
		return lexAll(tex), nil
	}
}

// mhchemSourceFromTokens reconstructs the original source string of a
// brace argument from its token list, inserting a space wherever the
// source has a gap (e.g. the trailing space after a control sequence).
func mhchemSourceFromTokens(toks []lexer.Token) string {
	if len(toks) == 0 {
		return ""
	}
	var b strings.Builder
	expected := toks[0].Loc.Start
	for _, t := range toks {
		if t.Loc.Start > expected {
			b.WriteByte(' ')
		}
		b.WriteString(t.Text)
		expected = t.Loc.End
	}
	return b.String()
}

// lexAll lexes text and returns its tokens in stack order (i.e. reversed).
func lexAll(text string) []lexer.Token {
	l := lexer.New(text)
	var toks []lexer.Token
	for {
		t := l.Lex()
		if t.IsEOF() {
			break
		}
		toks = append(toks, t)
	}
	// Reverse for stack order.
	for i, j := 0, len(toks)-1; i < j; i, j = i+1, j-1 {
		toks[i], toks[j] = toks[j], toks[i]
	}
	return toks
}

func defaultMacros() map[string]Definition {
	m := make(map[string]Definition, len(builtinTextMacros)+8)
	for k, v := range builtinTextMacros {
		n := maxArgRef(v)
		if n > 0 {
			// Pre-lex arg-using builtins so #N substitution happens at the
			// token level rather than via text join (which would lose
			// whitespace between `\rm` and `mod`, etc.).
			m[k] = Definition{Tokens: lexBodySource(v), NumArg: n}
			continue
		}
		m[k] = Definition{Text: v}
	}
	// Function-based macros.
	m[`\TextOrMath`] = Definition{Fn: textOrMath}
	m[`\def`] = Definition{Fn: handleDef(false, false)}
	m[`\gdef`] = Definition{Fn: handleDef(false, false)}
	m[`\edef`] = Definition{Fn: handleDef(false, true)}
	m[`\xdef`] = Definition{Fn: handleDef(false, true)}
	m[`\newcommand`] = Definition{Fn: handleNewCommand(true, false)}
	m[`\renewcommand`] = Definition{Fn: handleNewCommand(false, true)}
	m[`\providecommand`] = Definition{Fn: handleNewCommand(true, true)}
	m[`\@firstoftwo`] = Definition{Fn: firstOfTwo}
	m[`\@secondoftwo`] = Definition{Fn: secondOfTwo}
	m[`\@ifstar`] = Definition{Fn: ifStar}
	m[`\@ifnextchar`] = Definition{Fn: ifNextChar}
	m[`\noexpand`] = Definition{Fn: noExpand}
	m[`\message`] = Definition{Fn: discardOneArg}
	m[`\errmessage`] = Definition{Fn: discardOneArg}
	// HTML extensions: no-op, return the content arg (KaTeX renders content
	// only; no HTML attributes since we are output-format-agnostic here).
	htmlPass := func(e *Expander) ([]lexer.Token, error) {
		args, err := e.consumeArgsAsTokens(2)
		if err != nil {
			return nil, err
		}
		// args[1] is in source order; reverse for stack order.
		toks := make([]lexer.Token, len(args[1]))
		for i, t := range args[1] {
			toks[len(args[1])-1-i] = t
		}
		return toks, nil
	}
	m[`\htmlClass`] = Definition{Fn: htmlPass}
	m[`\htmlData`] = Definition{Fn: htmlPass}
	m[`\htmlId`] = Definition{Fn: htmlPass}
	m[`\htmlStyle`] = Definition{Fn: htmlPass}
	m[`\char`] = Definition{Fn: handleChar}
	m[`\ce`] = Definition{Fn: handleMhchem("ce")}
	m[`\pu`] = Definition{Fn: handleMhchem("pu")}
	// Text-form alias: \operatorname → \@ifstar\operatornamewithlimits\operatorname@
	m[`\operatorname`] = Definition{Text: `\@ifstar\operatornamewithlimits\operatorname@`}
	// \char: parse a number in decimal/octal/hex/backtick form and emit
	// tokens "\@char{N}". The parser doesn't yet implement \@char; for now
	// we approximate by passing through (most golden cases don't depend
	// on \char output beyond what's already covered by the symbols table).
	return m
}

// consumeArgsAsTokens reads n brace-or-single-token arguments and returns
// each arg's token slice in source order.
func (e *Expander) consumeArgsAsTokens(n int) ([][]lexer.Token, error) {
	args := make([][]lexer.Token, n)
	for i := 0; i < n; i++ {
		toks, err := e.consumeArgTokens()
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i+1, err)
		}
		args[i] = toks
	}
	return args, nil
}

func ifStar(e *Expander) ([]lexer.Token, error) {
	args, err := e.consumeArgsAsTokens(2)
	if err != nil {
		return nil, err
	}
	chosen := args[1] // default: the without-star branch
	// Skip leading spaces in the lookahead for `*`.
	for {
		t := e.pop()
		if t.IsEOF() {
			break
		}
		if t.IsSpace() {
			continue
		}
		if t.Text == "*" {
			chosen = args[0]
		} else {
			e.Push(t)
		}
		break
	}
	out := make([]lexer.Token, len(chosen))
	for i, t := range chosen {
		out[len(chosen)-1-i] = t
	}
	return out, nil
}

func ifNextChar(e *Expander) ([]lexer.Token, error) {
	args, err := e.consumeArgsAsTokens(3)
	if err != nil {
		return nil, err
	}
	// Skip spaces in the lookahead.
	for {
		t := e.pop()
		if t.IsEOF() {
			break
		}
		if t.IsSpace() {
			continue
		}
		e.Push(t)
		break
	}
	want := ""
	if len(args[0]) > 0 {
		// args[0] is in source order; the "first" char in original order is
		// the last element of the stack-reversed form, which equals
		// args[0][0] in source order.
		want = args[0][0].Text
	}
	chosen := args[2]
	if e.Future().Text == want {
		chosen = args[1]
	}
	out := make([]lexer.Token, len(chosen))
	for i, t := range chosen {
		out[len(chosen)-1-i] = t
	}
	return out, nil
}

func noExpand(e *Expander) ([]lexer.Token, error) {
	t := e.pop()
	if e.HasMacro(t.Text) {
		t.NoExpand = true
		t.TreatAsRelax = true
	}
	return []lexer.Token{t}, nil
}

// handleChar implements `\char` — parses a number in decimal/octal/hex/
// backtick form and emits "\@char{N}" tokens for the parser to consume.
//
//	\char"3FE   -> hex
//	\char'277   -> octal
//	\char`A     -> ASCII codepoint of next char
//	\char65     -> decimal
func handleChar(e *Expander) ([]lexer.Token, error) {
	t := e.pop()
	for t.IsSpace() {
		t = e.pop()
	}
	if t.IsEOF() {
		return nil, fmt.Errorf(`\char: unexpected EOF`)
	}
	loc := t.Loc
	var number int64
	switch t.Text {
	case "'":
		t = e.pop()
		number = parseInBase(t.Text, 8)
		readMore(e, &number, 8)
	case `"`:
		t = e.pop()
		number = parseInBase(t.Text, 16)
		readMore(e, &number, 16)
	case "`":
		t = e.pop()
		s := t.Text
		var ch rune
		if len(s) > 0 && s[0] == '\\' && len(s) > 1 {
			for _, r := range s[1:] {
				ch = r
				break
			}
		} else if len(s) > 0 {
			for _, r := range s {
				ch = r
				break
			}
		}
		number = int64(ch)
	default:
		number = parseInBase(t.Text, 10)
		readMore(e, &number, 10)
	}
	s := fmt.Sprintf("%d", number)
	out := []lexer.Token{
		{Text: "}", Loc: loc},
	}
	for i := len(s) - 1; i >= 0; i-- {
		out = append(out, lexer.Token{Text: string(s[i]), Loc: loc})
	}
	out = append(out, lexer.Token{Text: "{", Loc: loc})
	out = append(out, lexer.Token{Text: `\@char`, Loc: loc})
	return out, nil
}

func parseInBase(s string, base int) int64 {
	var n int64
	for _, c := range s {
		var d int
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		default:
			return n
		}
		if d >= base {
			return n
		}
		n = n*int64(base) + int64(d)
	}
	return n
}

func readMore(e *Expander, n *int64, base int) {
	for {
		t := e.Future()
		// Only single-char digit tokens contribute.
		if len(t.Text) != 1 {
			return
		}
		c := t.Text[0]
		if !isDigitForBase(c, base) {
			return
		}
		more := parseInBase(t.Text, base)
		e.pop()
		*n = (*n)*int64(base) + more
	}
}

func isDigitForBase(c byte, base int) bool {
	switch {
	case c >= '0' && c <= '9':
		return int(c-'0') < base
	case c >= 'a' && c <= 'f':
		return base > 10 && int(c-'a')+10 < base
	case c >= 'A' && c <= 'F':
		return base > 10 && int(c-'A')+10 < base
	}
	return false
}

func discardOneArg(e *Expander) ([]lexer.Token, error) {
	if _, err := e.consumeArgsAsTokens(1); err != nil {
		return nil, err
	}
	return nil, nil
}

// handleNewCommand returns a function macro for \newcommand / \renewcommand
// / \providecommand. Reads `\name`, optional `[N]` (arg count), optional
// `[default]` (for first-arg default — not yet honoured), and a body.
func handleNewCommand(allowNew, allowExisting bool) FnMacro {
	return func(e *Expander) ([]lexer.Token, error) {
		// Skip spaces.
		var nameTok lexer.Token
		for {
			t := e.pop()
			if t.IsEOF() {
				return nil, fmt.Errorf("\\newcommand: unexpected EOF")
			}
			if t.IsSpace() {
				continue
			}
			nameTok = t
			break
		}
		name := nameTok.Text
		if name == "{" {
			// Brace-wrapped name.
			t := e.pop()
			name = t.Text
			closing := e.pop()
			if closing.Text != "}" {
				return nil, fmt.Errorf("\\newcommand: expected } after name")
			}
		}
		if name == "" || name[0] != '\\' {
			return nil, fmt.Errorf("\\newcommand: expected control sequence")
		}
		if e.HasMacro(name) && !allowExisting {
			return nil, fmt.Errorf("\\newcommand: %s is already defined", name)
		}
		if !e.HasMacro(name) && !allowNew {
			return nil, fmt.Errorf("\\newcommand: %s is not yet defined", name)
		}
		// Optional [num]
		numArgs := 0
		for {
			t := e.pop()
			if t.IsSpace() {
				continue
			}
			if t.Text == "[" {
				var raw []byte
				for {
					tt := e.pop()
					if tt.Text == "]" {
						break
					}
					if tt.IsEOF() {
						return nil, fmt.Errorf("\\newcommand: unterminated [num]")
					}
					raw = append(raw, tt.Text...)
				}
				s := strings.TrimSpace(string(raw))
				if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
					numArgs = int(s[0] - '0')
				}
				continue
			}
			// Optional default for first arg `[default]` is also possible;
			// skip it for the minimal implementation.
			e.Push(t)
			break
		}
		// Read the body group.
		t := e.pop()
		if t.Text != "{" {
			return nil, fmt.Errorf("\\newcommand: expected body group")
		}
		var body []lexer.Token
		depth := 1
		for {
			tt := e.pop()
			if tt.IsEOF() {
				return nil, fmt.Errorf("\\newcommand: unterminated body")
			}
			if tt.Text == "{" {
				depth++
			} else if tt.Text == "}" {
				depth--
				if depth == 0 {
					break
				}
			}
			body = append(body, tt)
		}
		if e.HasMacro(name) && allowNew && allowExisting {
			// providecommand: keep existing definition, discard new one.
			return nil, nil
		}
		e.macros[name] = Definition{Tokens: body, NumArg: numArgs}
		return nil, nil
	}
}

// maxArgRef returns the largest #N appearing in template, or 0 if none.
// `##` escapes are ignored.
func maxArgRef(s string) int {
	max := 0
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '#' {
			continue
		}
		nx := s[i+1]
		if nx == '#' {
			i++
			continue
		}
		if nx >= '1' && nx <= '9' {
			n := int(nx - '0')
			if n > max {
				max = n
			}
		}
	}
	return max
}

func firstOfTwo(e *Expander) ([]lexer.Token, error) {
	args, err := e.consumeArgs(2)
	if err != nil {
		return nil, err
	}
	return lexAll(args[0]), nil
}

func secondOfTwo(e *Expander) ([]lexer.Token, error) {
	args, err := e.consumeArgs(2)
	if err != nil {
		return nil, err
	}
	return lexAll(args[1]), nil
}

// handleDef returns the function macro that implements \def-style
// commands. Reads the next control-sequence name, optional argument
// template (up to the opening brace), and body, then registers the macro
// in the expander's namespace.
//
// The body is stored as a token slice (preserving original source
// positions), so the resulting AST nodes carry locs from the original
// input — matching the upstream behaviour.
func handleDef(global, expandBody bool) FnMacro {
	return func(e *Expander) ([]lexer.Token, error) {
		var name string
		for {
			t := e.pop()
			if t.IsEOF() {
				return nil, fmt.Errorf("\\def: unexpected EOF reading name")
			}
			if t.IsSpace() {
				continue
			}
			name = t.Text
			break
		}
		if name == "" || name[0] != '\\' {
			return nil, fmt.Errorf("\\def: expected control sequence, got %q", name)
		}
		// Skim the template up to `{`, counting #N references.
		numArgs := 0
		for {
			t := e.pop()
			if t.IsEOF() {
				return nil, fmt.Errorf("\\def: unexpected EOF reading template")
			}
			if t.Text == "{" {
				e.Push(t)
				break
			}
			if t.Text == "#" {
				nt := e.pop()
				if nt.IsEOF() {
					return nil, fmt.Errorf("\\def: unexpected EOF after #")
				}
				if len(nt.Text) == 1 && nt.Text[0] >= '1' && nt.Text[0] <= '9' {
					n := int(nt.Text[0] - '0')
					if n > numArgs {
						numArgs = n
					}
				}
				continue
			}
		}
		// Read body tokens preserving order (and their original locs).
		t := e.pop()
		if t.Text != "{" {
			return nil, fmt.Errorf("\\def: expected body group, got %q", t.Text)
		}
		var bodyToks []lexer.Token
		depth := 1
		for {
			tt := e.pop()
			if tt.IsEOF() {
				return nil, fmt.Errorf("\\def: unterminated body")
			}
			if tt.Text == "{" {
				depth++
			} else if tt.Text == "}" {
				depth--
				if depth == 0 {
					break
				}
			}
			bodyToks = append(bodyToks, tt)
		}
		// \edef / \xdef: fully expand body tokens at definition time.
		if expandBody {
			expanded, err := e.expandTokensFully(bodyToks)
			if err != nil {
				return nil, err
			}
			bodyToks = expanded
		}
		e.macros[name] = Definition{Tokens: bodyToks, NumArg: numArgs}
		_ = global
		return nil, nil
	}
}

// expandTokensFully runs the gullet over the given tokens, fully
// expanding any macro/function calls, and returns the resulting flat
// token list. Mirrors upstream MacroExpander::expand_tokens.
func (e *Expander) expandTokensFully(tokens []lexer.Token) ([]lexer.Token, error) {
	// Save the existing stack and replace with the new tokens (in stack
	// order — top is last). Our `tokens` are in document order, so push
	// them in reverse.
	saved := e.stack
	e.stack = make([]lexer.Token, 0, len(tokens))
	for i := len(tokens) - 1; i >= 0; i-- {
		e.stack = append(e.stack, tokens[i])
	}
	var result []lexer.Token
	for len(e.stack) > 0 {
		// Peek the top token without lexing more. If it's an
		// expandable macro, expand once and loop. Otherwise pop it as
		// a literal and append to result.
		top := e.stack[len(e.stack)-1]
		if top.NoExpand {
			e.stack = e.stack[:len(e.stack)-1]
			result = append(result, top)
			continue
		}
		def, ok := e.macros[top.Text]
		if !ok {
			e.stack = e.stack[:len(e.stack)-1]
			result = append(result, top)
			continue
		}
		// Expandable macro: pop and expand.
		e.stack = e.stack[:len(e.stack)-1]
		if e.expand++; e.expand > e.maxExp {
			e.stack = saved
			return nil, fmt.Errorf("macro expansion limit reached at %q", top.Text)
		}
		switch {
		case def.Fn != nil:
			toks, err := def.Fn(e)
			if err != nil {
				e.stack = saved
				return nil, err
			}
			e.stack = append(e.stack, toks...)
		case def.Tokens != nil:
			toks, err := expandTokens(e, def.Tokens, def.NumArg)
			if err != nil {
				e.stack = saved
				return nil, err
			}
			e.stack = append(e.stack, toks...)
		case def.NumArg > 0:
			args, err := e.consumeArgs(def.NumArg)
			if err != nil {
				e.stack = saved
				return nil, err
			}
			expansion, err := substituteArgs(def.Text, args)
			if err != nil {
				e.stack = saved
				return nil, err
			}
			e.stack = append(e.stack, lexAll(expansion)...)
		default:
			toks := lexAll(def.Text)
			e.stack = append(e.stack, toks...)
		}
	}
	e.stack = saved
	return result, nil
}

// textOrMath: \TextOrMath{textBranch}{mathBranch}.
//
// Reads two brace groups (or single tokens) and emits the appropriate one,
// in stack order, for the current mode.
func textOrMath(e *Expander) ([]lexer.Token, error) {
	textArg, err := e.consumeBraceArg()
	if err != nil {
		return nil, err
	}
	mathArg, err := e.consumeBraceArg()
	if err != nil {
		return nil, err
	}
	if e.mode == ModeText {
		return reverseToks(textArg), nil
	}
	return reverseToks(mathArg), nil
}

// consumeBraceArg reads either a `{...}` group or a single token. Returns
// the tokens *in source order* (caller reverses if needed).
func (e *Expander) consumeBraceArg() ([]lexer.Token, error) {
	// Skip leading spaces.
	for {
		t := e.pop()
		if t.IsEOF() {
			return nil, fmt.Errorf("unexpected EOF in macro argument")
		}
		if t.IsSpace() {
			continue
		}
		if t.Text != "{" {
			return []lexer.Token{t}, nil
		}
		break
	}
	var out []lexer.Token
	depth := 1
	for {
		t := e.pop()
		if t.IsEOF() {
			return nil, fmt.Errorf("unterminated macro argument")
		}
		if t.Text == "{" {
			depth++
		}
		if t.Text == "}" {
			depth--
			if depth == 0 {
				return out, nil
			}
		}
		out = append(out, t)
	}
}

func reverseToks(in []lexer.Token) []lexer.Token {
	out := make([]lexer.Token, len(in))
	for i, t := range in {
		out[len(in)-1-i] = t
	}
	return out
}
