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
		if def.NumArg > 0 {
			// Arg-based macros not yet supported — leave as-is and let the
			// parser raise "Undefined control sequence".
			return t, nil
		}
		switch {
		case def.Fn != nil:
			toks, err := def.Fn(e)
			if err != nil {
				return t, err
			}
			e.stack = append(e.stack, toks...)
		case def.Tokens != nil:
			e.stack = append(e.stack, def.Tokens...)
		default:
			toks := lexAll(def.Text)
			e.stack = append(e.stack, toks...)
		}
	}
}

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
		// Skip arg-based macros (those that use #1, #2, ...) — they need
		// proper consume_args support which we haven't ported yet.
		if hasMacroArg(v) {
			m[k] = Definition{Text: v, NumArg: 1} // marked, not expanded
			continue
		}
		m[k] = Definition{Text: v}
	}
	// Function-based macros.
	m[`\TextOrMath`] = Definition{Fn: textOrMath}
	return m
}

func hasMacroArg(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '#' {
			c := s[i+1]
			if c >= '0' && c <= '9' {
				return true
			}
		}
	}
	return false
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
