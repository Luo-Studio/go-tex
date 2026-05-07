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

func TestChemParseStrStub(t *testing.T) {
	_, err := ChemParseStr("H2O", "ce")
	if err != ErrNotImplemented {
		t.Errorf("ChemParseStr returned %v, want ErrNotImplemented", err)
	}
}
