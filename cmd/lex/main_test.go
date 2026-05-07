package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "x", "x\nEOF\n"},
		{"frac", `\frac{a}{b}`, "\\frac\n{\na\n}\n{\nb\n}\nEOF\n"},
		{"trim trailing newline", "x\n", "x\nEOF\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(strings.NewReader(c.in), &out); err != nil {
				t.Fatalf("run: %v", err)
			}
			if got := out.String(); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
