// Package lexer tokenises a LaTeX math input string into a stream of Tokens
// matching KaTeX's lexer behaviour. The parser and macro expander interpret
// meaning; the lexer's only job is to identify token boundaries.
package lexer

import "github.com/luo-studio/go-tex/tex/source"

// EOFText is the synthetic text used for the end-of-input token.
const EOFText = "EOF"

// Token is a single lexed token.
//
// Token identity is determined by [Token.Text]: the parser distinguishes
// control words ("\\frac", "\\alpha"), control symbols ("\\,"), control
// space ("\\ "), single characters ("a", "+", "{", "^", "_"), the space
// token (" "), and the end-of-input token ("EOF") purely from their text.
//
// This mirrors KaTeX's Token class.
type Token struct {
	// Text is the textual content of the token.
	Text string

	// Loc is the source range this token came from.
	Loc source.Location

	// NoExpand, when true, tells the macro expander not to expand this token.
	// Set by `\noexpand`.
	NoExpand bool

	// TreatAsRelax, when true, makes the parser treat this token as `\relax`.
	// Used together with NoExpand for the `\noexpand` semantics.
	TreatAsRelax bool
}

// NewToken constructs a token with the given text and source range.
func NewToken(text string, start, end int) Token {
	return Token{
		Text: text,
		Loc:  source.Location{Start: start, End: end},
	}
}

// EOF returns the synthetic end-of-input token at the given position.
func EOF(pos int) Token { return NewToken(EOFText, pos, pos) }

// IsEOF reports whether t is the synthetic end-of-input token.
func (t Token) IsEOF() bool { return t.Text == EOFText }

// IsCommand reports whether t is a control sequence (its text begins with `\`).
func (t Token) IsCommand() bool {
	return len(t.Text) > 0 && t.Text[0] == '\\'
}

// IsSpace reports whether t is the collapsed-whitespace token (" ").
func (t Token) IsSpace() bool { return t.Text == " " }

// CommandName returns the control sequence name without the leading `\`.
// For tokens that are not control sequences it returns "" and false.
func (t Token) CommandName() (string, bool) {
	if t.IsCommand() {
		return t.Text[1:], true
	}
	return "", false
}

// RangeTo returns a token spanning from t to end with the given text. The
// resulting token has both NoExpand and TreatAsRelax cleared.
func (t Token) RangeTo(end Token, text string) Token {
	return Token{
		Text: text,
		Loc:  source.Range(t.Loc, end.Loc),
	}
}
