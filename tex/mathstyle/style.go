// Package mathstyle implements TeX's math style transitions (D, T, S, SS and
// their cramped variants). The lookup tables match KaTeX's Style.ts so that
// every style transition produces byte-identical results.
package mathstyle

// Style is one of the eight TeX math styles.
type Style uint8

const (
	Display Style = iota
	DisplayCramped
	Text
	TextCramped
	Script
	ScriptCramped
	ScriptScript
	ScriptScriptCramped
)

// String returns a short human-readable name (D, Dc, T, Tc, S, Sc, SS, SSc).
func (s Style) String() string {
	switch s {
	case Display:
		return "D"
	case DisplayCramped:
		return "Dc"
	case Text:
		return "T"
	case TextCramped:
		return "Tc"
	case Script:
		return "S"
	case ScriptCramped:
		return "Sc"
	case ScriptScript:
		return "SS"
	case ScriptScriptCramped:
		return "SSc"
	default:
		return "?"
	}
}

// KaTeX Style.ts lookup tables, indexed by style id 0..7:
//
//	fracNum = [T, Tc, S, Sc, SS, SSc, SS, SSc]
//	fracDen = [Tc, Tc, Sc, Sc, SSc, SSc, SSc, SSc]
//	sup     = [S, Sc, S, Sc, SS, SSc, SS, SSc]
//	sub     = [Sc, Sc, Sc, Sc, SSc, SSc, SSc, SSc]
//	cramp   = [Dc, Dc, Tc, Tc, Sc, Sc, SSc, SSc]
//	text    = [D, Dc, T, Tc, T, Tc, T, Tc]
var (
	fracNum = [8]Style{Text, TextCramped, Script, ScriptCramped, ScriptScript, ScriptScriptCramped, ScriptScript, ScriptScriptCramped}
	fracDen = [8]Style{TextCramped, TextCramped, ScriptCramped, ScriptCramped, ScriptScriptCramped, ScriptScriptCramped, ScriptScriptCramped, ScriptScriptCramped}
	sup     = [8]Style{Script, ScriptCramped, Script, ScriptCramped, ScriptScript, ScriptScriptCramped, ScriptScript, ScriptScriptCramped}
	sub     = [8]Style{ScriptCramped, ScriptCramped, ScriptCramped, ScriptCramped, ScriptScriptCramped, ScriptScriptCramped, ScriptScriptCramped, ScriptScriptCramped}
	cramp   = [8]Style{DisplayCramped, DisplayCramped, TextCramped, TextCramped, ScriptCramped, ScriptCramped, ScriptScriptCramped, ScriptScriptCramped}
	textOf  = [8]Style{Display, DisplayCramped, Text, TextCramped, Text, TextCramped, Text, TextCramped}
)

// Numerator returns the style for the numerator of a fraction (preserves
// crampedness).
func (s Style) Numerator() Style { return fracNum[s] }

// Denominator returns the style for the denominator of a fraction (always
// cramped).
func (s Style) Denominator() Style { return fracDen[s] }

// Superscript returns the style for a superscript (preserves crampedness).
func (s Style) Superscript() Style { return sup[s] }

// Subscript returns the style for a subscript (always cramped).
func (s Style) Subscript() Style { return sub[s] }

// Cramped returns the cramped variant of the style.
func (s Style) Cramped() Style { return cramp[s] }

// Text returns the text-size equivalent (Script/ScriptScript become Text,
// preserving crampedness). Used inside `\text{}` blocks.
func (s Style) Text() Style { return textOf[s] }

// IsDisplay reports whether the style is Display or DisplayCramped.
func (s Style) IsDisplay() bool { return s == Display || s == DisplayCramped }

// IsCramped reports whether the style has the cramped flag set.
func (s Style) IsCramped() bool {
	switch s {
	case DisplayCramped, TextCramped, ScriptCramped, ScriptScriptCramped:
		return true
	}
	return false
}

// IsTight reports whether the style is one of the script sizes. Tight styles
// use reduced inter-atom spacing.
func (s Style) IsTight() bool {
	switch s {
	case Script, ScriptCramped, ScriptScript, ScriptScriptCramped:
		return true
	}
	return false
}

// SizeIndex returns the font-metrics size index: 0 (text), 1 (script), or 2
// (scriptscript).
func (s Style) SizeIndex() int {
	switch s {
	case Display, DisplayCramped, Text, TextCramped:
		return 0
	case Script, ScriptCramped:
		return 1
	default:
		return 2
	}
}

// SizeMultiplier returns the size multiplier relative to the base font size
// (text 1.0, script 0.7, scriptscript 0.5), per TeX's metric rules.
func (s Style) SizeMultiplier() float64 {
	switch s {
	case Display, DisplayCramped, Text, TextCramped:
		return 1.0
	case Script, ScriptCramped:
		return 0.7
	default:
		return 0.5
	}
}
