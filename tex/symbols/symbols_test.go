package symbols

import "testing"

func TestEquiv(t *testing.T) {
	got, ok := Lookup(`\equiv`, ModeMath)
	if !ok {
		t.Fatal("expected \\equiv in math mode")
	}
	if got.Group != GroupRel || got.Font != FontMain || got.Codepoint != '≡' {
		t.Errorf("equiv = %+v", got)
	}
}

func TestAlpha(t *testing.T) {
	got, _ := Lookup(`\alpha`, ModeMath)
	if got.Group != GroupMathOrd || got.Codepoint != 'α' {
		t.Errorf("alpha = %+v", got)
	}
}

func TestPlus(t *testing.T) {
	got, _ := Lookup("+", ModeMath)
	if got.Group != GroupBin {
		t.Errorf("+ group = %s, want bin", got.Group)
	}
}

func TestSum(t *testing.T) {
	got, _ := Lookup(`\sum`, ModeMath)
	if got.Group != GroupOpToken {
		t.Errorf("\\sum group = %s, want op-token", got.Group)
	}
}

func TestFracIsNotASymbol(t *testing.T) {
	if _, ok := Lookup(`\frac`, ModeMath); ok {
		t.Error("\\frac should not be in symbols table")
	}
}

func TestDigitsAreTextOrd(t *testing.T) {
	for d := '0'; d <= '9'; d++ {
		s := string([]rune{d})
		got, ok := Lookup(s, ModeMath)
		if !ok || got.Group != GroupTextOrd {
			t.Errorf("digit %s: got %+v ok=%v, want textord", s, got, ok)
		}
	}
}

func TestLowercaseLettersAreMathOrd(t *testing.T) {
	for ch := 'a'; ch <= 'z'; ch++ {
		s := string([]rune{ch})
		got, ok := Lookup(s, ModeMath)
		if !ok || got.Group != GroupMathOrd {
			t.Errorf("letter %s: got %+v ok=%v, want mathord", s, got, ok)
		}
	}
}

func TestAcceptUnicodeChar(t *testing.T) {
	got, ok := Lookup("α", ModeMath)
	if !ok || got.Name != `\alpha` {
		t.Errorf("α: got %+v ok=%v, want \\alpha", got, ok)
	}
}

func TestEqualsIsRel(t *testing.T) {
	got, _ := Lookup("=", ModeMath)
	if got.Group != GroupRel {
		t.Errorf("= group = %s, want rel", got.Group)
	}
}

func TestAMSSymbol(t *testing.T) {
	got, ok := Lookup(`\beth`, ModeMath)
	if !ok || got.Font != FontAMS || got.Codepoint != 'ℶ' {
		t.Errorf("beth = %+v ok=%v", got, ok)
	}
}
