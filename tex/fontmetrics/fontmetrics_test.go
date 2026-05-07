package fontmetrics

import "testing"

func TestMainRegularA(t *testing.T) {
	m, ok := Lookup(FontMainRegular, 'a')
	if !ok {
		t.Fatal("Main-Regular 'a' not found")
	}
	if abs(m.Height-0.43056) > 0.001 {
		t.Errorf("height = %v, want ~0.43056", m.Height)
	}
	if m.Width <= 0 {
		t.Errorf("width should be positive, got %v", m.Width)
	}
}

func TestMainRegularUppercaseA(t *testing.T) {
	m, ok := Lookup(FontMainRegular, 'A')
	if !ok {
		t.Fatal("Main-Regular 'A' not found")
	}
	if abs(m.Height-0.68333) > 0.001 {
		t.Errorf("height = %v, want ~0.68333", m.Height)
	}
}

func TestMainRegularSpace(t *testing.T) {
	m, ok := Lookup(FontMainRegular, ' ')
	if !ok {
		t.Fatal("space not found")
	}
	if abs(m.Width-0.25) > 0.001 {
		t.Errorf("width = %v, want 0.25", m.Width)
	}
}

func TestAMSRegular(t *testing.T) {
	m, ok := Lookup(FontAMSRegular, 'A')
	if !ok {
		t.Fatal("AMS-Regular 'A' not found")
	}
	if abs(m.Height-0.68889) > 0.001 {
		t.Errorf("height = %v", m.Height)
	}
}

func TestExtraCharFallbackCyrillic(t *testing.T) {
	if _, ok := Lookup(FontMainRegular, 'А'); ok {
		t.Error("Cyrillic А shouldn't be in direct table")
	}
	m, ok := LookupWithFallback(FontMainRegular, 'А')
	if !ok || m.Height <= 0 {
		t.Errorf("fallback failed: %v / %v", m, ok)
	}
}

func TestMathConstantsTextstyle(t *testing.T) {
	mc := Global(0)
	if abs(mc.AxisHeight-0.25) > 0.001 {
		t.Errorf("AxisHeight = %v", mc.AxisHeight)
	}
	if abs(mc.Quad-1.0) > 0.001 {
		t.Errorf("Quad = %v", mc.Quad)
	}
}

func TestMathConstantsScriptstyle(t *testing.T) {
	mc := Global(1)
	if abs(mc.Quad-1.171) > 0.001 {
		t.Errorf("Quad = %v", mc.Quad)
	}
}

func TestCSSEmPerMu(t *testing.T) {
	mc := Global(0)
	if abs(mc.CSSEmPerMu()-1.0/18.0) > 0.0001 {
		t.Errorf("CSSEmPerMu = %v", mc.CSSEmPerMu())
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
