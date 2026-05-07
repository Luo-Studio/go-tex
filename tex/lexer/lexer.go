package lexer

import "unicode/utf8"

// Catcode is a TeX category code. Only a small subset is observed by the
// lexer: comment (14) and active (13).
type Catcode uint8

const (
	// CatcodeOther is the implicit code for any character without a custom
	// catcode set; the lexer treats it as a regular character.
	CatcodeOther Catcode = 0
	// CatcodeActive marks the character as TeX-active (e.g. `~`).
	CatcodeActive Catcode = 13
	// CatcodeComment marks the comment character (e.g. `%`).
	CatcodeComment Catcode = 14
)

// Lexer tokenises a LaTeX math input string.
//
// Lexing strategy (matches KaTeX/RaTeX):
//
//   - Control words: `\` followed by one or more ASCII letters or `@`.
//     Trailing whitespace is consumed.
//   - Control symbols: `\` followed by a single non-letter character.
//   - Control space: `\` followed by whitespace; produces "\\ ".
//   - `\verb` / `\verb*` consume a delimiter and read verbatim until the
//     same delimiter recurs (or EOF).
//   - Comments: comment-catcode character through end of line, skipped.
//   - Whitespace runs: collapsed to a single " " token.
//   - Regular characters: the character plus any trailing combining
//     diacritical marks (U+0300-U+036F) form one token.
//
// A Lexer is single-use: callers should construct a new one per input.
type Lexer struct {
	input    string
	pos      int
	catcodes []catcodeEntry
}

type catcodeEntry struct {
	ch   rune
	code Catcode
}

// New returns a Lexer over input with the default TeX catcodes
// (`%` → comment, `~` → active).
func New(input string) *Lexer {
	return &Lexer{
		input: input,
		catcodes: []catcodeEntry{
			{'%', CatcodeComment},
			{'~', CatcodeActive},
		},
	}
}

// Pos returns the current byte position into the input.
func (l *Lexer) Pos() int { return l.pos }

// SetCatcode sets the category code for ch. Used by the macro expander for
// commands like `\makeatletter`.
func (l *Lexer) SetCatcode(ch rune, code Catcode) {
	for i := range l.catcodes {
		if l.catcodes[i].ch == ch {
			l.catcodes[i].code = code
			return
		}
	}
	l.catcodes = append(l.catcodes, catcodeEntry{ch, code})
}

// Catcode returns the category code for ch (CatcodeOther if none is set).
func (l *Lexer) Catcode(ch rune) Catcode {
	for _, e := range l.catcodes {
		if e.ch == ch {
			return e.code
		}
	}
	return CatcodeOther
}

func (l *Lexer) atEnd() bool { return l.pos >= len(l.input) }

func (l *Lexer) peekByte() (byte, bool) {
	if l.atEnd() {
		return 0, false
	}
	return l.input[l.pos], true
}

func (l *Lexer) currentRune() (rune, int, bool) {
	if l.atEnd() {
		return 0, 0, false
	}
	r, sz := utf8.DecodeRuneInString(l.input[l.pos:])
	return r, sz, true
}

func (l *Lexer) advanceRune() (rune, bool) {
	r, sz, ok := l.currentRune()
	if !ok {
		return 0, false
	}
	l.pos += sz
	return r, true
}

func isWhitespaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// KaTeX includes `@` in the control-word character class.
func isControlWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '@'
}

func isCombiningDiacritical(r rune) bool {
	return r >= 0x0300 && r <= 0x036F
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && isWhitespaceByte(l.input[l.pos]) {
		l.pos++
	}
}

// lexVerb consumes the body of a `\verb` or `\verb*` token. The leading
// `\verb` (and `*`, if any) is treated as part of the resulting token text,
// matching KaTeX's behaviour where the entire verbatim run is one token.
func (l *Lexer) lexVerb(start int) Token {
	if l.atEnd() {
		return NewToken("\\verb", start, l.pos)
	}
	if b, ok := l.peekByte(); ok && b == '*' {
		l.pos++
	}
	if l.atEnd() {
		return NewToken(l.input[start:l.pos], start, l.pos)
	}
	delim, _ := l.advanceRune()
	for {
		r, ok := l.advanceRune()
		if !ok {
			// Unterminated \verb — return what we have.
			return NewToken(l.input[start:l.pos], start, l.pos)
		}
		if r == delim {
			return NewToken(l.input[start:l.pos], start, l.pos)
		}
	}
}

// Lex returns the next token. At end of input it returns an EOF token; it is
// safe to keep calling Lex after that — every subsequent call also returns
// EOF.
func (l *Lexer) Lex() Token {
	if l.atEnd() {
		return EOF(l.pos)
	}

	start := l.pos

	// Whitespace → single space token.
	if b, _ := l.peekByte(); isWhitespaceByte(b) {
		l.skipWhitespace()
		return NewToken(" ", start, l.pos)
	}

	ch, _, _ := l.currentRune()

	// Comment: skip to end of line and try again.
	if l.Catcode(ch) == CatcodeComment {
		l.advanceRune()
		for !l.atEnd() {
			if l.input[l.pos] == '\n' {
				l.pos++
				break
			}
			l.pos++
		}
		return l.Lex()
	}

	// Backslash → control sequence.
	if ch == '\\' {
		l.pos++ // consume the backslash
		if l.atEnd() {
			return NewToken("\\", start, l.pos)
		}

		next := l.input[l.pos]

		if isWhitespaceByte(next) {
			// Control space: \<whitespace> → "\\ ", trailing whitespace consumed.
			l.pos++
			l.skipWhitespace()
			return NewToken("\\ ", start, l.pos)
		}

		if isControlWordByte(next) {
			for l.pos < len(l.input) && isControlWordByte(l.input[l.pos]) {
				l.pos++
			}
			text := l.input[start:l.pos]
			if text == "\\verb" {
				return l.lexVerb(start)
			}
			end := l.pos
			// Trailing whitespace after a control word is consumed (TeX rule).
			l.skipWhitespace()
			return NewToken(text, start, end)
		}

		// Control symbol: \<single non-letter rune>
		_, sz, _ := l.currentRune()
		l.pos += sz
		return NewToken(l.input[start:l.pos], start, l.pos)
	}

	// Active character (catcode 13).
	if l.Catcode(ch) == CatcodeActive {
		l.advanceRune()
		return NewToken(l.input[start:l.pos], start, l.pos)
	}

	// Regular character + trailing combining marks.
	l.advanceRune()
	for {
		r, _, ok := l.currentRune()
		if !ok || !isCombiningDiacritical(r) {
			break
		}
		l.advanceRune()
	}
	return NewToken(l.input[start:l.pos], start, l.pos)
}

// All lexes the entire input and returns every token, including the final EOF.
func (l *Lexer) All() []Token {
	var toks []Token
	for {
		t := l.Lex()
		toks = append(toks, t)
		if t.IsEOF() {
			return toks
		}
	}
}
