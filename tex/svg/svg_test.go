package svg

import (
	"strings"
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
	out := Render(dl, DefaultOptions())
	if !strings.HasPrefix(out, `<svg xmlns="http://www.w3.org/2000/svg"`) {
		t.Errorf("missing xmlns: %s", out)
	}
	if !strings.Contains(out, "KaTeX_Math") {
		t.Errorf("expected KaTeX_Math font: %s", out)
	}
	if !strings.Contains(out, ">a<") {
		t.Errorf("expected glyph 'a': %s", out)
	}
}

func TestRenderEmpty(t *testing.T) {
	dl := layout.DisplayList{}
	out := Render(dl, DefaultOptions())
	if !strings.HasPrefix(out, `<svg `) {
		t.Errorf("expected svg prefix: %s", out)
	}
}
