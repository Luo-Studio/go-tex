package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/parser"
)

// shouldUseOpLimits decides whether base+sup+sub should be drawn as a
// big op with limits stacked above/below (rather than to the right).
// Mirrors upstream should_use_op_limits.
func shouldUseOpLimits(base parser.Node, opts Options) bool {
	switch v := base.(type) {
	case *parser.Op:
		ahss := false
		if v.AlwaysHandleSupSub != nil {
			ahss = *v.AlwaysHandleSupSub
		}
		return v.Limits && (opts.Style.IsDisplay() || ahss)
	case *parser.OperatorName:
		return v.AlwaysHandleSupSub && (opts.Style.IsDisplay() || v.Limits)
	}
	return false
}

// layoutOpWithLimits handles `\sum_a^b`, `\lim\limits_x`, etc.: base
// with sub stacked below, sup stacked above. Mirrors upstream
// layout_op_with_limits + layout_op_limits_inner.
func layoutOpWithLimits(base parser.Node, sup, sub parser.Node, opts Options) *Box {
	var baseBox *Box
	suppressBaseShift := false
	switch v := base.(type) {
	case *parser.Op:
		if v.SuppressBaseShift != nil && *v.SuppressBaseShift {
			suppressBaseShift = true
		}
		// Mirror upstream build_op_base: for body ops we want the raw
		// laid-out body without the math-axis raise that layoutOp adds
		// for user-defined \mathop{...}. For named symbol/text ops we
		// want layoutOpSymbol / layoutOp's name path.
		if v.Body != nil && len(v.Body) > 0 {
			baseBox = layoutExpression(v.Body, opts, true)
		} else {
			baseBox = layoutOp(v, opts)
		}
	case *parser.OperatorName:
		baseBox = layoutOperatorName(v, opts)
	default:
		// Fallback to regular SupSub layout.
		return layoutSupSubNode(base, sup, sub, opts)
	}

	// Center symbol-style ops on the math axis (KaTeX baseShift).
	baseShift := 0.0
	if op, ok := base.(*parser.Op); ok && op.Symbol && !suppressBaseShift {
		axis := opts.Metrics().AxisHeight
		baseShift = (baseBox.Height-baseBox.Depth)/2 - axis
	}
	legacyKern := !suppressBaseShift

	m := opts.Metrics()
	supStyle := opts.Style.Superscript()
	subStyle := opts.Style.Subscript()
	supRatio := supStyle.SizeMultiplier() / opts.Style.SizeMultiplier()
	subRatio := subStyle.SizeMultiplier() / opts.Style.SizeMultiplier()

	extraKern := 0.0
	if legacyKern {
		extraKern = 0.08
	}

	var supBox, subBox *Box
	supKern, subKern := 0.0, 0.0
	if sup != nil {
		supBox = layoutNode(sup, opts.WithStyle(supStyle))
		d := supBox.Depth * supRatio
		if !legacyKern {
			d = supBox.Depth
		}
		supKern = m.BigOpSpacing1 + extraKern
		if v := m.BigOpSpacing3 - d + extraKern; v > supKern {
			supKern = v
		}
	}
	if sub != nil {
		subBox = layoutNode(sub, opts.WithStyle(subStyle))
		h := subBox.Height * subRatio
		if !legacyKern {
			h = subBox.Height
		}
		subKern = m.BigOpSpacing2 + extraKern
		if v := m.BigOpSpacing4 - h + extraKern; v > subKern {
			subKern = v
		}
	}
	sp5 := m.BigOpSpacing5

	var totalH, totalD, totalW float64
	switch {
	case supBox != nil && subBox != nil:
		supH := supBox.Height * supRatio
		supD := supBox.Depth * supRatio
		subH := subBox.Height * subRatio
		subD := subBox.Depth * subRatio
		bottom := sp5 + subH + subD + subKern + baseBox.Depth + baseShift
		totalH = baseBox.Height - baseShift + supKern + supH + supD + sp5
		totalD = bottom
		totalW = baseBox.Width
		if w := supBox.Width * supRatio; w > totalW {
			totalW = w
		}
		if w := subBox.Width * subRatio; w > totalW {
			totalW = w
		}
	case subBox != nil:
		subH := subBox.Height * subRatio
		subD := subBox.Depth * subRatio
		totalH = baseBox.Height - baseShift
		totalD = baseBox.Depth + baseShift + subKern + subH + subD + sp5
		totalW = baseBox.Width
		if w := subBox.Width * subRatio; w > totalW {
			totalW = w
		}
	case supBox != nil:
		supH := supBox.Height * supRatio
		supD := supBox.Depth * supRatio
		totalH = baseBox.Height - baseShift + supKern + supH + supD + sp5
		totalD = baseBox.Depth + baseShift
		totalW = baseBox.Width
		if w := supBox.Width * supRatio; w > totalW {
			totalW = w
		}
	default:
		return baseBox
	}

	// Slant of the base (italic correction of the symbol). Used to
	// shift the sup right and sub left, so they're centred relative to
	// the slanted glyph rather than its bounding box.
	slant := opSymbolSlant(baseBox)

	return &Box{
		Width: totalW, Height: totalH, Depth: totalD, Color: opts.Color,
		Content: OpLimits{
			Base:      baseBox,
			Sup:       supBox,
			Sub:       subBox,
			BaseShift: baseShift,
			SupKern:   supKern,
			SubKern:   subKern,
			Slant:     slant,
			SupScale:  supRatio,
			SubScale:  subRatio,
		},
	}
}

