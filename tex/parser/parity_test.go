package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luo-studio/go-tex/internal/parity"
)

// TestParseParityWithUpstream measures, for every line in the golden corpus,
// how often our parser produces byte-identical JSON to the upstream
// `parse` binary. Skips when the binary is unavailable.
//
// The intent is a running score the user can watch grow as functions are
// added to the parser.
func TestParseParityWithUpstream(t *testing.T) {
	upstream := parity.FindUpstreamBin("parse")
	if upstream == "" {
		t.Skip("upstream `parse` binary not found; set RATEX_PARSE or RATEX_TARGET_DIR")
	}

	cases := []string{"test_cases.txt", "test_case_ce.txt"}
	totalMatch, totalCount := 0, 0
	for _, name := range cases {
		path := filepath.Join("..", "..", "testdata", "golden", name)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		match, count, _ := scoreFile(t, f, upstream)
		f.Close()
		totalMatch += match
		totalCount += count
		t.Logf("%s: %d/%d byte-identical (%.2f%%)", name, match, count, percent(match, count))
	}
	t.Logf("OVERALL: %d/%d byte-identical (%.2f%%)", totalMatch, totalCount, percent(totalMatch, totalCount))
}

func scoreFile(t *testing.T, r *os.File, upstream string) (match, count int, examples []string) {
	t.Helper()
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	for s.Scan() {
		expr := strings.TrimSpace(s.Text())
		if expr == "" {
			continue
		}
		count++
		goOut, _ := goParseLine(expr)
		rustOut := upstreamParseLine(t, upstream, expr)
		if jsonsEqual(goOut, rustOut) {
			match++
		}
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return match, count, examples
}

// goParseLine runs the in-process equivalent of cmd/parse for a single line.
func goParseLine(expr string) (string, error) {
	body, err := Parse(expr)
	if err != nil {
		return errorEnvelope(expr, err), nil
	}
	js, err := json.Marshal(body)
	if err != nil {
		return errorEnvelope(expr, err), nil
	}
	return string(js), nil
}

func errorEnvelope(expr string, err error) string {
	js, _ := json.Marshal(struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
		Input   string `json:"input"`
	}{true, err.Error(), expr})
	return string(js)
}

func upstreamParseLine(t *testing.T, bin, expr string) string {
	t.Helper()
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(expr + "\n")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("upstream parse failed on %q: %v: %s", expr, err, errBuf.String())
	}
	return strings.TrimRight(out.String(), "\n")
}

// jsonsEqual compares two JSON strings semantically — order-preserving for
// arrays, but key-order-insensitive for objects, so trivial map-iteration
// differences don't degrade the score.
func jsonsEqual(a, b string) bool {
	if a == b {
		return true
	}
	var ja, jb any
	if err := json.Unmarshal([]byte(a), &ja); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &jb); err != nil {
		return false
	}
	return jsonDeepEqual(ja, jb)
}

func jsonDeepEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, va := range av {
			vb, ok := bv[k]
			if !ok || !jsonDeepEqual(va, vb) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonDeepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func percent(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}
