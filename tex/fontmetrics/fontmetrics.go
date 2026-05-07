// Package fontmetrics holds the per-character font metrics tables and the
// global TeX math constants (axis_height, num1, sup1, ...). All values are
// in em units, matching upstream's KaTeX/RaTeX fontMetricsData.
package fontmetrics

import "sort"

// Font names KaTeX uses. Mirrors upstream's FontId enum.
const (
	FontAMSRegular         = "AMS-Regular"
	FontCaligraphicRegular = "Caligraphic-Regular"
	FontFrakturRegular     = "Fraktur-Regular"
	FontFrakturBold        = "Fraktur-Bold"
	FontMainBold           = "Main-Bold"
	FontMainBoldItalic     = "Main-BoldItalic"
	FontMainItalic         = "Main-Italic"
	FontMainRegular        = "Main-Regular"
	FontMathBoldItalic     = "Math-BoldItalic"
	FontMathItalic         = "Math-Italic"
	FontSansSerifBold      = "SansSerif-Bold"
	FontSansSerifItalic    = "SansSerif-Italic"
	FontSansSerifRegular   = "SansSerif-Regular"
	FontScriptRegular      = "Script-Regular"
	FontSize1Regular       = "Size1-Regular"
	FontSize2Regular       = "Size2-Regular"
	FontSize3Regular       = "Size3-Regular"
	FontSize4Regular       = "Size4-Regular"
	FontTypewriterRegular  = "Typewriter-Regular"
)

// CharMetrics is per-character font metrics, all in em units.
type CharMetrics struct {
	Depth, Height, Italic, Skew, Width float64
}

// MathConstants is the set of TeX math constants used by the layout engine
// (all in em units). Mirrors upstream's MathConstants.
type MathConstants struct {
	Slant, Space, Stretch, Shrink, XHeight, Quad, ExtraSpace          float64
	Num1, Num2, Num3, Denom1, Denom2                                  float64
	Sup1, Sup2, Sup3, Sub1, Sub2, SupDrop, SubDrop                    float64
	Delim1, Delim2, AxisHeight, DefaultRuleThickness                  float64
	BigOpSpacing1, BigOpSpacing2, BigOpSpacing3, BigOpSpacing4        float64
	BigOpSpacing5                                                     float64
	SqrtRuleThickness, PtPerEm, DoubleRuleSep, ArrayRuleWidth         float64
	FBoxSep, FBoxRule                                                 float64
}

// CSSEmPerMu returns 1/18 of a quad — the conversion from `mu` to em.
func (m *MathConstants) CSSEmPerMu() float64 { return m.Quad / 18.0 }

// MathConstantsBySize is the three-element table indexed by size:
//
//	0: textstyle (>=9pt)   from cmsy10
//	1: scriptstyle (7-8pt) from cmsy7
//	2: scriptscriptstyle    from cmsy5
//
// Values from KaTeX's fontMetrics.ts sigmasAndXis table.
var MathConstantsBySize = [3]MathConstants{
	{ // textstyle
		Slant: 0.250, XHeight: 0.431, Quad: 1.0,
		Num1: 0.677, Num2: 0.394, Num3: 0.444,
		Denom1: 0.686, Denom2: 0.345,
		Sup1: 0.413, Sup2: 0.363, Sup3: 0.289,
		Sub1: 0.150, Sub2: 0.247,
		SupDrop: 0.386, SubDrop: 0.050,
		Delim1: 2.390, Delim2: 1.010,
		AxisHeight: 0.250, DefaultRuleThickness: 0.04,
		BigOpSpacing1: 0.111, BigOpSpacing2: 0.166,
		BigOpSpacing3: 0.2, BigOpSpacing4: 0.6, BigOpSpacing5: 0.1,
		SqrtRuleThickness: 0.04, PtPerEm: 10.0,
		DoubleRuleSep: 0.2, ArrayRuleWidth: 0.04,
		FBoxSep: 0.3, FBoxRule: 0.04,
	},
	{ // scriptstyle
		Slant: 0.250, XHeight: 0.431, Quad: 1.171,
		Num1: 0.732, Num2: 0.384, Num3: 0.471,
		Denom1: 0.752, Denom2: 0.344,
		Sup1: 0.503, Sup2: 0.431, Sup3: 0.286,
		Sub1: 0.143, Sub2: 0.286,
		SupDrop: 0.353, SubDrop: 0.071,
		Delim1: 1.700, Delim2: 1.157,
		AxisHeight: 0.250, DefaultRuleThickness: 0.049,
		BigOpSpacing1: 0.111, BigOpSpacing2: 0.166,
		BigOpSpacing3: 0.2, BigOpSpacing4: 0.611, BigOpSpacing5: 0.143,
		SqrtRuleThickness: 0.04, PtPerEm: 10.0,
		DoubleRuleSep: 0.2, ArrayRuleWidth: 0.04,
		FBoxSep: 0.3, FBoxRule: 0.04,
	},
	{ // scriptscriptstyle
		Slant: 0.250, XHeight: 0.431, Quad: 1.472,
		Num1: 0.925, Num2: 0.387, Num3: 0.504,
		Denom1: 1.025, Denom2: 0.532,
		Sup1: 0.504, Sup2: 0.404, Sup3: 0.294,
		Sub1: 0.200, Sub2: 0.400,
		SupDrop: 0.494, SubDrop: 0.100,
		Delim1: 1.980, Delim2: 1.420,
		AxisHeight: 0.250, DefaultRuleThickness: 0.049,
		BigOpSpacing1: 0.111, BigOpSpacing2: 0.166,
		BigOpSpacing3: 0.2, BigOpSpacing4: 0.611, BigOpSpacing5: 0.143,
		SqrtRuleThickness: 0.04, PtPerEm: 10.0,
		DoubleRuleSep: 0.2, ArrayRuleWidth: 0.04,
		FBoxSep: 0.3, FBoxRule: 0.04,
	},
}

