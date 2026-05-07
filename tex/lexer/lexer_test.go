package lexer

import (
	"reflect"
	"testing"
)

// lexTexts is a test helper that returns just the text of each token,
// including the final "EOF" sentinel — mirroring the upstream Rust helper.
func lexTexts(input string) []string {
	out := []string{}
	for _, t := range New(input).All() {
		out = append(out, t.Text)
	}
	return out
}

func assertTokens(t *testing.T, input string, want []string) {
	t.Helper()
	got := lexTexts(input)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lex(%q):\n  got:  %q\n  want: %q", input, got, want)
	}
}

// === Basic character tokens ===

func TestSingleLetter(t *testing.T)    { assertTokens(t, "a", []string{"a", "EOF"}) }
func TestMultipleLetters(t *testing.T) { assertTokens(t, "abc", []string{"a", "b", "c", "EOF"}) }
func TestDigit(t *testing.T)           { assertTokens(t, "123", []string{"1", "2", "3", "EOF"}) }
func TestEmptyInput(t *testing.T)      { assertTokens(t, "", []string{"EOF"}) }

// === Control sequences ===

func TestControlWord(t *testing.T)      { assertTokens(t, `\frac`, []string{`\frac`, "EOF"}) }
func TestControlWordAlpha(t *testing.T) { assertTokens(t, `\alpha`, []string{`\alpha`, "EOF"}) }
func TestControlWordFollowedByLetter(t *testing.T) {
	// `\frac x` → control word eats trailing space, then x is its own token.
	assertTokens(t, `\frac x`, []string{`\frac`, "x", "EOF"})
}
func TestControlWordNoSpace(t *testing.T) {
	assertTokens(t, `\fracx`, []string{`\fracx`, "EOF"})
}
func TestControlWordBrace(t *testing.T) {
	assertTokens(t, `\frac{}`, []string{`\frac`, "{", "}", "EOF"})
}
func TestControlSymbol(t *testing.T)          { assertTokens(t, `\,`, []string{`\,`, "EOF"}) }
func TestControlSymbolSemicolon(t *testing.T) { assertTokens(t, `\;`, []string{`\;`, "EOF"}) }
func TestControlSpace(t *testing.T)           { assertTokens(t, "\\ ", []string{"\\ ", "EOF"}) }
func TestDoubleBackslash(t *testing.T) {
	assertTokens(t, `\\`, []string{`\\`, "EOF"})
}

// === Whitespace ===

func TestWhitespaceCollapsed(t *testing.T) {
	assertTokens(t, "a   b", []string{"a", " ", "b", "EOF"})
}
func TestNewlineAsSpace(t *testing.T) {
	assertTokens(t, "a\nb", []string{"a", " ", "b", "EOF"})
}
func TestTabAsSpace(t *testing.T) {
	assertTokens(t, "a\tb", []string{"a", " ", "b", "EOF"})
}
func TestControlWordEatsTrailingSpace(t *testing.T) {
	assertTokens(t, `\frac  x`, []string{`\frac`, "x", "EOF"})
}

// === Comments ===

func TestCommentToEOL(t *testing.T) {
	assertTokens(t, "a%comment\nb", []string{"a", "b", "EOF"})
}
func TestCommentAtEnd(t *testing.T) {
	assertTokens(t, "a%comment", []string{"a", "EOF"})
}
func TestCommentWithCommands(t *testing.T) {
	assertTokens(t, "\\alpha%skip\n\\beta", []string{`\alpha`, `\beta`, "EOF"})
}

// === Special characters ===

func TestBraces(t *testing.T) { assertTokens(t, "{x}", []string{"{", "x", "}", "EOF"}) }
func TestCaretUnderscore(t *testing.T) {
	assertTokens(t, "a^2_3", []string{"a", "^", "2", "_", "3", "EOF"})
}
func TestAmpersand(t *testing.T) {
	assertTokens(t, "a&b", []string{"a", "&", "b", "EOF"})
}
func TestTildeAsActive(t *testing.T) {
	assertTokens(t, "a~b", []string{"a", "~", "b", "EOF"})
}

// === Complex expressions ===

