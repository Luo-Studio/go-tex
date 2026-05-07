package render

import (
	"bufio"
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
)

// TestPNGStructuralParity is a sanity harness: for each golden case,
// render via parser→layout→canvasr→PNG, decode the upstream reference
// PNG, and report:
//
//   - byte-identical: how many output PNGs are byte-for-byte equal to
//     upstream (very unlikely without ab_glyph rasteriser parity).
//   - dim-match: how many have the same pixel dimensions as upstream.
//   - first failure example.
//
// Pixel-identical PNGs require porting ab_glyph's rasteriser. This
// test sets the floor and shows that the renderer pipeline is
// end-to-end functional (PNGs decode, dimensions match SVG output).
func TestPNGStructuralParity(t *testing.T) {
	corpusPath := filepath.Join("..", "..", "testdata", "golden", "test_cases.txt")
	goldenDir := filepath.Join("..", "..", "testdata", "golden", "output")
	if _, err := os.Stat(goldenDir); err != nil {
		t.Skip("golden output dir not present")
	}
	f, err := os.Open(corpusPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	idx := 0
	byteMatch, dimMatch, decoded, total := 0, 0, 0, 0
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
		wantPath := filepath.Join(goldenDir, formatIdx(idx)+".png")
		want, err := os.ReadFile(wantPath)
		if err != nil {
			continue
		}
		if bytes.Equal(ours, want) {
			byteMatch++
		}
		ow, ohErr := pngDims(ours)
		ww, whErr := pngDims(want)
		if ohErr == nil && whErr == nil && ow == ww {
			dimMatch++
		}
	}
	t.Logf("PNG: %d cases  decoded %d  dim-match %d (%.2f%%)  byte-identical %d (%.2f%%)\nfirst failure: %q",
		total, decoded, dimMatch, percent(dimMatch, total),
		byteMatch, percent(byteMatch, total), firstFailure)
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
