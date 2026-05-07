package parser

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/luo-studio/go-tex/tex/source"
)

func TestMathOrdJSON(t *testing.T) {
	n := &MathOrd{Mode: ModeMath, Text: "x", Loc: &source.Location{Start: 0, End: 1}}
	got, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"mathord","mode":"math","text":"x","loc":{"start":0,"end":1}}`
	if string(got) != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}

func TestSupSubJSON(t *testing.T) {
	n := &SupSub{
		Mode: ModeMath,
		Base: &MathOrd{Mode: ModeMath, Text: "x"},
		Sup:  &TextOrd{Mode: ModeMath, Text: "2"},
	}
	got, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	g := string(got)
	for _, want := range []string{
		`"type":"supsub"`,
		`"base":{"type":"mathord"`,
		`"sup":{"type":"textord"`,
	} {
		if !strings.Contains(g, want) {
			t.Errorf("missing %s in %s", want, g)
		}
	}
}

func TestGenFracJSON(t *testing.T) {
	n := &GenFrac{
		Mode:       ModeMath,
		Continued:  false,
		Numer:      &MathOrd{Mode: ModeMath, Text: "a"},
		Denom:      &MathOrd{Mode: ModeMath, Text: "b"},
		HasBarLine: true,
	}
	got, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	g := string(got)
	if !strings.Contains(g, `"type":"genfrac"`) || !strings.Contains(g, `"hasBarLine":true`) {
		t.Errorf("got %s", g)
	}
	// All four pointer fields are expected to serialize as `null` when nil
	// (matching serde's default for Option<T> without skip_serializing_if).
	if !strings.Contains(g, `"leftDelim":null`) {
		t.Errorf("expected leftDelim:null in %s", g)
	}
}

func TestNormalizeArgument(t *testing.T) {
	g := &OrdGroup{Mode: ModeMath, Body: []Node{&MathOrd{Mode: ModeMath, Text: "x"}}}
	n := NormalizeArgument(g)
	if n.NodeType() != "mathord" {
		t.Errorf("expected mathord, got %s", n.NodeType())
	}
}
