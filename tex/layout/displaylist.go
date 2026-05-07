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
//
// Convention: (x, y) is the top-left of b in PARENT em units. The box's
// own Width/Height/Depth are in CHILD em (the box's design size).
// scale converts child em to parent em — child * scale = parent.
//
// So a glyph's baseline in parent em is `y + b.Height*scale`.
func emit(b *Box, dl *DisplayList, x, y, scale float64) {
	switch c := b.Content.(type) {
	case Glyph:
		dl.Items = append(dl.Items, GlyphPath{
			X:        x,
			Y:        y + b.Height*scale,
			Scale:    scale,
			Font:     c.FontID,
			CharCode: c.CharCode,
			Color:    b.Color,
		})
	case Rule:
		dl.Items = append(dl.Items, Rect{
			X: x, Y: y, Width: b.Width * scale, Height: c.Thickness * scale, Color: b.Color,
		})
	case HBox:
		cx := x
		for _, ch := range c.Children {
			emit(ch, dl, cx, y+(b.Height-ch.Height)*scale, scale)
			cx += ch.Width * scale
		}
	case VBox:
		cy := y
		for _, ch := range c.Children {
			switch k := ch.Kind.(type) {
			case VBoxBox:
				emit(k.Box, dl, x, cy+ch.Shift*scale, scale)
				cy += k.Box.TotalHeight() * scale
			case VBoxKern:
				cy += k.Em * scale
			}
		}
	case Fraction:
		// Bar / shifts are in parent em (computed from parent metrics).
		numerScale := scale * c.NumerScale
		denomScale := scale * c.DenomScale
		numerWParent := c.Numer.Width * numerScale
		denomWParent := c.Denom.Width * denomScale
		numerCenterX := x + (b.Width*scale-numerWParent)/2
		denomCenterX := x + (b.Width*scale-denomWParent)/2
		numerY := y + (b.Height-c.NumerShift)*scale - c.Numer.Height*numerScale
		denomY := y + (b.Height+c.DenomShift)*scale - c.Denom.Height*denomScale
		emit(c.Numer, dl, numerCenterX, numerY, numerScale)
		emit(c.Denom, dl, denomCenterX, denomY, denomScale)
		if c.BarThickness > 0 {
			axis := 0.25 // approximate axis_height
			barY := y + (b.Height-axis)*scale
			dl.Items = append(dl.Items, Line{
				X:         x,
				Y:         barY,
				Width:     b.Width * scale,
				Thickness: c.BarThickness * scale,
				Color:     b.Color,
			})
		}
	case SupSub:
		// Base baseline at y + b.Height*scale.
		emit(c.Base, dl, x, y+(b.Height-c.Base.Height)*scale, scale)
		baseW := c.Base.Width * scale
		if c.Sup != nil {
			supScale := scale * c.SupScale
			supTop := y + (b.Height-c.SupShift)*scale - c.Sup.Height*supScale
			emit(c.Sup, dl, x+baseW, supTop, supScale)
		}
		if c.Sub != nil {
			subScale := scale * c.SubScale
			subTop := y + (b.Height+c.SubShift)*scale - c.Sub.Height*subScale
			emit(c.Sub, dl, x+baseW, subTop, subScale)
		}
	case Overline:
		emit(c.Body, dl, x, y+4*c.RuleThickness*scale, scale)
		dl.Items = append(dl.Items, Line{
			X: x, Y: y + 2*c.RuleThickness*scale,
			Width: b.Width * scale, Thickness: c.RuleThickness * scale, Color: b.Color,
		})
	case Underline:
		emit(c.Body, dl, x, y, scale)
		dl.Items = append(dl.Items, Line{
			X: x, Y: y + (b.Height+b.Depth-2*c.RuleThickness)*scale,
			Width: b.Width * scale, Thickness: c.RuleThickness * scale, Color: b.Color,
		})
	case LeftRight:
		emit(c.Inner, dl, x, y, scale)
	case Accent:
		baseTop := y + (b.Height-c.Base.Height)*scale
		emit(c.Base, dl, x, baseTop, scale)
		baseBaselineY := baseTop + c.Base.Height*scale
		// Upstream to_display.rs:
		//   accent_y = base_baseline - clearance + (accent.h - min(0.35, accent.h))
		accentY := baseBaselineY + (-c.Clearance+c.Correction)*scale
		accentX := x + (c.Base.Width-c.AccentBox.Width)*scale/2 + c.Skew*scale
		dl.Items = append(dl.Items, GlyphPath{
			X:        accentX,
			Y:        accentY,
			Scale:    scale,
			Font:     fontIDOfGlyph(c.AccentBox),
			CharCode: charCodeOfGlyph(c.AccentBox),
			Color:    b.Color,
		})
	case Radical:
		// Surd glyph at left, baseline at body baseline.
		baselineY := y + b.Height*scale
		dl.Items = append(dl.Items, GlyphPath{
			X:        x,
			Y:        baselineY,
			Scale:    scale,
			Font:     "Main-Regular",
			CharCode: 0x221A,
			Color:    b.Color,
		})
		// Body shifted right by the surd width (IndexOffset).
		bodyX := x + c.IndexOffset*scale
		bodyY := y + 4*c.RuleThickness*scale
		emit(c.Body, dl, bodyX, bodyY, scale)
		// Overline rule above the body.
		dl.Items = append(dl.Items, Line{
			X:         bodyX,
			Y:         y + 2*c.RuleThickness*scale,
			Width:     c.Body.Width * scale,
			Thickness: c.RuleThickness * scale,
			Color:     b.Color,
		})
	case Array:
		// Top of array content (y) corresponds to box top (y + 0).
		// Each row r has baseline at sum_{i<r}(row_h[i]+row_d[i]) + row_h[r]
		// from the box top.
		cy := y
		for ri, row := range c.Cells {
			rowH := c.RowHeights[ri]
			rowD := c.RowDepths[ri]
			baselineY := cy + rowH*scale
			cx := x + c.ContentXOffset*scale
			for ci, cell := range row {
				if ci > 0 {
					cx += c.ColGap * scale
				}
				colW := c.ColWidths[ci]
				var cellX float64
				switch c.ColAligns[ci] {
				case 'l':
					cellX = cx
				case 'r':
					cellX = cx + (colW-cell.Width)*scale
				default:
					cellX = cx + (colW-cell.Width)*scale/2
				}
				cellTopY := baselineY - cell.Height*scale
				emit(cell, dl, cellX, cellTopY, scale)
				cx += colW * scale
			}
			cy += (rowH + rowD) * scale
		}
	case Empty, Kern:
		// no items
	}
}

func fontIDOfGlyph(b *Box) string {
	if g, ok := b.Content.(Glyph); ok {
		return g.FontID
	}
	return "Main-Regular"
}

func charCodeOfGlyph(b *Box) rune {
	if g, ok := b.Content.(Glyph); ok {
		return g.CharCode
	}
	return 0
}
