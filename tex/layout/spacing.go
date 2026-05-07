package layout

import "github.com/luo-studio/go-tex/tex/parser"

// MathClass identifies the TeX math atom class for inter-atom spacing.
type MathClass uint8

const (
	ClassOrd MathClass = iota
	ClassOp
	ClassBin
	ClassRel
	ClassOpen
	ClassClose
	ClassPunct
	ClassInner
)

// spacingTable is TeXbook p.170 Table 18 in mu (1mu = 1/18 em).
//
//	  rows = left, cols = right
//	  order: Ord, Op, Bin, Rel, Open, Close, Punct, Inner
var spacingTable = [8][8]int8{
	/*Ord*/ {0, 3, 4, 5, 0, 0, 0, 3},
	/*Op*/ {3, 4, 0, 5, 0, 0, 0, 3},
	/*Bin*/ {4, 4, 0, 0, 4, 0, 0, 4},
	/*Rel*/ {5, 5, 0, 0, 5, 0, 0, 5},
	/*Open*/ {0, 0, 0, 0, 0, 0, 0, 0},
	/*Close*/ {0, 3, 4, 5, 0, 0, 0, 3},
	/*Punct*/ {3, 3, 0, 5, 3, 3, 3, 3},
	/*Inner*/ {3, 3, 4, 5, 3, 0, 3, 3},
}

// tightSpacingTable: same shape but for script/scriptscript styles.
var tightSpacingTable = [8][8]int8{
	{0, 3, 0, 0, 0, 0, 0, 0},
	{3, 3, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 3, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 3, 0, 0, 0, 0, 0, 0},
}

// atomSpacing returns the inter-atom spacing in mu.
func atomSpacing(left, right MathClass, tight bool) float64 {
	if tight {
		return float64(tightSpacingTable[left][right])
	}
	return float64(spacingTable[left][right])
}

// muToEm converts mu to em given the current style's quad width.
func muToEm(mu, quad float64) float64 { return mu * quad / 18.0 }

// nodeMathClass returns the atom class for a parser node, or invalid if the
// node has no spacing class (skipped in atom-pair lookups).
func nodeMathClass(n parser.Node) (MathClass, bool) {
	switch v := n.(type) {
	case *parser.Atom:
		switch v.Family {
		case parser.FamilyBin:
			return ClassBin, true
		case parser.FamilyClose:
			return ClassClose, true
		case parser.FamilyInner:
			return ClassInner, true
		case parser.FamilyOpen:
			return ClassOpen, true
		case parser.FamilyPunct:
			return ClassPunct, true
		case parser.FamilyRel:
			return ClassRel, true
		}
	case *parser.MClass:
		// \mathrel{...} / \mathbin{...} / etc. force the wrapped body to
		// be classified as the named class for outer spacing.
		switch v.Mclass {
		case "mrel":
			return ClassRel, true
		case "mbin":
			return ClassBin, true
		case "mopen":
			return ClassOpen, true
		case "mclose":
			return ClassClose, true
		case "mpunct":
			return ClassPunct, true
		case "minner":
			return ClassInner, true
		case "mop":
			return ClassOp, true
		}
		return ClassOrd, true
	case *parser.HtmlMathMl:
		// Derive class from the first meaningful child in the html branch.
		for _, child := range v.Html {
			if c, ok := nodeMathClass(child); ok {
				return c, true
			}
		}
		return ClassOrd, true
	case *parser.SupSub:
		if v.Base != nil {
			if c, ok := nodeMathClass(v.Base); ok {
				return c, true
			}
		}
		return ClassOrd, true
	case *parser.XArrow:
		return ClassRel, true
	case *parser.GenFrac:
		// With delimiters (\binom etc.) → Ord; without → Inner.
		hasL := v.LeftDelim != nil && *v.LeftDelim != "" && *v.LeftDelim != "."
		hasR := v.RightDelim != nil && *v.RightDelim != "" && *v.RightDelim != "."
		if hasL || hasR {
			return ClassOrd, true
		}
		return ClassInner, true
	case *parser.Lap:
		return 0, false
	case *parser.MathOrd, *parser.TextOrd, *parser.AccentToken,
		*parser.OrdGroup, *parser.Sqrt,
		*parser.Accent, *parser.AccentUnder, *parser.Font, *parser.Color,
		*parser.Phantom, *parser.VPhantom, *parser.Smash, *parser.Overline,
		*parser.Underline, *parser.Rule, *parser.RaiseBox,
		*parser.HBox, *parser.MathChoice, *parser.Styling, *parser.Sizing,
		*parser.Enclose, *parser.Pmb, *parser.Href, *parser.URL,
		*parser.HorizBrace, *parser.VCenter:
		return ClassOrd, true
	case *parser.OpToken, *parser.Op, *parser.OperatorName:
		return ClassOp, true
	case *parser.LeftRight:
		return ClassInner, true
	case *parser.DelimSizing:
		switch v.Mclass {
		case "mopen":
			return ClassOpen, true
		case "mclose":
			return ClassClose, true
		case "mrel":
			return ClassRel, true
		case "mord":
			return ClassOrd, true
		case "mbin":
			return ClassBin, true
		case "minner":
			return ClassInner, true
		case "mpunct":
			return ClassPunct, true
		case "mop":
			return ClassOp, true
		}
	case *parser.Spacing:
		// Spacing nodes don't participate in inter-atom glue lookups.
	case *parser.Kern:
		// Kern nodes don't participate either.
	}
	return 0, false
}

// applyBinCancellation rewrites Bin atoms that appear next to certain
// classes (Open, Rel, Op, Punct, …) into Ord, per TeXbook Rule 5/6
// (binLeftCanceller / binRightCanceller).
func applyBinCancellation(raw []optClass) []optClass {
	eff := make([]optClass, len(raw))
	copy(eff, raw)
	for i, c := range raw {
		if !c.ok || c.class != ClassBin {
			continue
		}
		var prev optClass
		if i > 0 {
			prev = raw[i-1]
		}
		leftCancel := !prev.ok ||
			prev.class == ClassBin || prev.class == ClassOpen ||
			prev.class == ClassRel || prev.class == ClassOp ||
			prev.class == ClassPunct
		if leftCancel {
			eff[i] = optClass{class: ClassOrd, ok: true}
		}
	}
	for i, c := range raw {
		if !c.ok || c.class != ClassBin {
			continue
		}
		var next optClass
		if i+1 < len(raw) {
			next = raw[i+1]
		}
		rightCancel := !next.ok ||
			next.class == ClassRel || next.class == ClassClose ||
			next.class == ClassPunct
		if rightCancel {
			eff[i] = optClass{class: ClassOrd, ok: true}
		}
	}
	return eff
}

// optClass is an Option<MathClass>.
type optClass struct {
	class MathClass
	ok    bool
}
