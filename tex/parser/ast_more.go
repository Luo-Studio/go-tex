package parser

// =============================================================================
// Function-generated nodes. Each variant is a separate Go struct so its JSON
// shape (field order, tag renames, omitempty rules) can mirror the upstream
// serde layout exactly.
// =============================================================================

// GenFrac is a generalized fraction (\frac, \dfrac, \tfrac, \binom, …).
type GenFrac struct {
	Mode       Mode         `json:"mode"`
	Continued  bool         `json:"continued"`
	Numer      Node         `json:"numer"`
	Denom      Node         `json:"denom"`
	HasBarLine bool         `json:"hasBarLine"`
	LeftDelim  *string      `json:"leftDelim"`
	RightDelim *string      `json:"rightDelim"`
	BarSize    *Measurement `json:"barSize"`
	Loc        Loc          `json:"loc,omitempty"`
}

func (g *GenFrac) NodeType() string { return "genfrac" }
func (g *GenFrac) NodeMode() Mode   { return g.Mode }
func (g *GenFrac) MarshalJSON() ([]byte, error) {
	type alias GenFrac
	return withType("genfrac", (*alias)(g))
}

// Sqrt is `\sqrt[index]{body}`.
type Sqrt struct {
	Mode  Mode `json:"mode"`
	Body  Node `json:"body"`
	Index Node `json:"index,omitempty"`
	Loc   Loc  `json:"loc,omitempty"`
}

func (s *Sqrt) NodeType() string { return "sqrt" }
func (s *Sqrt) NodeMode() Mode   { return s.Mode }
func (s *Sqrt) MarshalJSON() ([]byte, error) {
	type alias Sqrt
	return withType("sqrt", (*alias)(s))
}

// Accent is `\hat{x}`, `\tilde{x}`, etc.
type Accent struct {
	Mode       Mode  `json:"mode"`
	Label      string `json:"label"`
	IsStretchy *bool  `json:"isStretchy,omitempty"`
	IsShifty   *bool  `json:"isShifty,omitempty"`
	Base       Node   `json:"base"`
	Loc        Loc    `json:"loc,omitempty"`
}

func (a *Accent) NodeType() string { return "accent" }
func (a *Accent) NodeMode() Mode   { return a.Mode }
func (a *Accent) MarshalJSON() ([]byte, error) {
	type alias Accent
	return withType("accent", (*alias)(a))
}

// AccentUnder is `\underbrace`, `\underline`-style under-accents.
type AccentUnder struct {
	Mode       Mode  `json:"mode"`
	Label      string `json:"label"`
	IsStretchy *bool  `json:"isStretchy,omitempty"`
	IsShifty   *bool  `json:"isShifty,omitempty"`
	Base       Node   `json:"base"`
	Loc        Loc    `json:"loc,omitempty"`
}

func (a *AccentUnder) NodeType() string { return "accentUnder" }
func (a *AccentUnder) NodeMode() Mode   { return a.Mode }
func (a *AccentUnder) MarshalJSON() ([]byte, error) {
	type alias AccentUnder
	return withType("accentUnder", (*alias)(a))
}

// Op is a math operator like \sin, \int, \sum.
type Op struct {
	Mode                Mode    `json:"mode"`
	Limits              bool    `json:"limits"`
	AlwaysHandleSupSub  *bool   `json:"alwaysHandleSupSub,omitempty"`
	SuppressBaseShift   *bool   `json:"suppressBaseShift,omitempty"`
	ParentIsSupSub      bool    `json:"parentIsSupSub"`
	Symbol              bool    `json:"symbol"`
	Name                *string `json:"name,omitempty"`
	Body                []Node  `json:"body,omitempty"`
	Loc                 Loc     `json:"loc,omitempty"`
}

func (o *Op) NodeType() string { return "op" }
func (o *Op) NodeMode() Mode   { return o.Mode }
func (o *Op) MarshalJSON() ([]byte, error) {
	type alias Op
	return withType("op", (*alias)(o))
}

// OperatorName is `\operatorname{...}`.
type OperatorName struct {
	Mode               Mode   `json:"mode"`
	Body               []Node `json:"body"`
	AlwaysHandleSupSub bool   `json:"alwaysHandleSupSub"`
	Limits             bool   `json:"limits"`
	ParentIsSupSub     bool   `json:"parentIsSupSub"`
	Loc                Loc    `json:"loc,omitempty"`
}