// Global returns the math constants for the given size index (clamped to
// [0, 2]).
func Global(sizeIndex int) *MathConstants {
	if sizeIndex < 0 {
		sizeIndex = 0
	}
	if sizeIndex > 2 {
		sizeIndex = 2
	}
	return &MathConstantsBySize[sizeIndex]
}

// tableFor returns the metrics slice for a given font name, or nil for
// unknown fonts.
func tableFor(font string) []Entry {
	switch font {
	case FontAMSRegular:
		return AmsRegular
	case FontCaligraphicRegular:
		return CaligraphicRegular
	case FontFrakturRegular:
		return FrakturRegular
	case FontFrakturBold:
		return FrakturBold
	case FontMainBold:
		return MainBold
	case FontMainBoldItalic:
		return MainBolditalic
	case FontMainItalic:
		return MainItalic
	case FontMainRegular:
		return MainRegular
	case FontMathBoldItalic:
		return MathBolditalic
	case FontMathItalic:
		return MathItalic
	case FontSansSerifBold:
		return SansserifBold
	case FontSansSerifItalic:
		return SansserifItalic
	case FontSansSerifRegular:
		return SansserifRegular
	case FontScriptRegular:
		return ScriptRegular
	case FontSize1Regular:
		return Size1Regular
	case FontSize2Regular:
		return Size2Regular
	case FontSize3Regular:
		return Size3Regular
	case FontSize4Regular:
		return Size4Regular
	case FontTypewriterRegular:
		return TypewriterRegular
	}
	return nil
}

// Lookup returns the metrics for code in font, or (zero, false) if the
// glyph isn't present. Uses binary search on the sorted code column.
func Lookup(font string, code rune) (CharMetrics, bool) {
	t := tableFor(font)
	if t == nil {
		return CharMetrics{}, false
	}
	i := sort.Search(len(t), func(i int) bool { return t[i].Code >= code })
	if i < len(t) && t[i].Code == code {
		e := t[i]
		return CharMetrics{Depth: e.Depth, Height: e.Height, Italic: e.Italic, Skew: e.Skew, Width: e.Width}, true
	}
	return CharMetrics{}, false
}

// extraCharacterFallback maps Latin-1 / Cyrillic codepoints to similar Latin
// glyphs whose metrics we have. Mirrors KaTeX's extraCharacterMap.
var extraCharacterFallback = map[rune]rune{
	'Å': 'A', 'Ð': 'D', 'Þ': 'o', 'å': 'a', 'ð': 'd', 'þ': 'o',
	'А': 'A', 'Б': 'B', 'В': 'B', 'Г': 'F', 'Д': 'A',
	'Е': 'E', 'Ж': 'K', 'З': '3', 'И': 'N', 'Й': 'N',
	'К': 'K', 'Л': 'N', 'М': 'M', 'Н': 'H', 'О': 'O',
	'П': 'N', 'Р': 'P', 'С': 'C', 'Т': 'T', 'У': 'y',
	'Ф': 'O', 'Х': 'X', 'Ц': 'U', 'Ч': 'h', 'Ш': 'W',
	'Щ': 'W', 'Ъ': 'B', 'Ы': 'X', 'Ь': 'B', 'Э': '3',
	'Ю': 'X', 'Я': 'R',
	'а': 'a', 'б': 'b', 'в': 'a', 'г': 'r', 'д': 'y',
	'е': 'e', 'ж': 'm', 'з': 'e', 'и': 'n', 'й': 'n',
	'к': 'n', 'л': 'n', 'м': 'm', 'н': 'n', 'о': 'o',
	'п': 'n', 'р': 'p', 'с': 'c', 'т': 'o', 'у': 'y',
	'ф': 'b', 'х': 'x', 'ц': 'n', 'ч': 'n', 'ш': 'w',
	'щ': 'w', 'ъ': 'a', 'ы': 'm', 'ь': 'a', 'э': 'e',
	'ю': 'm', 'я': 'r',
}

// LookupWithFallback first tries Lookup, then falls back to the Latin/
// Cyrillic approximation table.
func LookupWithFallback(font string, code rune) (CharMetrics, bool) {
	if m, ok := Lookup(font, code); ok {
		return m, true
	}
	if mapped, ok := extraCharacterFallback[code]; ok {
		return Lookup(font, mapped)
	}
	return CharMetrics{}, false
}
