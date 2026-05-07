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
	case Fraction:
		// Numerator: top-aligned within numer_height area.
		// Position numer so its baseline sits at b.height - numer_shift
		// (i.e. numer_shift above the axis).
		numerW := c.Numer.Width * c.NumerScale
		numerCenterX := x + (b.Width-numerW)/2
		numerY := y + b.Height - c.NumerShift - c.Numer.Height*c.NumerScale
		emit(c.Numer, dl, numerCenterX, numerY, scale*c.NumerScale)
		// Denominator: positioned with baseline at axis - denom_shift below.
		denomW := c.Denom.Width * c.DenomScale
		denomCenterX := x + (b.Width-denomW)/2
		denomY := y + b.Height + c.DenomShift - c.Denom.Height*c.DenomScale
		emit(c.Denom, dl, denomCenterX, denomY, scale*c.DenomScale)
		// Bar at the axis (b.height - axis_height).
		if c.BarThickness > 0 {
			// axis_height ~ 0.25em at textstyle. We don't have direct access
			// here; the bar y comes from b.Height - axis_height (parent's
			// metrics axis was set at layout time). Use 0.25 fallback.
			axis := 0.25
			barY := y + b.Height - axis
			dl.Items = append(dl.Items, Line{
				X:         x,
				Y:         barY,
				Width:     b.Width,
				Thickness: c.BarThickness,
				Color:     b.Color,
			})
		}
	case SupSub:
		// Base sits at the same baseline as the box.
		emit(c.Base, dl, x, y+(b.Height-c.Base.Height), scale)
		baseW := c.Base.Width
		if c.Sup != nil {
			supW := c.Sup.Width * c.SupScale
			_ = supW
			emit(c.Sup, dl, x+baseW, y+(b.Height-c.SupShift-c.Sup.Height*c.SupScale), scale*c.SupScale)
		}
		if c.Sub != nil {
			emit(c.Sub, dl, x+baseW, y+(b.Height+c.SubShift-c.Sub.Height*c.SubScale), scale*c.SubScale)
		}
	case Overline:
		emit(c.Body, dl, x, y+4*c.RuleThickness, scale)
		dl.Items = append(dl.Items, Line{X: x, Y: y + 2*c.RuleThickness, Width: b.Width, Thickness: c.RuleThickness, Color: b.Color})
	case Underline:
		emit(c.Body, dl, x, y, scale)
		dl.Items = append(dl.Items, Line{X: x, Y: y + b.Height + b.Depth - 2*c.RuleThickness, Width: b.Width, Thickness: c.RuleThickness, Color: b.Color})
	case LeftRight:
		emit(c.Inner, dl, x, y, scale)
	case Radical:
		// Surd glyph would go on the left; we don't draw it yet, but the
		// body needs to be placed right of the surd allowance (~0.5em).
		emit(c.Body, dl, x+0.5, y, scale)
	case Empty, Kern:
		// no items
	}
}
