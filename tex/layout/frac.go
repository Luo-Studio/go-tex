package layout

import (
	"github.com/luo-studio/go-tex/tex/parser"
)

// layoutFraction implements TeXbook Rule 15d. Mirrors upstream's
// layout_fraction in engine.rs.
func layoutFraction(numer parser.Node, denom parser.Node, barThickness float64, continued bool, opts Options) *Box {
	numerStyle := opts.Style.Numerator()
	denomStyle := opts.Style.Denominator()
	numerOpts := opts.WithStyle(numerStyle)
	denomOpts := opts.WithStyle(denomStyle)

	numerBox := layoutNode(numer, numerOpts)
	if continued {
		pt := opts.Metrics().PtPerEm
		hMin := 8.5 / pt
		dMin := 3.5 / pt
		if numerBox.Height < hMin {
			numerBox.Height = hMin
		}
		if numerBox.Depth < dMin {
			numerBox.Depth = dMin
		}
	}
	denomBox := layoutNode(denom, denomOpts)

	numerRatio := numerStyle.SizeMultiplier() / opts.Style.SizeMultiplier()
	denomRatio := denomStyle.SizeMultiplier() / opts.Style.SizeMultiplier()

	numerHeight := numerBox.Height * numerRatio
	numerDepth := numerBox.Depth * numerRatio
	denomHeight := denomBox.Height * denomRatio
	denomDepth := denomBox.Depth * denomRatio
	numerWidth := numerBox.Width * numerRatio
	denomWidth := denomBox.Width * denomRatio

	metrics := opts.Metrics()
	axis := metrics.AxisHeight
	rule := barThickness

	var numShift, denShift float64
	if opts.Style.IsDisplay() {
		numShift = metrics.Num1
		denShift = metrics.Denom1
	} else if barThickness > 0 {
		numShift = metrics.Num2
		denShift = metrics.Denom2
	} else {
		numShift = metrics.Num3
		denShift = metrics.Denom2
	}

	if barThickness > 0 {
		var minClearance float64
		if opts.Style.IsDisplay() {
			minClearance = 3 * rule
		} else {
			minClearance = rule
		}
		numClearance := (numShift - numerDepth) - (axis + rule/2)
		if numClearance < minClearance {
			numShift += minClearance - numClearance
		}
		denClearance := (axis - rule/2) + (denShift - denomHeight)
		if denClearance < minClearance {
			denShift += minClearance - denClearance
		}
	} else {
		var minGap float64
		if opts.Style.IsDisplay() {
			minGap = 7 * metrics.DefaultRuleThickness
		} else {
			minGap = 3 * metrics.DefaultRuleThickness
		}
		gap := (numShift - numerDepth) - (denomHeight - denShift)
		if gap < minGap {
			adjust := (minGap - gap) / 2
			numShift += adjust
			denShift += adjust
		}
	}

	totalWidth := max64(numerWidth, denomWidth)
	height := numerHeight + numShift
	depth := denomDepth + denShift
	return &Box{
		Width:  totalWidth,
		Height: height,
		Depth:  depth,
		Color:  opts.Color,
		Content: Fraction{
			Numer:        numerBox,
			Denom:        denomBox,
			NumerShift:   numShift,
			DenomShift:   denShift,
			BarThickness: rule,
			NumerScale:   numerRatio,
			DenomScale:   denomRatio,
		},
	}
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