func TestFracExpression(t *testing.T) {
	assertTokens(t, `\frac{a^2}{b}`, []string{`\frac`, "{", "a", "^", "2", "}", "{", "b", "}", "EOF"})
}
func TestSqrtWithOptional(t *testing.T) {
	assertTokens(t, `\sqrt[3]{x}`, []string{`\sqrt`, "[", "3", "]", "{", "x", "}", "EOF"})
}
func TestComplexFraction(t *testing.T) {
	assertTokens(t, `\frac{a + b}{c - d}`, []string{
		`\frac`, "{", "a", " ", "+", " ", "b", "}",
		"{", "c", " ", "-", " ", "d", "}", "EOF",
	})
}
func TestSumWithLimits(t *testing.T) {
	assertTokens(t, `\sum_{i=0}^{n}`, []string{
		`\sum`, "_", "{", "i", "=", "0", "}", "^", "{", "n", "}", "EOF",
	})
}
func TestMatrixRow(t *testing.T) {
	assertTokens(t, `a & b \\ c & d`, []string{
		"a", " ", "&", " ", "b", " ", `\\`, " ", "c", " ", "&", " ", "d", "EOF",
	})
}
func TestNestedFrac(t *testing.T) {
	assertTokens(t, `\frac{\sqrt{a^2+b^2}}{c}`, []string{
		`\frac`, "{", `\sqrt`, "{", "a", "^", "2", "+", "b", "^", "2",
		"}", "}", "{", "c", "}", "EOF",
	})
}

// === Unicode ===

func TestUnicodeChar(t *testing.T)  { assertTokens(t, "α", []string{"α", "EOF"}) }
func TestUnicodeMixed(t *testing.T) { assertTokens(t, "x + α", []string{"x", " ", "+", " ", "α", "EOF"}) }

// === Edge cases ===