// opSymbolSlant returns the italic correction of the operator glyph.
func opSymbolSlant(b *Box) float64 {
	switch c := b.Content.(type) {
	case Glyph:
		if cm, ok := fontmetrics.LookupWithFallback(c.FontID, c.CharCode); ok {
			return cm.Italic
		}
	case HBox:
		if len(c.Children) > 0 {
			return opSymbolSlant(c.Children[len(c.Children)-1])
		}
	}
	return 0
}

// layoutSupSubNode implements TeXbook Rule 18 (super-/subscript shifts).
// Mirrors upstream's layout_supsub in engine.rs.
func layoutSupSubNode(base, sup, sub parser.Node, opts Options) *Box {
	if base != nil && shouldUseOpLimits(base, opts) {
		return layoutOpWithLimits(base, sup, sub, opts)
	}
	var baseBox *Box
	if base != nil {
		baseBox = layoutNode(base, opts)
	} else {
		baseBox = NewEmpty()
	}
	isCharBox := isCharacterBox(base)
	metrics := opts.Metrics()
	scriptSpace := 0.5 / metrics.PtPerEm / opts.SizeMultiplier()

	supStyle := opts.Style.Superscript()
	subStyle := opts.Style.Subscript()
	supRatio := supStyle.SizeMultiplier() / opts.Style.SizeMultiplier()
	subRatio := subStyle.SizeMultiplier() / opts.Style.SizeMultiplier()

	var supBox, subBox *Box
	if sup != nil {
		supBox = layoutNode(sup, opts.WithStyle(supStyle))
	}
	if sub != nil {
		subBox = layoutNode(sub, opts.WithStyle(subStyle))
	}

	supH := 0.0
	supD := 0.0
	subH := 0.0
	subD := 0.0
	if supBox != nil {
		supH = supBox.Height * supRatio
		supD = supBox.Depth * supRatio
	}
	if subBox != nil {
		subH = subBox.Height * subRatio
		subD = subBox.Depth * subRatio
	}

	supStyleM := fontmetrics.Global(supStyle.SizeIndex())
	subStyleM := fontmetrics.Global(subStyle.SizeIndex())

	supShift := 0.0
	subShift := 0.0
	if !isCharBox && supBox != nil {
		supShift = baseBox.Height - supStyleM.SupDrop*supRatio
	}
	if !isCharBox && subBox != nil {
		subShift = baseBox.Depth + subStyleM.SubDrop*subRatio
	}

	var minSupShift float64
	switch {
	case opts.Style.IsCramped():
		minSupShift = metrics.Sup3
	case opts.Style.IsDisplay():
		minSupShift = metrics.Sup1
	default:
		minSupShift = metrics.Sup2
	}

	if supBox != nil && subBox != nil {
		supShift = max64(max64(supShift, minSupShift), supD+0.25*metrics.XHeight)
		subShift = max64(subShift, metrics.Sub2)
		ruleW := metrics.DefaultRuleThickness
		maxW := 4 * ruleW
		gap := (supShift - supD) - (subH - subShift)
		if gap < maxW {
			subShift = maxW - (supShift - supD) + subH
			psi := 0.8*metrics.XHeight - (supShift - supD)
			if psi > 0 {
				supShift += psi
				subShift -= psi
			}
		}
	} else if subBox != nil {
		subShift = max64(max64(subShift, metrics.Sub1), subH-0.8*metrics.XHeight)
	} else if supBox != nil {
		supShift = max64(max64(supShift, minSupShift), supD+0.25*metrics.XHeight)
	}

	height := baseBox.Height
	depth := baseBox.Depth
	totalWidth := baseBox.Width

	if supBox != nil {
		if h := supShift + supH; h > height {
			height = h
		}
		if w := baseBox.Width + supBox.Width*supRatio + scriptSpace; w > totalWidth {
			totalWidth = w
		}
	}
	if subBox != nil {
		if d := subShift + subD; d > depth {
			depth = d
		}
		if w := baseBox.Width + subBox.Width*subRatio + scriptSpace; w > totalWidth {
			totalWidth = w
		}
	}

	return &Box{
		Width:  totalWidth,
		Height: height,
		Depth:  depth,
		Color:  opts.Color,
		Content: SupSub{
			Base:     baseBox,
			Sup:      supBox,
			Sub:      subBox,
			SupShift: supShift,
			SubShift: subShift,
			SupScale: supRatio,
			SubScale: subRatio,
		},
	}
}

// isCharacterBox reports whether node is "character-like" — used by
// supsub to decide whether to consume sup_drop / sub_drop. Matches the
// upstream is_character_box helper.
func isCharacterBox(n parser.Node) bool {
	if n == nil {
		return false
	}
	switch n.(type) {
	case *parser.MathOrd, *parser.TextOrd, *parser.Atom,
		*parser.OpToken, *parser.AccentToken, *parser.Spacing:
		return true
	}
	return false
}
