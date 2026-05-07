package lexer

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luo-studio/go-tex/internal/parity"
)

// TestLexParityWithUpstream feeds every line of the golden corpus to both
// the Go lex CLI (built from this module) and the upstream Rust `lex`
// binary, and asserts the outputs are byte-identical.
//
// It skips if the upstream binary cannot be found. Set RATEX_LEX to point at
// it, or run `cargo build --release -p ratex-lexer --bin lex` in a checkout
// of erweixin/RaTeX and set RATEX_TARGET_DIR=<that-checkout>/target.
func TestLexParityWithUpstream(t *testing.T) {
	upstream := parity.FindUpstreamBin("lex")
	if upstream == "" {
		t.Skip("upstream `lex` binary not found; set RATEX_LEX or RATEX_TARGET_DIR")
	}

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
			matched := 0
			total := 0
			var firstFailure string
			for s.Scan() {
				line++
				input := strings.TrimRight(s.Text(), "\r")
				total++
				goOut := lexCLI(t, input)
				rustOut := runUpstream(t, upstream, input)
				if goOut == rustOut {
					matched++
					continue
				}
				if firstFailure == "" {
					firstFailure = formatDiff(line, input, goOut, rustOut)
				}
			}
			if err := s.Err(); err != nil {
				t.Fatalf("scan %s: %v", name, err)
			}
			t.Logf("%s: %d/%d byte-identical to upstream lex", name, matched, total)
			if matched != total {
				t.Errorf("lex parity failed: %d/%d match\n%s", matched, total, firstFailure)
			}
		})
	}
}

// lexCLI runs the in-process equivalent of `cmd/lex` to keep the test
// hermetic — no separate go binary needed.
func lexCLI(t *testing.T, input string) string {
	t.Helper()
	var out bytes.Buffer
	for _, tok := range New(input).All() {
		text := tok.Text
		if text != EOFText {
			text = strings.ReplaceAll(text, "\n", `\n`)
		}
		out.WriteString(text)
		out.WriteByte('\n')
	}
	return out.String()
}

func runUpstream(t *testing.T, bin, input string) string {
	t.Helper()
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(input)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("upstream lex failed on %q: %v: %s", input, err, errBuf.String())
	}
	return out.String()
}

func formatDiff(line int, input, got, want string) string {
	gotLines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	wantLines := strings.Split(strings.TrimSuffix(want, "\n"), "\n")
	var b strings.Builder
	b.WriteString("first failure at line ")
	b.WriteString(itoa(line))
	b.WriteString(": ")
	b.WriteString(quoteOneLine(input))
	b.WriteByte('\n')
	max := len(gotLines)
	if len(wantLines) > max {
		max = len(wantLines)
	}
	for i := 0; i < max; i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		marker := " "
		if g != w {
			marker = "*"
		}
		b.WriteString(marker)
		b.WriteString(" go=")
		b.WriteString(quoteOneLine(g))
		b.WriteString(" rust=")
		b.WriteString(quoteOneLine(w))
		b.WriteByte('\n')
	}
	return b.String()
}

func quoteOneLine(s string) string {
	const max = 80
	q := strings.ReplaceAll(s, "\n", `\n`)
	if len(q) > max {
		q = q[:max] + "..."
	}
	return "'" + q + "'"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
