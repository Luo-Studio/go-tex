package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
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
	expandVisualBounds(&dl)
	return dl
}

// expandVisualBounds mirrors upstream to_display.rs: when display items
// extend outside the nominal box (e.g. accent ring wider than base, smash
// zeroing height/depth, mathllap zero width), shift items right/down and
// grow width/depth so the rasterizer's pixmap covers the ink.
func expandVisualBounds(dl *DisplayList) {
	if len(dl.Items) == 0 {
		return
	}
	minX, maxX, minY, maxY := computeVisualBounds(dl.Items)
	totalH := dl.Height + dl.Depth

	// Vertical: when nominal total height is near-zero (\smash), grow.
	if totalH < 0.01 {
		if minY < -0.001 {
			extra := -minY
			dl.Height += extra
			for i := range dl.Items {
				shiftItemY(&dl.Items[i], extra)
			}
		}
		nominalBottom := dl.Height + dl.Depth
		shiftedMaxY := maxY
		if minY < -0.001 {
			shiftedMaxY = maxY - minY
		}
		if shiftedMaxY > nominalBottom+0.001 {
			dl.Depth = shiftedMaxY - dl.Height
		}
	}

	// Horizontal: zero-width box (mathllap) gets full expansion; otherwise
	// shift+grow when ink extends left of nominal x=0.
	if dl.Width < 0.01 {
		if minX < -0.001 {
			extra := -minX
			dl.Width += extra
			for i := range dl.Items {
				shiftItemX(&dl.Items[i], extra)
			}
		}
		shiftedMaxX := maxX
		if minX < -0.001 {
			shiftedMaxX = maxX - minX
		}
		if shiftedMaxX > dl.Width+0.001 {
			dl.Width = shiftedMaxX
		}
	} else if minX < -0.001 {
		extra := -minX
		w := dl.Width + extra
		if maxX+extra > w {
			w = maxX + extra
		}
		dl.Width = w
		for i := range dl.Items {
			shiftItemX(&dl.Items[i], extra)
		}
	}

	// Vertical (full): when total_h >= 0.01 and ink extends above y=0,
	// shift down and grow depth as needed.
	if totalH >= 0.01 && minY < -0.001 {
		extra := -minY
		dl.Height += extra
		for i := range dl.Items {
			shiftItemY(&dl.Items[i], extra)
		}
		newBottom := dl.Height + dl.Depth
		adjustedMaxY := maxY + extra
		if adjustedMaxY > newBottom+0.001 {
			dl.Depth = adjustedMaxY - dl.Height
		}
	}

	// Expand depth when ink extends below nominal bottom.
	if totalH >= 0.01 && minY >= -0.001 && maxY > dl.Height+dl.Depth+0.001 {
		dl.Depth = maxY - dl.Height
	}
}

func computeVisualBounds(items []DisplayItem) (minX, maxX, minY, maxY float64) {
	minX, minY = 1e18, 1e18
	maxX, maxY = -1e18, -1e18
	for _, it := range items {
		switch v := it.(type) {
		case GlyphPath:
			cm, ok := fontmetrics.LookupWithFallback(v.Font, v.CharCode)
			w, h, d := 0.0, 0.0, 0.0
			if ok {
				w, h, d = cm.Width, cm.Height, cm.Depth
			}
			if v.X < minX {
				minX = v.X
			}
			if v.X+w*v.Scale > maxX {
				maxX = v.X + w*v.Scale
			}
			if v.Y-h*v.Scale < minY {
				minY = v.Y - h*v.Scale
			}
			if v.Y+d*v.Scale > maxY {
				maxY = v.Y + d*v.Scale
			}
		case Line:
			if v.X < minX {
				minX = v.X
			}
			if v.X+v.Width > maxX {
				maxX = v.X + v.Width
			}
			if v.Y-v.Thickness/2 < minY {
				minY = v.Y - v.Thickness/2
			}
			if v.Y+v.Thickness/2 > maxY {
				maxY = v.Y + v.Thickness/2
			}
		case Rect:
			if v.X < minX {
				minX = v.X
			}
			if v.X+v.Width > maxX {
				maxX = v.X + v.Width
			}
			if v.Y < minY {
				minY = v.Y
			}
			if v.Y+v.Height > maxY {
				maxY = v.Y + v.Height
			}
		case PathItem:
			const maxEm = 50.0
			consider := func(cx, cy float64) {
				if cx < -maxEm || cx > maxEm || cy < -maxEm || cy > maxEm {
					return
				}
				ax, ay := v.X+cx, v.Y+cy
				if ax < minX {
					minX = ax
				}
				if ax > maxX {
					maxX = ax
				}
				if ay < minY {
					minY = ay
				}
				if ay > maxY {
					maxY = ay
				}
			}
			for _, cmd := range v.Commands {
				switch cmd.Kind {
				case path.KindMoveTo, path.KindLineTo:
					consider(cmd.X, cmd.Y)
				case path.KindCubicTo:
					consider(cmd.X1, cmd.Y1)
					consider(cmd.X2, cmd.Y2)
					consider(cmd.X, cmd.Y)
				case path.KindQuadTo:
					consider(cmd.X1, cmd.Y1)
					consider(cmd.X, cmd.Y)
				}
			}
		}
	}
	return
}