func (o *OperatorName) NodeType() string { return "operatorname" }
func (o *OperatorName) NodeMode() Mode   { return o.Mode }
func (o *OperatorName) MarshalJSON() ([]byte, error) {
	type alias OperatorName
	return withType("operatorname", (*alias)(o))
}

// Font is `\mathrm{...}`, `\mathbf{...}`, etc.
type Font struct {
	Mode Mode   `json:"mode"`
	Font string `json:"font"`
	Body Node   `json:"body"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (f *Font) NodeType() string { return "font" }
func (f *Font) NodeMode() Mode   { return f.Mode }
func (f *Font) MarshalJSON() ([]byte, error) {
	type alias Font
	return withType("font", (*alias)(f))
}

// Text is a `\text{...}` block — switches to text mode for its body.
type Text struct {
	Mode Mode    `json:"mode"`
	Body []Node  `json:"body"`
	Font *string `json:"font,omitempty"`
	Loc  Loc     `json:"loc,omitempty"`
}

func (t *Text) NodeType() string { return "text" }
func (t *Text) NodeMode() Mode   { return t.Mode }
func (t *Text) MarshalJSON() ([]byte, error) {
	type alias Text
	return withType("text", (*alias)(t))
}

// Color is `\textcolor{red}{...}` / `\color{red}` body.
type Color struct {
	Mode  Mode   `json:"mode"`
	Color string `json:"color"`
	Body  []Node `json:"body"`
	Loc   Loc    `json:"loc,omitempty"`
}

func (c *Color) NodeType() string { return "color" }
func (c *Color) NodeMode() Mode   { return c.Mode }
func (c *Color) MarshalJSON() ([]byte, error) {
	type alias Color
	return withType("color", (*alias)(c))
}

// ColorToken is the colour argument before being applied to a body.
type ColorToken struct {
	Mode  Mode   `json:"mode"`
	Color string `json:"color"`
	Loc   Loc    `json:"loc,omitempty"`
}

func (c *ColorToken) NodeType() string { return "color-token" }
func (c *ColorToken) NodeMode() Mode   { return c.Mode }
func (c *ColorToken) MarshalJSON() ([]byte, error) {
	type alias ColorToken
	return withType("color-token", (*alias)(c))
}

// Size is a parsed measurement value (e.g. `1pt`, `2em`).
type Size struct {
	Mode    Mode        `json:"mode"`
	Value   Measurement `json:"value"`
	IsBlank bool        `json:"isBlank"`
	Loc     Loc         `json:"loc,omitempty"`
}

func (s *Size) NodeType() string { return "size" }
func (s *Size) NodeMode() Mode   { return s.Mode }
func (s *Size) MarshalJSON() ([]byte, error) {
	type alias Size
	return withType("size", (*alias)(s))
}

// Styling is `{\displaystyle ...}`, etc.
type Styling struct {
	Mode  Mode     `json:"mode"`
	Style StyleStr `json:"style"`
	Body  []Node   `json:"body"`
	Loc   Loc      `json:"loc,omitempty"`
}

func (s *Styling) NodeType() string { return "styling" }
func (s *Styling) NodeMode() Mode   { return s.Mode }
func (s *Styling) MarshalJSON() ([]byte, error) {
	type alias Styling
	return withType("styling", (*alias)(s))
}

// Sizing is `\Huge ...`.
type Sizing struct {
	Mode Mode   `json:"mode"`
	Size uint8  `json:"size"`
	Body []Node `json:"body"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (s *Sizing) NodeType() string { return "sizing" }
func (s *Sizing) NodeMode() Mode   { return s.Mode }
func (s *Sizing) MarshalJSON() ([]byte, error) {
	type alias Sizing
	return withType("sizing", (*alias)(s))
}

// DelimSizing is `\big`, `\Big`, etc.
type DelimSizing struct {
	Mode   Mode   `json:"mode"`
	Size   uint8  `json:"size"`
	Mclass string `json:"mclass"`
	Delim  string `json:"delim"`
	Loc    Loc    `json:"loc,omitempty"`
}

func (d *DelimSizing) NodeType() string { return "delimsizing" }
func (d *DelimSizing) NodeMode() Mode   { return d.Mode }
func (d *DelimSizing) MarshalJSON() ([]byte, error) {
	type alias DelimSizing
	return withType("delimsizing", (*alias)(d))
}

// LeftRight is `\left( ... \right)`.
type LeftRight struct {
	Mode       Mode    `json:"mode"`
	Body       []Node  `json:"body"`
	Left       string  `json:"left"`
	Right      string  `json:"right"`
	RightColor *string `json:"rightColor,omitempty"`
	Loc        Loc     `json:"loc,omitempty"`
}

func (l *LeftRight) NodeType() string { return "leftright" }
func (l *LeftRight) NodeMode() Mode   { return l.Mode }
func (l *LeftRight) MarshalJSON() ([]byte, error) {
	type alias LeftRight
	return withType("leftright", (*alias)(l))
}

// LeftRightRight is the right-hand `\right<delim>` placeholder before being
// merged into a LeftRight.
type LeftRightRight struct {
	Mode  Mode    `json:"mode"`
	Delim string  `json:"delim"`
	Color *string `json:"color,omitempty"`
	Loc   Loc     `json:"loc,omitempty"`
}

func (l *LeftRightRight) NodeType() string { return "leftright-right" }
func (l *LeftRightRight) NodeMode() Mode   { return l.Mode }
func (l *LeftRightRight) MarshalJSON() ([]byte, error) {
	type alias LeftRightRight
	return withType("leftright-right", (*alias)(l))
}

// Middle is `\middle|` between `\left` and `\right`.
type Middle struct {
	Mode  Mode   `json:"mode"`
	Delim string `json:"delim"`
	Loc   Loc    `json:"loc,omitempty"`
}

func (m *Middle) NodeType() string { return "middle" }
func (m *Middle) NodeMode() Mode   { return m.Mode }
func (m *Middle) MarshalJSON() ([]byte, error) {
	type alias Middle
	return withType("middle", (*alias)(m))
}

// Overline is `\overline{x}`.
type Overline struct {
	Mode Mode `json:"mode"`
	Body Node `json:"body"`
	Loc  Loc  `json:"loc,omitempty"`
}

func (o *Overline) NodeType() string { return "overline" }
func (o *Overline) NodeMode() Mode   { return o.Mode }
func (o *Overline) MarshalJSON() ([]byte, error) {
	type alias Overline
	return withType("overline", (*alias)(o))
}

// Underline is `\underline{x}`.
type Underline struct {
	Mode Mode `json:"mode"`
	Body Node `json:"body"`
	Loc  Loc  `json:"loc,omitempty"`
}

func (u *Underline) NodeType() string { return "underline" }
func (u *Underline) NodeMode() Mode   { return u.Mode }
func (u *Underline) MarshalJSON() ([]byte, error) {
	type alias Underline
	return withType("underline", (*alias)(u))
}

// Rule is `\rule[shift]{width}{height}`.
type Rule struct {
	Mode   Mode         `json:"mode"`
	Shift  *Measurement `json:"shift,omitempty"`
	Width  Measurement  `json:"width"`
	Height Measurement  `json:"height"`
	Loc    Loc          `json:"loc,omitempty"`
}

func (r *Rule) NodeType() string { return "rule" }
func (r *Rule) NodeMode() Mode   { return r.Mode }
func (r *Rule) MarshalJSON() ([]byte, error) {
	type alias Rule
	return withType("rule", (*alias)(r))
}

// Kern is `\kern1pt`, `\mkern`, etc.
type Kern struct {
	Mode      Mode        `json:"mode"`
	Dimension Measurement `json:"dimension"`
	Loc       Loc         `json:"loc,omitempty"`
}

func (k *Kern) NodeType() string { return "kern" }
func (k *Kern) NodeMode() Mode   { return k.Mode }
func (k *Kern) MarshalJSON() ([]byte, error) {
	type alias Kern
	return withType("kern", (*alias)(k))
}

// Phantom is `\phantom{...}`.
type Phantom struct {
	Mode Mode   `json:"mode"`
	Body []Node `json:"body"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (p *Phantom) NodeType() string { return "phantom" }
func (p *Phantom) NodeMode() Mode   { return p.Mode }
func (p *Phantom) MarshalJSON() ([]byte, error) {
	type alias Phantom
	return withType("phantom", (*alias)(p))
}

// VPhantom is `\vphantom{...}`.
type VPhantom struct {
	Mode Mode `json:"mode"`
	Body Node `json:"body"`
	Loc  Loc  `json:"loc,omitempty"`
}

func (p *VPhantom) NodeType() string { return "vphantom" }
func (p *VPhantom) NodeMode() Mode   { return p.Mode }
func (p *VPhantom) MarshalJSON() ([]byte, error) {
	type alias VPhantom
	return withType("vphantom", (*alias)(p))
}

// Smash is `\smash[h]{...}`.
type Smash struct {
	Mode        Mode `json:"mode"`
	Body        Node `json:"body"`
	SmashHeight bool `json:"smashHeight"`
	SmashDepth  bool `json:"smashDepth"`
	Loc         Loc  `json:"loc,omitempty"`
}

func (s *Smash) NodeType() string { return "smash" }
func (s *Smash) NodeMode() Mode   { return s.Mode }
func (s *Smash) MarshalJSON() ([]byte, error) {
	type alias Smash
	return withType("smash", (*alias)(s))
}

// MClass is `\mathbin{...}`, `\mathrel{...}`, etc.
type MClass struct {
	Mode            Mode   `json:"mode"`
	Mclass          string `json:"mclass"`
	Body            []Node `json:"body"`
	IsCharacterBox  bool   `json:"isCharacterBox"`
	Loc             Loc    `json:"loc,omitempty"`
}

func (m *MClass) NodeType() string { return "mclass" }
func (m *MClass) NodeMode() Mode   { return m.Mode }
func (m *MClass) MarshalJSON() ([]byte, error) {
	type alias MClass
	return withType("mclass", (*alias)(m))
}

// Cr is `\\` or `\cr` inside an array.
type Cr struct {
	Mode    Mode         `json:"mode"`
	NewLine bool         `json:"newLine"`
	Size    *Measurement `json:"size,omitempty"`
	Loc     Loc          `json:"loc,omitempty"`
}

func (c *Cr) NodeType() string { return "cr" }
func (c *Cr) NodeMode() Mode   { return c.Mode }
func (c *Cr) MarshalJSON() ([]byte, error) {
	type alias Cr
	return withType("cr", (*alias)(c))
}

// Infix is a `\over`, `\atop`-style infix operator before being lowered to
// a GenFrac.
type Infix struct {
	Mode        Mode         `json:"mode"`
	ReplaceWith string       `json:"replaceWith"`
	Size        *Measurement `json:"size,omitempty"`
	Loc         Loc          `json:"loc,omitempty"`
}

func (i *Infix) NodeType() string { return "infix" }
func (i *Infix) NodeMode() Mode   { return i.Mode }
func (i *Infix) MarshalJSON() ([]byte, error) {
	type alias Infix
	return withType("infix", (*alias)(i))
}

// Internal is the no-op placeholder produced by `\relax` and friends.
type Internal struct {
	Mode Mode `json:"mode"`
	Loc  Loc  `json:"loc,omitempty"`
}

func (i *Internal) NodeType() string { return "internal" }
func (i *Internal) NodeMode() Mode   { return i.Mode }
func (i *Internal) MarshalJSON() ([]byte, error) {
	type alias Internal
	return withType("internal", (*alias)(i))
}

// HorizBrace is `\overbrace{...}` / `\underbrace{...}`.
type HorizBrace struct {
	Mode   Mode   `json:"mode"`
	Label  string `json:"label"`
	IsOver bool   `json:"isOver"`
	Base   Node   `json:"base"`
	Loc    Loc    `json:"loc,omitempty"`
}

func (h *HorizBrace) NodeType() string { return "horizBrace" }
func (h *HorizBrace) NodeMode() Mode   { return h.Mode }
func (h *HorizBrace) MarshalJSON() ([]byte, error) {
	type alias HorizBrace
	return withType("horizBrace", (*alias)(h))
}

// Verb is `\verb|...|`.
type Verb struct {
	Mode Mode   `json:"mode"`
	Body string `json:"body"`
	Star bool   `json:"star"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (v *Verb) NodeType() string { return "verb" }
func (v *Verb) NodeMode() Mode   { return v.Mode }
func (v *Verb) MarshalJSON() ([]byte, error) {
	type alias Verb
	return withType("verb", (*alias)(v))
}
