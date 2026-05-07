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

	"github.com/luo-studio/go-tex/tex/lexer"
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
		m[k] = Definition{Text: v, NumArg: maxArgRef(v)}
	}
	// Function-based macros.
	m[`\TextOrMath`] = Definition{Fn: textOrMath}
	m[`\def`] = Definition{Fn: handleDef(false)}
	m[`\gdef`] = Definition{Fn: handleDef(false)}
	m[`\edef`] = Definition{Fn: handleDef(false)}
	m[`\xdef`] = Definition{Fn: handleDef(false)}
	m[`\@firstoftwo`] = Definition{Fn: firstOfTwo}
	m[`\@secondoftwo`] = Definition{Fn: secondOfTwo}
	return m
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
func handleDef(global bool) FnMacro {
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
		e.macros[name] = Definition{Tokens: bodyToks, NumArg: numArgs}
		_ = global
		return nil, nil
	}
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
