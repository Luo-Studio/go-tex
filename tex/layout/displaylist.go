package layout

import (
	"github.com/luo-studio/go-tex/tex/path"
)

// DisplayList is the flat list of absolute drawing commands the layout
// engine emits. Renderers consume this directly.
type DisplayList struct {
	Items  []DisplayItem `json:"items"`
	Width  float64       `json:"width"`
	Height float64       `json:"height"`
	Depth  float64       `json:"depth"`
}

// DisplayItem is one drawing primitive.
type DisplayItem interface{ displayItemMarker() }

func (GlyphPath) displayItemMarker() {}
func (Line) displayItemMarker()      {}
func (Rect) displayItemMarker()      {}
func (PathItem) displayItemMarker()  {}

// GlyphPath draws a glyph outline at (x, y).
type GlyphPath struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Scale    float64 `json:"scale"`
	Font     string  `json:"font"`
	CharCode rune    `json:"char_code"`
	Color    Color   `json:"color"`
}

// Line draws a horizontal line (fraction bars, rules, ...).
type Line struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Thickness float64 `json:"thickness"`
	Color     Color   `json:"color"`
	Dashed    bool    `json:"dashed,omitempty"`
}

// Rect draws a filled rectangle.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Color  Color   `json:"color"`
}

// PathItem is an arbitrary SVG-style path (radicals, delimiters, …).
type PathItem struct {
	X        float64        `json:"x"`
	Y        float64        `json:"y"`
	Commands []path.Command `json:"commands"`
	Fill     bool           `json:"fill"`
	Color    Color          `json:"color"`
}

// ToDisplayList walks a Box and emits absolute-position drawing commands.
//
// This is a starting port; many BoxContent variants currently emit no
// items. It does count item totals so the parity harness can score.
func ToDisplayList(b *Box) DisplayList {
	dl := DisplayList{Width: b.Width, Height: b.Height, Depth: b.Depth}
	emit(b, &dl, 0, 0, 1.0)
	return dl
}

// emit recursively walks b, appending DisplayItems with absolute positions.
// (x, y) is the top-left of b, scale is the cumulative scale factor.
func emit(b *Box, dl *DisplayList, x, y, scale float64) {
	switch c := b.Content.(type) {
	case Glyph:
		dl.Items = append(dl.Items, GlyphPath{
			X:        x,
			Y:        y + b.Height,
			Scale:    scale,
			Font:     c.FontID,
			CharCode: c.CharCode,
			Color:    b.Color,
		})
	case Rule:
		dl.Items = append(dl.Items, Rect{
			X: x, Y: y, Width: b.Width, Height: c.Thickness, Color: b.Color,
		})
	case HBox:
		cx := x
		for _, ch := range c.Children {
			emit(ch, dl, cx, y+(b.Height-ch.Height), scale)
			cx += ch.Width
		}
	case VBox:
		cy := y
		for _, ch := range c.Children {
			switch k := ch.Kind.(type) {
			case VBoxBox:
				emit(k.Box, dl, x, cy+ch.Shift, scale)
				cy += k.Box.TotalHeight()
			case VBoxKern:
				cy += k.Em
			}
		}
	case Empty, Kern:
		// no items
	}
}
