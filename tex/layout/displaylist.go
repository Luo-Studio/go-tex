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
		// Box height = body.height + 3*rt. Body sits at the bottom of the
		// box, with rule centered 2.5*rt above body's top — i.e. 0.5*rt
		// below the box top.
		emit(c.Body, dl, x, y+3*c.RuleThickness*scale, scale)
		dl.Items = append(dl.Items, Line{
			X: x, Y: y + 0.5*c.RuleThickness*scale,
			Width: b.Width * scale, Thickness: c.RuleThickness * scale, Color: b.Color,
		})
	case Underline:
		// Box depth = body.depth + 3*rt. Body sits at the top, with rule
		// centered 2.5*rt below body's bottom (baseline + body.depth).
		emit(c.Body, dl, x, y, scale)
		dl.Items = append(dl.Items, Line{
			X: x, Y: y + (c.Body.Height+c.Body.Depth+2.5*c.RuleThickness)*scale,
			Width: b.Width * scale, Thickness: c.RuleThickness * scale, Color: b.Color,
		})
	case LeftRight:
		// left, inner, right concatenated horizontally. The delimiters'
		// own y-extent already centres them on the math axis when their
		// h+d > inner h+d; for matching upstream baseline we put each
		// child at top y + (b.Height - child.Height)*scale.
		cx := x
		emit(c.Left, dl, cx, y+(b.Height-c.Left.Height)*scale, scale)
		cx += c.Left.Width * scale
		emit(c.Inner, dl, cx, y+(b.Height-c.Inner.Height)*scale, scale)
		cx += c.Inner.Width * scale
		emit(c.Right, dl, cx, y+(b.Height-c.Right.Height)*scale, scale)
	case Accent:
		// Box width may be larger than base.Width (min 0.5em). Centre
		// base within the box, then place accent in the box's frame.
		baseX := x + (b.Width-c.Base.Width)*scale/2
		baseTop := y + (b.Height-c.Base.Height)*scale
		emit(c.Base, dl, baseX, baseTop, scale)
		baseBaselineY := baseTop + c.Base.Height*scale
		// Upstream to_display.rs:
		//   accent_y = base_baseline - clearance + (accent.h - min(0.35, accent.h))
		accentY := baseBaselineY + (-c.Clearance+c.Correction)*scale
		accentX := x + (b.Width-c.AccentBox.Width)*scale/2 + c.Skew*scale
		dl.Items = append(dl.Items, GlyphPath{
			X:        accentX,
			Y:        accentY,
			Scale:    scale,
			Font:     fontIDOfGlyph(c.AccentBox),
			CharCode: charCodeOfGlyph(c.AccentBox),
			Color:    b.Color,
		})
	case Radical:
		// Surd left edge: shift right by max(0, index.Width*indexScale - surdW + 0.3)
		// when index is present.
		surdX := x
		bodyX := x + c.IndexOffset*scale
		if c.Index != nil {
			extra := c.Index.Width*c.IndexScale - c.IndexOffset + 0.3
			if extra > 0 {
				surdX += extra * scale
				bodyX += extra * scale
			}
			// Index above-left of surd. Approximate position.
			indexX := x + 0.3*scale
			indexY := y + c.RuleThickness*scale + c.Index.Height*scale*c.IndexScale
			emit(c.Index, dl, indexX, indexY-c.Index.Height*scale*c.IndexScale, scale*c.IndexScale)
		}
		baselineY := y + b.Height*scale
		dl.Items = append(dl.Items, GlyphPath{
			X:        surdX,
			Y:        baselineY,
			Scale:    scale,
			Font:     "Main-Regular",
			CharCode: 0x221A,
			Color:    b.Color,
		})
		bodyY := y + 4*c.RuleThickness*scale
		emit(c.Body, dl, bodyX, bodyY, scale)
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
	case Scaled:
		// Multiply current scale by child scale; child dims are in
		// child em.
		emit(c.Body, dl, x, y, scale*c.ChildScale)
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