func shiftItemX(it *DisplayItem, dx float64) {
	switch v := (*it).(type) {
	case GlyphPath:
		v.X += dx
		*it = v
	case Line:
		v.X += dx
		*it = v
	case Rect:
		v.X += dx
		*it = v
	case PathItem:
		v.X += dx
		*it = v
	}
}

func shiftItemY(it *DisplayItem, dy float64) {
	switch v := (*it).(type) {
	case GlyphPath:
		v.Y += dy
		*it = v
	case Line:
		v.Y += dy
		*it = v
	case Rect:
		v.Y += dy
		*it = v
	case PathItem:
		v.Y += dy
		*it = v
	}
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
		// Rule's baseline is at y + b.Height*scale. Raise = distance
		// from baseline to bottom edge of the rule. Line CENTER is then
		// at baseline - (raise + thickness/2) * scale. Mirrors upstream
		// emit_box's Rule branch.
		baselineY := y + b.Height*scale
		lineCenterY := baselineY - (c.Raise+c.Thickness/2)*scale
		// Line draws from yc - t/2 to yc + t/2; we use Rect for parity
		// with upstream which emits a filled rect for solid rules.
		topY := lineCenterY - c.Thickness*scale/2
		dl.Items = append(dl.Items, Rect{
			X: x, Y: topY, Width: b.Width * scale, Height: c.Thickness * scale, Color: b.Color,
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
	case OpLimits:
		// Mirror upstream emit. baselineY in our top-left coords:
		baselineY := y + b.Height*scale
		boxW := b.Width * scale
		// Base centered horizontally; baseline shifted by base_shift.
		baseX := x + (boxW-c.Base.Width*scale)/2
		baseTop := baselineY + c.BaseShift*scale - c.Base.Height*scale
		emit(c.Base, dl, baseX, baseTop, scale)
		if c.Sup != nil {
			childScale := scale * c.SupScale
			supX := x + (boxW-c.Sup.Width*childScale)/2 + c.Slant*scale/2
			supBaseline := baselineY - (c.Base.Height-c.BaseShift)*scale - c.SupKern*scale - c.Sup.Depth*childScale
			emit(c.Sup, dl, supX, supBaseline-c.Sup.Height*childScale, childScale)
		}
		if c.Sub != nil {
			childScale := scale * c.SubScale
			subX := x + (boxW-c.Sub.Width*childScale)/2 - c.Slant*scale/2
			subBaseline := baselineY + (c.Base.Depth+c.BaseShift)*scale + c.SubKern*scale + c.Sub.Height*childScale
			emit(c.Sub, dl, subX, subBaseline-c.Sub.Height*childScale, childScale)
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
			emit(c.Sub, dl, x+baseW+c.SubHKern*scale, subTop, subScale)
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
		// Mirror upstream: surd starts at x + index_offset, body
		// follows at surd_x + radical_width (where radical_width =
		// lbox.width - index_offset - body.width = surd glyph width).
		baselineY := y + b.Height*scale
		radicalWidth := (b.Width - c.IndexOffset - c.Body.Width) * scale
		surdX := x + c.IndexOffset*scale
		// Pick surd glyph from one of Main / Size1-4 by InnerHeight.
		surdFont := surdFontForInnerHeight(c.InnerHeight)
		gh := 0.0
		if cm, ok := fontmetrics.Lookup(surdFont, 0x221A); ok {
			gh = cm.Height
		} else {
			gh = b.Height - c.RuleThickness
		}
		surdShift := b.Height - c.RuleThickness - gh
		surdY := baselineY - surdShift*scale
		// Index: baseline at y - to_shift where to_shift = 0.6*(h-d).
		if c.Index != nil {
			toShift := 0.6 * (b.Height - b.Depth)
			indexBaselineY := baselineY - toShift*scale
			childScale := scale * c.IndexScale
			indexX := surdX + (5.0/18.0)*scale
			emit(c.Index, dl, indexX, indexBaselineY-c.Index.Height*childScale, childScale)
		}
		dl.Items = append(dl.Items, GlyphPath{
			X:        surdX,
			Y:        surdY,
			Scale:    scale,
			Font:     surdFont,
			CharCode: 0x221A,
			Color:    b.Color,
		})
		// Vinculum (rule) on top of body, level with surd's top bar.
		lineCenterY := surdY - gh*scale + (c.RuleThickness*scale)/2
		dl.Items = append(dl.Items, Line{
			X:         surdX + radicalWidth,
			Y:         lineCenterY,
			Width:     c.Body.Width * scale,
			Thickness: c.RuleThickness * scale,
			Color:     b.Color,
		})
		// Body emitted at surd_x + radical_width.
		bodyTop := baselineY - c.Body.Height*scale
		emit(c.Body, dl, surdX+radicalWidth, bodyTop, scale)
	case Array:
		// Top of array content (y) corresponds to box top.
		yTop := y
		arrayTotalHeight := (b.Height + b.Depth) * scale
		lineThickness := c.RuleThickness * scale
		if c.RuleThickness == 0 {
			// Default thickness from KaTeX: 0.04em.
			lineThickness = 0.04 * scale
		}
		// Boundary y positions (top of array, then after each row).
		boundaryYs := make([]float64, 0, len(c.Cells)+1)
		curB := yTop
		boundaryYs = append(boundaryYs, curB)
		for ri := range c.Cells {
			curB += (c.RowHeights[ri] + c.RowDepths[ri]) * scale
			boundaryYs = append(boundaryYs, curB)
		}
		ruleStep := lineThickness + c.DoubleRuleSep*scale
		// Horizontal rules.
		for r, hlines := range c.HLinesBeforeRow {
			if r >= len(boundaryYs) {
				break
			}
			n := len(hlines)
			var startY float64
			if r == 0 {
				startY = boundaryYs[0]
			} else {
				startY = boundaryYs[r] - float64(n-1)*ruleStep
			}
			width := c.ArrayInnerWidth * scale
			if width == 0 {
				width = b.Width * scale
			}
			for i, dashed := range hlines {
				dl.Items = append(dl.Items, Line{
					X:         x,
					Y:         startY + float64(i)*ruleStep,
					Width:     width,
					Thickness: lineThickness,
					Color:     b.Color,
					Dashed:    dashed,
				})
			}
		}
		// Vertical column separators.
		colGapHalf := c.ColGap / 2
		for i, sep := range c.ColSeparators {
			if sep == nil {
				continue
			}
			isDashed := *sep
			prefixW := 0.0
			for j := 0; j < i && j < len(c.ColWidths); j++ {
				prefixW += c.ColWidths[j]
			}
			localX := c.ContentXOffset - colGapHalf + prefixW + c.ColGap*float64(i)
			absX := x + localX*scale - lineThickness/2
			if isDashed {
				// Dashed: draw segments.
				dash := 4 * lineThickness
				period := 2 * dash
				curY := yTop
				for curY < yTop+arrayTotalHeight {
					segH := dash
					if curY+segH > yTop+arrayTotalHeight {
						segH = yTop + arrayTotalHeight - curY
					}
					dl.Items = append(dl.Items, Rect{
						X: absX, Y: curY, Width: lineThickness, Height: segH, Color: b.Color,
					})
					curY += period
				}
			} else {
				dl.Items = append(dl.Items, Rect{
					X: absX, Y: yTop, Width: lineThickness, Height: arrayTotalHeight, Color: b.Color,
				})
			}
		}
		// Cells.
		cy := yTop
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
			// Equation tag (right-aligned in tag column).
			if c.TagColWidth > 0 && ri < len(c.RowTags) {
				if tb := c.RowTags[ri]; tb != nil {
					tagStartEm := c.ArrayInnerWidth - c.ContentXOffset + c.TagGapEm
					tagX := x + tagStartEm*scale + (c.TagColWidth-tb.Width)*scale
					tagTop := baselineY - tb.Height*scale
					emit(tb, dl, tagX, tagTop, scale)
				}
			}
			cy += (rowH + rowD) * scale
		}
	case Scaled:
		// Multiply current scale by child scale; child dims are in
		// child em.
		emit(c.Body, dl, x, y, scale*c.ChildScale)
	case Framed:
		// Bg fill (whole box), then 4 border rects (top/bottom/left/right),
		// then body emitted at outer-baseline shifted right by padding.
		outerW := b.Width * scale
		outerHD := (b.Height + b.Depth) * scale
		if c.BgColor != nil {
			dl.Items = append(dl.Items, Rect{
				X: x, Y: y, Width: outerW, Height: outerHD, Color: *c.BgColor,
			})
		}
		if c.HasBorder {
			bt := c.BorderThickness * scale
			// Top
			dl.Items = append(dl.Items, Rect{X: x, Y: y, Width: outerW, Height: bt, Color: c.BorderColor})
			// Bottom
			dl.Items = append(dl.Items, Rect{X: x, Y: y + outerHD - bt, Width: outerW, Height: bt, Color: c.BorderColor})
			// Left
			dl.Items = append(dl.Items, Rect{X: x, Y: y, Width: bt, Height: outerHD, Color: c.BorderColor})
			// Right
			dl.Items = append(dl.Items, Rect{X: x + outerW - bt, Y: y, Width: bt, Height: outerHD, Color: c.BorderColor})
		}
		// Body shift: upstream always uses padding + border_thickness
		// (even on no-border \colorbox — matches the inner_offset
		// term in to_display.rs).
		bodyOffsetEm := c.Padding + c.BorderThickness
		// In our top-left convention, the body's top sits at outer.top + outer_pad*scale
		// so its baseline lines up with the outer baseline.
		outerPad := c.Padding
		if c.HasBorder {
			outerPad += c.BorderThickness
		}
		bodyTopY := y + outerPad*scale
		emit(c.Body, dl, x+bodyOffsetEm*scale, bodyTopY, scale)
	case RaiseBox:
		// Positive Shift moves the body up. The wrapping box has
		// Height = body.Height + Shift; emitting the body with its top
		// at y places its baseline at y + body.Height, which equals
		// (y + Box.Height - Shift), i.e. raised by Shift relative to
		// the box's baseline.
		emit(c.Body, dl, x, y, scale)
	case SvgPath:
		// Path coords are in body-relative em with y measured from
		// baseline (negative = up).
		baselineY := y + b.Height*scale
		scaled := make([]path.Command, len(c.Commands))
		for i, cmd := range c.Commands {
			scaled[i] = scalePathCommand(cmd, scale)
		}
		dl.Items = append(dl.Items, PathItem{
			X: x, Y: baselineY,
			Commands: scaled, Fill: c.Fill, Color: b.Color,
		})
	case Angl:
		// Body shares baseline at y + b.Height*scale. Path commands are
		// in body-relative em with y measured from baseline (negative =
		// up). Path is emitted FIRST to match upstream item ordering.
		baselineY := y + b.Height*scale
		scaled := make([]path.Command, len(c.PathCommands))
		for i, cmd := range c.PathCommands {
			scaled[i] = scalePathCommand(cmd, scale)
		}
		dl.Items = append(dl.Items, PathItem{
			X: x, Y: baselineY,
			Commands: scaled, Fill: false, Color: b.Color,
		})
		emit(c.Body, dl, x, y+(b.Height-c.Body.Height)*scale, scale)
	case Empty, Kern:
		// no items
	}
}

// surdFontForInnerHeight picks the KaTeX font containing the √ glyph at a
// suitable size for the given inner radical height. Mirrors upstream
// surd_font_for_inner_height.
func surdFontForInnerHeight(h float64) string {
	switch {
	case h <= 1.0:
		return "Main-Regular"
	case h <= 1.2:
		return "Size1-Regular"
	case h <= 1.8:
		return "Size2-Regular"
	case h <= 2.4:
		return "Size3-Regular"
	}
	return "Size4-Regular"
}

func scalePathCommand(cmd path.Command, scale float64) path.Command {
	out := cmd
	switch cmd.Kind {
	case path.KindMoveTo, path.KindLineTo:
		out.X = cmd.X * scale
		out.Y = cmd.Y * scale
	case path.KindCubicTo:
		out.X1 = cmd.X1 * scale
		out.Y1 = cmd.Y1 * scale
		out.X2 = cmd.X2 * scale
		out.Y2 = cmd.Y2 * scale
		out.X = cmd.X * scale
		out.Y = cmd.Y * scale
	case path.KindQuadTo:
		out.X1 = cmd.X1 * scale
		out.Y1 = cmd.Y1 * scale
		out.X = cmd.X * scale
		out.Y = cmd.Y * scale
	}
	return out
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
