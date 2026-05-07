package parser

import (
	"encoding/json"
	"testing"
)

func parseToJSON(t *testing.T, input string) string {
	t.Helper()
	body, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q): %v", input, err)
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseSingleLetter(t *testing.T) {
	want := `[{"type":"mathord","mode":"math","text":"x","loc":{"start":0,"end":1}}]`
	if got := parseToJSON(t, "x"); got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}

func TestParseDigit(t *testing.T) {
	want := `[{"type":"textord","mode":"math","text":"1","loc":{"start":0,"end":1}}]`
	if got := parseToJSON(t, "1"); got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}

func TestParseAB(t *testing.T) {
	got := parseToJSON(t, "ab")
	want := `[{"type":"mathord","mode":"math","text":"a","loc":{"start":0,"end":1}},` +
		`{"type":"mathord","mode":"math","text":"b","loc":{"start":1,"end":2}}]`
	if got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}

func TestParseAlpha(t *testing.T) {
	got := parseToJSON(t, `\alpha`)
	want := `[{"type":"mathord","mode":"math","text":"\\alpha","loc":{"start":0,"end":6}}]`
	if got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}

func TestParsePlus(t *testing.T) {
	got := parseToJSON(t, "a+b")
	want := `[{"type":"mathord","mode":"math","text":"a","loc":{"start":0,"end":1}},` +
		`{"type":"atom","mode":"math","family":"bin","text":"+","loc":{"start":1,"end":2}},` +
		`{"type":"mathord","mode":"math","text":"b","loc":{"start":2,"end":3}}]`
	if got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}

func TestParseGroup(t *testing.T) {
	got := parseToJSON(t, "{x}")
	want := `[{"type":"ordgroup","mode":"math","body":[` +
		`{"type":"mathord","mode":"math","text":"x","loc":{"start":1,"end":2}}` +
		`],"loc":{"start":0,"end":3}}]`
	if got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}
