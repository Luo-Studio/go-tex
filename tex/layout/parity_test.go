package layout

import (
	"bufio"
	"bytes"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luo-studio/go-tex/internal/parity"
	"github.com/luo-studio/go-tex/tex/parser"
)

// TestLayoutParityWithUpstream measures, for every line in the golden
// corpus, how often our layout produces the same {width, height, depth,
// itemCount} summary as the upstream `ratex-layout` binary.
//
// Skips when the binary is unavailable. Set RATEX_LAYOUT or
// RATEX_TARGET_DIR to point at it.
func TestLayoutParityWithUpstream(t *testing.T) {
	upstream := parity.FindUpstreamBin("ratex-layout")
	if upstream == "" {
		upstream = parity.FindUpstreamBin("layout")
	}
	if upstream == "" {
		t.Skip("upstream `ratex-layout` binary not found; set RATEX_LAYOUT or RATEX_TARGET_DIR")
	}

	cases := []string{"test_cases.txt", "test_case_ce.txt"}
	totalAll, totalDimAll := 0, 0
	totalDimMatch, totalCountMatch, totalAllMatch := 0, 0, 0
	for _, name := range cases {
		path := filepath.Join("..", "..", "testdata", "golden", name)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		dimMatch, countMatch, allMatch, count, dimEligible := scoreLayoutFile(t, f, upstream)
		f.Close()
		t.Logf("%s: dims %d/%d (%.2f%%)  itemCount %d/%d  full %d/%d  eligible(non-error) %d",
			name, dimMatch, count, percent(dimMatch, count),
			countMatch, count, allMatch, count, dimEligible)
		totalAll += count
		totalDimAll += dimEligible
		totalDimMatch += dimMatch
		totalCountMatch += countMatch
		totalAllMatch += allMatch
	}
	t.Logf("OVERALL dim parity: %d/%d (%.2f%%)  full parity: %d/%d (%.2f%%)",
		totalDimMatch, totalAll, percent(totalDimMatch, totalAll),
		totalAllMatch, totalAll, percent(totalAllMatch, totalAll))
}

type layoutSummary struct {
	Box struct {
		Width, Height, Depth float64
	} `json:"box"`
	DisplayList struct {
		Width, Height, Depth float64
		ItemCount            int `json:"itemCount"`
	} `json:"displayList"`
	Error bool `json:"error"`
}

func scoreLayoutFile(t *testing.T, f *os.File, upstream string) (dim, cnt, all, total, eligible int) {
	t.Helper()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	for s.Scan() {
		expr := strings.TrimSpace(s.Text())
		if expr == "" {
			continue
		}
		total++
		want := upstreamLayout(t, upstream, expr)
		if want.Error {
			continue
		}
		eligible++
		got := goLayout(expr)
		if got.Error {
			continue
		}
		dimsMatch := closeF(got.Box.Width, want.Box.Width) &&
			closeF(got.Box.Height, want.Box.Height) &&
			closeF(got.Box.Depth, want.Box.Depth)
		if dimsMatch {
			dim++
		}
		if got.DisplayList.ItemCount == want.DisplayList.ItemCount {
			cnt++
		}
		if dimsMatch && got.DisplayList.ItemCount == want.DisplayList.ItemCount {
			all++
		}
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return dim, cnt, all, total, eligible
}

func goLayout(expr string) layoutSummary {
	body, err := parser.Parse(expr)
	if err != nil {
		return layoutSummary{Error: true}
	}
	box := Layout(body, DefaultOptions())
	dl := ToDisplayList(box)
	var s layoutSummary
	s.Box.Width = round5(box.Width)
	s.Box.Height = round5(box.Height)
	s.Box.Depth = round5(box.Depth)
	s.DisplayList.Width = round5(dl.Width)
	s.DisplayList.Height = round5(dl.Height)
	s.DisplayList.Depth = round5(dl.Depth)
	s.DisplayList.ItemCount = len(dl.Items)
	return s
}

func upstreamLayout(t *testing.T, bin, expr string) layoutSummary {
	t.Helper()
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(expr + "\n")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("upstream layout failed on %q: %v: %s", expr, err, errBuf.String())
	}
	var s layoutSummary
	dec := json.NewDecoder(strings.NewReader(out.String()))
	for {
		if err := dec.Decode(&s); err != nil {
			break
		}
	}
	return s
}

func closeF(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func round5(v float64) float64 { return math.Round(v*1e5) / 1e5 }

func percent(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}
