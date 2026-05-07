package render

import (
	"testing"

	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
)

func TestPNGSimpleA(t *testing.T) {
	body, err := parser.Parse("a")
	if err != nil {
		t.Fatal(err)
	}
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	pngBytes, err := PNG(dl)
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

func TestPNGFraction(t *testing.T) {
	body, err := parser.Parse(`\frac{1}{2}`)
	if err != nil {
		t.Fatal(err)
	}
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	pngBytes, err := PNG(dl, WithScale(2.0))
	if err != nil {
		t.Fatal(err)
	}
	if len(pngBytes) < 200 {
		t.Errorf("expected non-trivial png, got %d bytes", len(pngBytes))
	}
}
