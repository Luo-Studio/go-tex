package lexer

import "testing"

func TestTokenCreation(t *testing.T) {
	tok := NewToken("\\frac", 0, 5)
	if tok.Text != "\\frac" {
		t.Errorf("Text = %q, want %q", tok.Text, "\\frac")
	}
	if tok.Loc.Start != 0 || tok.Loc.End != 5 {
		t.Errorf("Loc = %+v, want {0 5}", tok.Loc)
	}
}

func TestIsCommand(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"\\frac", true},
		{"\\ ", true},
		{"a", false},
		{"{", false},
	}
	for _, c := range cases {
		if got := NewToken(c.text, 0, len(c.text)).IsCommand(); got != c.want {
			t.Errorf("IsCommand(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestIsEOF(t *testing.T) {
	if !EOF(10).IsEOF() {
		t.Error("EOF token should report IsEOF")
	}
	if NewToken("x", 0, 1).IsEOF() {
		t.Error("non-EOF token should not report IsEOF")
	}
}

func TestCommandName(t *testing.T) {
	cases := []struct {
		text string
		want string
		ok   bool
	}{
		{"\\frac", "frac", true},
		{"\\alpha", "alpha", true},
		{"\\ ", " ", true},
		{"a", "", false},
	}
	for _, c := range cases {
		got, ok := NewToken(c.text, 0, len(c.text)).CommandName()
		if got != c.want || ok != c.ok {
			t.Errorf("CommandName(%q) = (%q, %v), want (%q, %v)", c.text, got, ok, c.want, c.ok)
		}
	}
}

func TestIsSpace(t *testing.T) {
	if !NewToken(" ", 0, 1).IsSpace() {
		t.Error(`" " should be space`)
	}
	if NewToken("a", 0, 1).IsSpace() {
		t.Error(`"a" should not be space`)
	}
	if NewToken("\\ ", 0, 2).IsSpace() {
		t.Error(`"\ " should not be a (collapsed) space token`)
	}
}

func TestNoExpandDefaultsFalse(t *testing.T) {
	tok := NewToken("\\foo", 0, 4)
	if tok.NoExpand || tok.TreatAsRelax {
		t.Errorf("default flags = %+v, want both false", tok)
	}
}

func TestRangeTo(t *testing.T) {
	t1 := NewToken("a", 0, 1)
	t2 := NewToken("c", 4, 5)
	spanned := t1.RangeTo(t2, "abc")
	if spanned.Text != "abc" {
		t.Errorf("Text = %q, want %q", spanned.Text, "abc")
	}
	if spanned.Loc.Start != 0 || spanned.Loc.End != 5 {
		t.Errorf("Loc = %+v, want {0 5}", spanned.Loc)
	}
}
