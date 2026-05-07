package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/parser"
)

// layoutSupSubNode implements TeXbook Rule 18 (super-/subscript shifts).
// Mirrors upstream's layout_supsub in engine.rs.
func layoutSupSubNode(base, sup, sub parser.Node, opts Options) *Box {
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
