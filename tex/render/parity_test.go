package render

import (
	"bufio"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"bytes"

	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
)

// TestPNGStructuralParity is a sanity harness: for each golden case,
// render via parser→layout→canvasr→PNG, decode upstream's reference SVG
// to read its dimensions, and report:
//
//   - decoded: how many of our PNGs decode successfully.
//   - dim-match: how many have the same pixel dimensions as upstream
//     (SVG width/height rounded to nearest integer pixel).
//
// We dropped PNG goldens — bit-exact PNG comparison is infeasible without
// porting ab_glyph's rasteriser, and dimensions are equally available
// from the SVG corpus.
func TestPNGStructuralParity(t *testing.T) {
	corpusPath := filepath.Join("..", "..", "testdata", "golden", "test_cases.txt")
	goldenDir := filepath.Join("..", "..", "testdata", "golden", "output_svg")
	if _, err := os.Stat(goldenDir); err != nil {
		t.Skip("golden output_svg dir not present")
	}
	f, err := os.Open(corpusPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	idx := 0
	dimMatch, decoded, total := 0, 0, 0
	var firstFailure string
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
		ours, err := PNG(dl)
		if err != nil {
			if firstFailure == "" {
				firstFailure = expr + " (render: " + err.Error() + ")"
			}
			continue
		}
		decoded++
		ow, ohErr := pngDims(ours)
		if ohErr != nil {
			continue
		}
		wantPath := filepath.Join(goldenDir, formatIdx(idx)+".svg")
		want, err := os.ReadFile(wantPath)
		if err != nil {
			continue
		}
		ww, whErr := svgDims(want)
		if whErr == nil && ow == ww {
			dimMatch++
		}
	}
	t.Logf("PNG: %d cases  decoded %d  dim-match %d (%.2f%%)\nfirst failure: %q",
		total, decoded, dimMatch, percent(dimMatch, total), firstFailure)
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

type pngWH struct{ w, h int }

func pngDims(b []byte) (pngWH, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return pngWH{}, err
	}
	return pngWH{cfg.Width, cfg.Height}, nil
}

var reSVGW = regexp.MustCompile(`width="([0-9.]+)"`)
var reSVGH = regexp.MustCompile(`height="([0-9.]+)"`)

func svgDims(b []byte) (pngWH, error) {
	wm := reSVGW.FindSubmatch(b)
	hm := reSVGH.FindSubmatch(b)
	if wm == nil || hm == nil {
		return pngWH{}, os.ErrInvalid
	}
	wf, _ := strconv.ParseFloat(string(wm[1]), 64)
	hf, _ := strconv.ParseFloat(string(hm[1]), 64)
	return pngWH{int(wf + 0.5), int(hf + 0.5)}, nil
}
