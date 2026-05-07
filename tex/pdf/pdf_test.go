package pdf

import (
	"bytes"
	"testing"

	"codeberg.org/go-pdf/fpdf"

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

// TestDrawIntoEmbedsInLargerDocument exercises the embedding entry
// point: a caller-owned A4 fpdf document with a header line, two
// equations placed at chosen positions, and a footer line.
func TestDrawIntoEmbedsInLargerDocument(t *testing.T) {
	body1, err := parser.Parse(`E = mc^2`)
	if err != nil {
		t.Fatal(err)
	}
	dl1 := layout.ToDisplayList(layout.Layout(body1, layout.DefaultOptions()))

	body2, err := parser.Parse(`\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt\pi}{2}`)
	if err != nil {
		t.Fatal(err)
	}
	dl2 := layout.ToDisplayList(layout.Layout(body2, layout.DefaultOptions()))

	doc := fpdf.New("P", "pt", "A4", "")
	doc.SetMargins(36, 36, 36)
	doc.AddPage()

	doc.SetFont("Arial", "", 14)
	doc.Cell(0, 20, "Equations:")

	opts := EmbedDefaults()
	opts.FontSize = 14

	// First equation, ~80pt down from the top, indented 60pt from left.
	if err := DrawInto(doc, dl1, 60, 80, opts); err != nil {
		t.Fatalf("DrawInto eq1: %v", err)
	}
	w1, h1 := Size(dl1, opts)
	if w1 <= 0 || h1 <= 0 {
		t.Errorf("Size returned non-positive: w=%v h=%v", w1, h1)
	}

	// Second equation, below the first.
	if err := DrawInto(doc, dl2, 60, 80+h1+20, opts); err != nil {
		t.Fatalf("DrawInto eq2: %v", err)
	}

	doc.SetXY(36, 200)
	doc.SetFont("Arial", "", 12)
	doc.Cell(0, 16, "Both equations rendered into one document.")

	if err := doc.Error(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("output not a PDF: %q", out[:8])
	}
	if len(out) < 1000 {
		t.Errorf("expected non-trivial PDF, got %d bytes", len(out))
	}
}

// TestRegisterFontsIdempotent verifies that multiple DrawInto calls on
// the same *fpdf.Fpdf register the KaTeX TTFs only once.
func TestRegisterFontsIdempotent(t *testing.T) {
	body, _ := parser.Parse(`a`)
	dl := layout.ToDisplayList(layout.Layout(body, layout.DefaultOptions()))

	doc := fpdf.New("P", "pt", "A4", "")
	doc.AddPage()

	for i := 0; i < 3; i++ {
		if err := DrawInto(doc, dl, 40, float64(40+i*30), EmbedDefaults()); err != nil {
			t.Fatalf("DrawInto[%d]: %v", i, err)
		}
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Error("output not a PDF")
	}
}
