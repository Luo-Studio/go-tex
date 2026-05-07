package render

import (
	"testing"

	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
	"github.com/luo-studio/go-tex/tex/svg"
)

func TestPNGSimpleA(t *testing.T) {
	body, err := parser.Parse("a")
	if err != nil {
		t.Fatal(err)
	}
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	out := svg.Render(dl, svg.DefaultOptions())
	pngBytes, err := PNG(out, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pngBytes) < 100 {
		t.Errorf("expected non-trivial png, got %d bytes", len(pngBytes))
	}
	// PNG signature: 0x89 P N G
	if pngBytes[0] != 0x89 || string(pngBytes[1:4]) != "PNG" {
		t.Error("output is not a valid PNG")
	}
}
