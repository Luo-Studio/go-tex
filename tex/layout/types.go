// Package layout implements TeX-style box layout, mirroring the upstream
// ratex-layout crate. The output box's width/height/depth (in em) drives
// the SVG/PNG renderers.
//
// The full upstream engine is ~7700 lines; this port grows incrementally
// and tracks progress via tex/layout/parity_test.go.
package layout

import (
	"github.com/luo-studio/go-tex/tex/fontmetrics"
	"github.com/luo-studio/go-tex/tex/mathstyle"
	"github.com/luo-studio/go-tex/tex/parser"
	"github.com/luo-studio/go-tex/tex/path"
)

// Color is a render-time RGBA colour. Mirrors ratex-types color::Color.
type Color struct {
	R, G, B, A uint8
}

// Black is the default color (0,0,0,255).
var Black = Color{0, 0, 0, 255}

// Options carry the math style and other rendering knobs through the layout
// recursion.
type Options struct {
	Style                  mathstyle.Style
	Color                  Color
	AlignRelationSpacing   *float64 // capped relation spacing inside align/aligned
	LeftRightDelimHeight   *float64 // stretch height for \middle delimiters in second pass
	InterGlyphKernEm       float64
}

// DefaultOptions returns the upstream LayoutOptions::default() — display
// style, black, no overrides.
func DefaultOptions() Options {
	return Options{Style: mathstyle.Display, Color: Black}
}

// SizeMultiplier returns the size multiplier for the current style.
func (o Options) SizeMultiplier() float64 { return o.Style.SizeMultiplier() }

// Metrics returns the math constants for the current style's size index.
func (o Options) Metrics() *fontmetrics.MathConstants {
	return fontmetrics.Global(o.Style.SizeIndex())
}

// WithStyle returns a copy of o with style replaced.
func (o Options) WithStyle(s mathstyle.Style) Options { o.Style = s; return o }

// WithColor returns a copy with color replaced.
func (o Options) WithColor(c Color) Options { o.Color = c; return o }

// Box is a TeX box: width × (height ascent + depth descent) — all in em.
type Box struct {
	Width   float64
	Height  float64
	Depth   float64
	Content Content
	Color   Color
}

// TotalHeight is height + depth.
func (b *Box) TotalHeight() float64 { return b.Height + b.Depth }

// NewEmpty returns a zero-size empty box.
func NewEmpty() *Box { return &Box{Color: Black, Content: Empty{}} }

// NewKern returns a horizontal kern box with the given em width.
func NewKern(w float64) *Box { return &Box{Width: w, Color: Black, Content: Kern{}} }

// NewRule returns a filled rule box of width × (height+depth).
func NewRule(w, h, d, thickness, raise float64) *Box {
	return &Box{Width: w, Height: h, Depth: d, Color: Black,
		Content: Rule{Thickness: thickness, Raise: raise}}
}

// Content is the discriminated body of a Box.
type Content interface{ contentMarker() }

func (Empty) contentMarker()    {}
func (Kern) contentMarker()     {}
func (Rule) contentMarker()     {}
func (Glyph) contentMarker()    {}
func (HBox) contentMarker()     {}
func (VBox) contentMarker()     {}
func (Fraction) contentMarker() {}
func (SupSub) contentMarker()   {}
func (Radical) contentMarker()  {}
func (OpLimits) contentMarker() {}
func (Accent) contentMarker()   {}
func (LeftRight) contentMarker(){}
func (Array) contentMarker()    {}
func (SvgPath) contentMarker()  {}
func (Framed) contentMarker()   {}
func (RaiseBox) contentMarker() {}
func (Scaled) contentMarker()   {}
func (Angl) contentMarker()     {}
func (Overline) contentMarker() {}
func (Underline) contentMarker(){}

// Empty is the zero-content placeholder.
type Empty struct{}

// Kern is empty horizontal space.
type Kern struct{}

// Rule is a filled rectangle.
type Rule struct {
	Thickness float64
	Raise     float64
}

// Glyph is a single character drawn from a KaTeX font.
type Glyph struct {
	FontID   string
	CharCode rune
}