func TestBackslashAtEnd(t *testing.T) {
	assertTokens(t, `\`, []string{`\`, "EOF"})
}
func TestMultipleCommands(t *testing.T) {
	assertTokens(t, `\alpha\beta`, []string{`\alpha`, `\beta`, "EOF"})
}
func TestCommandThenDigit(t *testing.T) {
	assertTokens(t, `\frac1`, []string{`\frac`, "1", "EOF"})
}
func TestEqualsSign(t *testing.T) {
	assertTokens(t, "x = 1", []string{"x", " ", "=", " ", "1", "EOF"})
}

// === Source locations ===

func TestSourceLocations(t *testing.T) {
	l := New(`\frac{a}`)
	t1 := l.Lex()
	if t1.Text != `\frac` || t1.Loc.Start != 0 || t1.Loc.End != 5 {
		t.Errorf("t1 = %+v", t1)
	}
	t2 := l.Lex()
	if t2.Text != "{" || t2.Loc.Start != 5 {
		t.Errorf("t2 = %+v", t2)
	}
	t3 := l.Lex()
	if t3.Text != "a" || t3.Loc.Start != 6 {
		t.Errorf("t3 = %+v", t3)
	}
	t4 := l.Lex()
	if t4.Text != "}" || t4.Loc.Start != 7 {
		t.Errorf("t4 = %+v", t4)
	}
}

// =========================================================================
// KaTeX katex-spec.ts: "A parser" describe block — whitespace behavior
// =========================================================================

func TestKatexEmptyString(t *testing.T) { assertTokens(t, "", []string{"EOF"}) }
func TestKatexWhitespaceAroundAndBetween(t *testing.T) {
	assertTokens(t, " x y ", []string{" ", "x", " ", "y", " ", "EOF"})
}
func TestKatexWhitespaceInAtom(t *testing.T) {
	assertTokens(t, " x ^ y ", []string{" ", "x", " ", "^", " ", "y", " ", "EOF"})
}

// =========================================================================
// KaTeX katex-spec.ts: "A comment parser" describe block
// =========================================================================

func TestKatexCommentAtEndOfLine(t *testing.T) {
	assertTokens(t, "a^2 + b^2 = c^2 % Pythagoras' Theorem\n", []string{
		"a", "^", "2", " ", "+", " ", "b", "^", "2", " ", "=", " ", "c", "^", "2", " ", "EOF",
	})
}
func TestKatexCommentAtStartOfLine(t *testing.T) {
	assertTokens(t, "% comment\n", []string{"EOF"})
}
func TestKatexMultipleCommentLines(t *testing.T) {
	assertTokens(t, "% comment 1\n% comment 2\n", []string{"EOF"})
}
func TestKatexCommentBetweenSubSup(t *testing.T) {
	assertTokens(t, "x_3 %comment\n^2", []string{"x", "_", "3", " ", "^", "2", "EOF"})
	assertTokens(t, "x_3^2", []string{"x", "_", "3", "^", "2", "EOF"})
}
func TestKatexCommentAfterCaret(t *testing.T) {
	assertTokens(t, "x^ %comment\n{2}", []string{"x", "^", " ", "{", "2", "}", "EOF"})
}
func TestKatexCommentBeforeFrac(t *testing.T) {
	assertTokens(t, "x^ %comment\n\\frac{1}{2}", []string{
		"x", "^", " ", `\frac`, "{", "1", "}", "{", "2", "}", "EOF",
	})
}
func TestKatexCommentInKern(t *testing.T) {
	assertTokens(t, "\\kern{1 %kern\nem}", []string{
		`\kern`, "{", "1", " ", "e", "m", "}", "EOF",
	})
}
func TestKatexCommentInKernNoBrace(t *testing.T) {
	assertTokens(t, "\\kern1 %kern\nem", []string{`\kern`, "1", " ", "e", "m", "EOF"})
}
func TestKatexCommentInColor(t *testing.T) {
	assertTokens(t, "\\color{#f00%red\n}", []string{
		`\color`, "{", "#", "f", "0", "0", "}", "EOF",
	})
}
func TestKatexCommentBeforeExpression(t *testing.T) {
	with := lexTexts("%comment\n{2}")
	without := lexTexts("{2}")
	if !reflect.DeepEqual(with, without) {
		t.Errorf("with %v != without %v", with, without)
	}
}
func TestKatexCommentNoSpaceProduced(t *testing.T) {
	assertTokens(t, "hello% comment 1\nworld", []string{
		"h", "e", "l", "l", "o", "w", "o", "r", "l", "d", "EOF",
	})
}
func TestKatexCommentThenBlankLine(t *testing.T) {
	assertTokens(t, "hello% comment\n\nworld", []string{
		"h", "e", "l", "l", "o", " ", "w", "o", "r", "l", "d", "EOF",
	})
}
func TestKatexCommentNotInOutput(t *testing.T) {
	assertTokens(t, "5 % comment\n", []string{"5", " ", "EOF"})
}
func TestKatexCommentWithoutNewline(t *testing.T) {
	assertTokens(t, "x%y", []string{"x", "EOF"})
}

// =========================================================================
// KaTeX katex-spec.ts: backslash + newline in \text
// =========================================================================

func TestKatexBackslashWhitespaceSequence(t *testing.T) {
	assertTokens(t, "\\ \t\r \n \t\r ", []string{"\\ ", "EOF"})
}

// =========================================================================
// Combining diacritical marks
// =========================================================================

func TestCombiningDiacriticalMark(t *testing.T) {
	assertTokens(t, "á", []string{"á", "EOF"})
}
func TestMultipleCombiningMarks(t *testing.T) {
	assertTokens(t, "á̃", []string{"á̃", "EOF"})
}
func TestCombiningMarkBetweenChars(t *testing.T) {
	assertTokens(t, "áb", []string{"á", "b", "EOF"})
}

func TestKatexOrdCharacters(t *testing.T) {
	assertTokens(t, "1234", []string{"1", "2", "3", "4", "EOF"})
}
func TestKatexSpecialSymbols(t *testing.T) {
	assertTokens(t, "|/@.\"'", []string{"|", "/", "@", ".", "\"", "'", "EOF"})
}

// =========================================================================
// `@` in control words: \\[a-zA-Z@]+
// =========================================================================

func TestAtSignInControlWord(t *testing.T) {
	assertTokens(t, `\foo@bar`, []string{`\foo@bar`, "EOF"})
}
func TestAtIfStar(t *testing.T) {
	assertTokens(t, `\@ifstar`, []string{`\@ifstar`, "EOF"})
}

// =========================================================================
// \verb / \verb*
// =========================================================================

func TestVerbBasic(t *testing.T) {
	assertTokens(t, `\verb|hello|`, []string{`\verb|hello|`, "EOF"})
}
func TestVerbStar(t *testing.T) {
	assertTokens(t, `\verb*|hello world|`, []string{`\verb*|hello world|`, "EOF"})
}
func TestVerbWithSpecialChars(t *testing.T) {
	assertTokens(t, `\verb!\frac{a}{b}!`, []string{`\verb!\frac{a}{b}!`, "EOF"})
}
func TestVerbInExpression(t *testing.T) {
	assertTokens(t, `x + \verb|y| + z`, []string{
		"x", " ", "+", " ", `\verb|y|`, " ", "+", " ", "z", "EOF",
	})
}

// =========================================================================
// Real-world LaTeX coverage
// =========================================================================

func TestKatexQuadraticFormula(t *testing.T) {
	assertTokens(t, `\frac{-b \pm \sqrt{b^2-4ac}}{2a}`, []string{
		`\frac`, "{", "-", "b", " ", `\pm`, `\sqrt`, "{",
		"b", "^", "2", "-", "4", "a", "c", "}", "}", "{",
		"2", "a", "}", "EOF",
	})
}
func TestKatexMatrixBeginEnd(t *testing.T) {
	assertTokens(t, `\begin{pmatrix} a & b \\ c & d \end{pmatrix}`, []string{
		`\begin`, "{", "p", "m", "a", "t", "r", "i", "x", "}",
		" ", "a", " ", "&", " ", "b", " ", `\\`, " ",
		"c", " ", "&", " ", "d", " ",
		`\end`, "{", "p", "m", "a", "t", "r", "i", "x", "}",
		"EOF",
	})
}
