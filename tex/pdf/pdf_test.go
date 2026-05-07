package pdf

import (
	"bytes"
	"testing"

	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
)

func TestRenderSimpleA(t *testing.T) {
	body, err := parser.Parse("a")
	if err != nil {
		t.Fatal(err)
	}
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	out, err := Render(dl, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output not a PDF: %q", out[:8])
	}
	if len(out) < 1000 {
		t.Errorf("expected non-trivial PDF, got %d bytes", len(out))
	}
}

func TestRenderFraction(t *testing.T) {
	body, err := parser.Parse(`\frac{a}{b}`)
	if err != nil {
		t.Fatal(err)
	}
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	out, err := Render(dl, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output not a PDF")
	}
}

func TestRenderIntegralWithSubSup(t *testing.T) {
	body, err := parser.Parse(`\int_{0}^{\infty} e^{-x^2} dx = \frac{\sqrt{\pi}}{2}`)
	if err != nil {
		t.Fatal(err)
	}
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	out, err := Render(dl, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output not a PDF")
	}
}

func TestRenderCustomOptions(t *testing.T) {
	body, _ := parser.Parse(`x^2`)
	box := layout.Layout(body, layout.DefaultOptions())
	dl := layout.ToDisplayList(box)
	out, err := Render(dl, Options{FontSize: 24, Padding: 8, StrokeWidth: 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output not a PDF")
	}
}