// HBox is a horizontal list of children laid out left-to-right.
type HBox struct {
	Children []*Box
}

// VBox is a vertical list of children laid out top-to-bottom.
type VBox struct {
	Children []VBoxChild
}

// VBoxChild is one entry in a VBox: either a child Box (with shift) or a
// vertical Kern.
type VBoxChild struct {
	Kind  VBoxChildKind
	Shift float64
}

// VBoxChildKind is a sum type for VBoxChild.Kind.
type VBoxChildKind interface{ vboxChildKindMarker() }

func (VBoxBox) vboxChildKindMarker()  {}
func (VBoxKern) vboxChildKindMarker() {}

type VBoxBox struct {
	Box *Box
}
type VBoxKern struct {
	Em float64
}

// Fraction is numerator-over-denominator with optional bar.
type Fraction struct {
	Numer        *Box
	Denom        *Box
	NumerShift   float64
	DenomShift   float64
	BarThickness float64
	NumerScale   float64
	DenomScale   float64
}

// SupSub holds a base with optional super- and subscript.
type SupSub struct {
	Base             *Box
	Sup              *Box
	Sub              *Box
	SupShift         float64
	SubShift         float64
	SupScale         float64
	SubScale         float64
	CenterScripts    bool
	ItalicCorrection float64
	SubHKern         float64
}

// Radical is `\sqrt[index]{body}`.
type Radical struct {
	Body          *Box
	Index         *Box
	IndexOffset   float64
	IndexScale    float64
	RuleThickness float64
	InnerHeight   float64
}

// OpLimits is a big operator with limits above/below.
type OpLimits struct {
	Base      *Box
	Sup       *Box
	Sub       *Box
	BaseShift float64
	SupKern   float64
	SubKern   float64
	Slant     float64
	SupScale  float64
	SubScale  float64
}

// Accent is an over-/under-accent.
type Accent struct {
	Base       *Box
	AccentBox  *Box
	Clearance  float64
	Skew       float64
	IsBelow    bool
	UnderGapEm float64
}

// LeftRight is `\left ... \right` wrapping inner content.
type LeftRight struct {
	Left  *Box
	Right *Box
	Inner *Box
}

// Array is a 2D matrix layout.
type Array struct {
	Cells           [][]*Box
	ColWidths       []float64
	ColAligns       []byte // 'l', 'c', 'r'
	RowHeights      []float64
	RowDepths       []float64
	ColGap          float64
	Offset          float64
	ContentXOffset  float64
	ColSeparators   []*bool // nil = no rule, false = solid '|', true = dashed ':'
	HLinesBeforeRow [][]bool
	RuleThickness   float64
	DoubleRuleSep   float64
	ArrayInnerWidth float64
	TagGapEm        float64
	TagColWidth     float64
	RowTags         []*Box
	TagsLeft        bool
}

// SvgPath is a vector path used for arrows, braces, etc.
type SvgPath struct {
	Commands []path.Command
	Fill     bool
}

// Framed is a body wrapped in a (possibly drawn) border with optional fill.
type Framed struct {
	Body            *Box
	Padding         float64
	BorderThickness float64
	HasBorder       bool
	BgColor         *Color
	BorderColor     Color
}

// RaiseBox shifts a body vertically. Positive Shift moves content up.
type RaiseBox struct {
	Body  *Box
	Shift float64
}

// Scaled wraps a body at a different scale (used for \scriptstyle/etc.).
type Scaled struct {
	Body       *Box
	ChildScale float64
}

// Angl is the actuarial angle path with the body sharing its baseline.
type Angl struct {
	PathCommands []path.Command
	Body         *Box
}

// Overline draws a horizontal rule above its body.
type Overline struct {
	Body          *Box
	RuleThickness float64
}

// Underline draws a horizontal rule below its body.
type Underline struct {
	Body          *Box
	RuleThickness float64
}

// Layout is the public entry point: lay out a parsed body into a Box.
//
// This is a starting port; many node types fall back to an empty box. The
// layout-parity harness scores progress against the upstream `ratex-layout`
// CLI.
func Layout(nodes []parser.Node, options Options) *Box {
	return layoutExpression(nodes, options, true)
}
