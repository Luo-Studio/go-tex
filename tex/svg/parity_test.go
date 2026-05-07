package svg

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luo-studio/go-tex/internal/parity"
	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
)

// TestSVGParityWithUpstream measures byte-identical SVG output against the
// upstream `render-svg` CLI in default (text-mode) flavour, on the full
// golden corpus. Path-glyph (TTF embed) parity is a separate harness once
// the font loader is ported.
func TestSVGParityWithUpstream(t *testing.T) {
	upstream := parity.FindUpstreamBin("render-svg")
	if upstream == "" {
		t.Skip("upstream `render-svg` binary not found; set RATEX_RENDER_SVG or RATEX_TARGET_DIR")
	}
	tmp := t.TempDir()

	for _, name := range []string{"test_cases.txt", "test_case_ce.txt"} {
		corpusPath := filepath.Join("..", "..", "testdata", "golden", name)
		f, err := os.Open(corpusPath)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer f.Close()

		// Run upstream once on the whole corpus into a tmp dir.
		outDir := filepath.Join(tmp, strings.TrimSuffix(name, ".txt"))
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			t.Fatal(err)
		}
		corpusBytes, _ := os.ReadFile(corpusPath)
		cmd := exec.Command(upstream, "--output-dir", outDir, "--font-size", "40")
		cmd.Stdin = bytes.NewReader(corpusBytes)
		var errBuf bytes.Buffer
		cmd.Stderr = &errBuf
		if err := cmd.Run(); err != nil {
			t.Fatalf("upstream render-svg %s: %v: %s", name, err, errBuf.String())
		}

		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 1<<16), 1<<20)
		match, total := 0, 0
		var firstFailExpr string
		idx := 0
		for s.Scan() {
			expr := strings.TrimSpace(s.Text())
			if expr == "" {
				continue
			}
			idx++
			total++
			body, err := parser.Parse(expr)
			if err != nil {
				continue
			}
			box := layout.Layout(body, layout.DefaultOptions())
			dl := layout.ToDisplayList(box)
			got := Render(dl, DefaultOptions())
			wantPath := filepath.Join(outDir, formatIdx(idx)+".svg")
			want, err := os.ReadFile(wantPath)
			if err != nil {
				continue
			}
			if got == string(want) {
				match++
			} else if firstFailExpr == "" {
				firstFailExpr = expr
			}
		}
		t.Logf("%s: SVG byte parity %d/%d (%.2f%%)  first miss: %q",
			name, match, total, percent(match, total), firstFailExpr)
	}
}

func formatIdx(i int) string {
	s := []byte("0000")
	for j := 3; j >= 0; j-- {
		s[j] = byte('0' + i%10)
		i /= 10
	}
	return string(s)
}

func percent(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}
