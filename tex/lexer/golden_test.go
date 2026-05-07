package lexer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLexAllGoldenCases is a smoke test: every line in the upstream
// golden corpus must lex without panicking and must terminate on EOF.
// It does not check token content (a separate harness will diff against the
// upstream lex CLI). The intent is to catch regressions early.
func TestLexAllGoldenCases(t *testing.T) {
	for _, name := range []string{"test_cases.txt", "test_case_ce.txt"} {
		path := filepath.Join("..", "..", "testdata", "golden", name)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		t.Run(name, func(t *testing.T) {
			defer f.Close()
			s := bufio.NewScanner(f)
			s.Buffer(make([]byte, 1<<16), 1<<20)
			line := 0
			for s.Scan() {
				line++
				input := strings.TrimRight(s.Text(), "\r")
				toks := New(input).All()
				if len(toks) == 0 {
					t.Errorf("%s:%d empty token slice for %q", name, line, input)
					continue
				}
				if !toks[len(toks)-1].IsEOF() {
					t.Errorf("%s:%d last token not EOF for %q (got %q)", name, line, input, toks[len(toks)-1].Text)
				}
			}
			if err := s.Err(); err != nil {
				t.Fatalf("scan %s: %v", name, err)
			}
		})
	}
}
