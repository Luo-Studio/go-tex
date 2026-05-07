package mathstyle

import "testing"

func TestNumeratorAllStyles(t *testing.T) {
	// KaTeX: fracNum = [T, Tc, S, Sc, SS, SSc, SS, SSc]
	cases := []struct{ in, want Style }{
		{Display, Text},
		{DisplayCramped, TextCramped},
		{Text, Script},
		{TextCramped, ScriptCramped},
		{Script, ScriptScript},
		{ScriptCramped, ScriptScriptCramped},
		{ScriptScript, ScriptScript},
		{ScriptScriptCramped, ScriptScriptCramped},
	}
	for _, c := range cases {
		if got := c.in.Numerator(); got != c.want {
			t.Errorf("%s.Numerator() = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestDenominatorAllStyles(t *testing.T) {
	cases := []struct{ in, want Style }{
		{Display, TextCramped},
		{DisplayCramped, TextCramped},
		{Text, ScriptCramped},
		{TextCramped, ScriptCramped},
		{Script, ScriptScriptCramped},
		{ScriptCramped, ScriptScriptCramped},
		{ScriptScript, ScriptScriptCramped},
		{ScriptScriptCramped, ScriptScriptCramped},
	}
	for _, c := range cases {
		if got := c.in.Denominator(); got != c.want {
			t.Errorf("%s.Denominator() = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestSuperscriptAllStyles(t *testing.T) {
	cases := []struct{ in, want Style }{
		{Display, Script},
		{DisplayCramped, ScriptCramped},
		{Text, Script},
		{TextCramped, ScriptCramped},
		{Script, ScriptScript},
		{ScriptCramped, ScriptScriptCramped},
		{ScriptScript, ScriptScript},
		{ScriptScriptCramped, ScriptScriptCramped},
	}
	for _, c := range cases {
		if got := c.in.Superscript(); got != c.want {
			t.Errorf("%s.Superscript() = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestSubscriptAllStyles(t *testing.T) {
	cases := []struct{ in, want Style }{
		{Display, ScriptCramped},
		{DisplayCramped, ScriptCramped},
		{Text, ScriptCramped},
		{TextCramped, ScriptCramped},
		{Script, ScriptScriptCramped},
		{ScriptCramped, ScriptScriptCramped},
		{ScriptScript, ScriptScriptCramped},
		{ScriptScriptCramped, ScriptScriptCramped},
	}
	for _, c := range cases {
		if got := c.in.Subscript(); got != c.want {
			t.Errorf("%s.Subscript() = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestCrampedAllStyles(t *testing.T) {
	cases := []struct{ in, want Style }{
		{Display, DisplayCramped},
		{DisplayCramped, DisplayCramped},
		{Text, TextCramped},
		{TextCramped, TextCramped},
		{Script, ScriptCramped},
		{ScriptCramped, ScriptCramped},
		{ScriptScript, ScriptScriptCramped},
		{ScriptScriptCramped, ScriptScriptCramped},
	}
	for _, c := range cases {
		if got := c.in.Cramped(); got != c.want {
			t.Errorf("%s.Cramped() = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestTextAllStyles(t *testing.T) {
	cases := []struct{ in, want Style }{
		{Display, Display},
		{DisplayCramped, DisplayCramped},
		{Text, Text},
		{TextCramped, TextCramped},
		{Script, Text},
		{ScriptCramped, TextCramped},
		{ScriptScript, Text},
		{ScriptScriptCramped, TextCramped},
	}
	for _, c := range cases {
		if got := c.in.Text(); got != c.want {
			t.Errorf("%s.Text() = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestIsDisplay(t *testing.T) {
	if !Display.IsDisplay() || !DisplayCramped.IsDisplay() {
		t.Error("Display/DisplayCramped should be display")
	}
	if Text.IsDisplay() || Script.IsDisplay() {
		t.Error("Text/Script should not be display")
	}
}

func TestIsTight(t *testing.T) {
	if Display.IsTight() || Text.IsTight() {
		t.Error("Display/Text should not be tight")
	}
	for _, s := range []Style{Script, ScriptCramped, ScriptScript, ScriptScriptCramped} {
		if !s.IsTight() {
			t.Errorf("%s should be tight", s)
		}
	}
}

func TestSizeIndex(t *testing.T) {
	cases := []struct {
		in   Style
		want int
	}{
		{Display, 0}, {Text, 0}, {Script, 1}, {ScriptScript, 2},
	}
	for _, c := range cases {
		if got := c.in.SizeIndex(); got != c.want {
			t.Errorf("%s.SizeIndex() = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestSizeMultiplier(t *testing.T) {
	cases := []struct {
		in   Style
		want float64
	}{
		{Display, 1.0}, {Text, 1.0}, {Script, 0.7}, {ScriptScript, 0.5},
	}
	for _, c := range cases {
		if got := c.in.SizeMultiplier(); got != c.want {
			t.Errorf("%s.SizeMultiplier() = %v, want %v", c.in, got, c.want)
		}
	}
}
