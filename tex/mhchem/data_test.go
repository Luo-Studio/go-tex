package mhchem

import "testing"

func TestLoadData(t *testing.T) {
	d, err := LoadData()
	if err != nil {
		t.Fatalf("LoadData: %v", err)
	}
	if len(d.Machines) == 0 {
		t.Fatal("expected non-empty machines table")
	}
	if _, ok := d.Machines["ce"]; !ok {
		t.Error("expected 'ce' machine")
	}
	if _, ok := d.Machines["pu"]; !ok {
		t.Error("expected 'pu' machine")
	}
	if len(d.Regexes) == 0 {
		t.Fatal("expected compiled regex patterns")
	}
}

func TestBuffer(t *testing.T) {
	b := NewBuffer()
	SetSlot(&b.A, "x")
	if b.A == nil || *b.A != "x" {
		t.Errorf("A = %v, want 'x'", b.A)
	}
	SetSlot(&b.A, "y")
	if *b.A != "y" {
		t.Errorf("A = %s, want 'y'", *b.A)
	}
	if !IsSlotEmpty(b.B) {
		t.Error("B should be empty")
	}
	b.ClearSoft()
	if !IsSlotEmpty(b.A) {
		t.Error("A should be cleared")
	}
}

func TestChemParseStrSmoke(t *testing.T) {
	out, err := ChemParseStr("H2O", "ce")
	if err != nil {
		t.Fatalf("ChemParseStr: %v", err)
	}
	if out == "" {
		t.Fatal("ChemParseStr returned empty string")
	}
	if !contains(out, "H") {
		t.Errorf("output %q doesn't contain 'H'", out)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
