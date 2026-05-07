// Package parser implements a KaTeX-compatible LaTeX parser. The AST
// produced here serializes to JSON in the exact shape that the upstream
// RaTeX `parse` binary emits, so byte-level parity tests are possible.
package parser

import (
	"encoding/json"
	"strconv"

	"github.com/luo-studio/go-tex/tex/source"
)

// Mode is the parse mode (math vs text).
type Mode uint8

const (
	ModeMath Mode = iota
	ModeText
)

// MarshalJSON encodes Mode as "math" or "text".
func (m Mode) MarshalJSON() ([]byte, error) {
	switch m {
	case ModeMath:
		return []byte(`"math"`), nil
	case ModeText:
		return []byte(`"text"`), nil
	}
	return nil, &json.UnsupportedValueError{Str: "invalid Mode"}
}

// UnmarshalJSON decodes "math" or "text" into a Mode.
func (m *Mode) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"math"`:
		*m = ModeMath
	case `"text"`:
		*m = ModeText
	default:
		return &json.SyntaxError{}
	}
	return nil
}

// AtomFamily is the spacing family for an Atom node.
type AtomFamily uint8

const (
	FamilyBin AtomFamily = iota
	FamilyClose
	FamilyInner
	FamilyOpen
	FamilyPunct
	FamilyRel
)

// MarshalJSON encodes the family as the lower-case KaTeX string.
func (f AtomFamily) MarshalJSON() ([]byte, error) {
	switch f {
	case FamilyBin:
		return []byte(`"bin"`), nil
	case FamilyClose:
		return []byte(`"close"`), nil
	case FamilyInner:
		return []byte(`"inner"`), nil
	case FamilyOpen:
		return []byte(`"open"`), nil
	case FamilyPunct:
		return []byte(`"punct"`), nil
	case FamilyRel:
		return []byte(`"rel"`), nil
	}
	return nil, &json.UnsupportedValueError{Str: "invalid AtomFamily"}
}

// UnmarshalJSON decodes a family string.
func (f *AtomFamily) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"bin"`:
		*f = FamilyBin
	case `"close"`:
		*f = FamilyClose
	case `"inner"`:
		*f = FamilyInner
	case `"open"`:
		*f = FamilyOpen
	case `"punct"`:
		*f = FamilyPunct
	case `"rel"`:
		*f = FamilyRel
	default:
		return &json.SyntaxError{}
	}
	return nil
}

// StyleStr is the style argument of `\displaystyle`, `\textstyle`, etc.
type StyleStr string

const (
	StyleDisplay      StyleStr = "display"
	StyleText         StyleStr = "text"
	StyleScript       StyleStr = "script"
	StyleScriptScript StyleStr = "scriptscript"
)

// Measurement is a number with a unit (e.g. {3.0, "pt"}).
type Measurement struct {
	Number float64 `json:"number"`
	Unit   string  `json:"unit"`
}

// MarshalJSON renders the Number with at least one decimal digit so the
// output matches upstream serde's f64 serialization (3 → "3.0", not "3").
func (m Measurement) MarshalJSON() ([]byte, error) {
	return []byte(`{"number":` + formatFloat(m.Number) + `,"unit":` + jsonString(m.Unit) + `}`), nil
}

// formatFloat serializes f matching Rust serde's f64 output: whole numbers
// get a trailing ".0", other values use Go's %g shortest form.
func formatFloat(f float64) string {
	// Use the shortest representation that round-trips.
	s := strconv.FormatFloat(f, 'g', -1, 64)
	// Detect "whole" number forms (no '.' and no 'e'/'E') and add ".0".
	hasDot := false
	hasExp := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			hasDot = true
		case 'e', 'E':
			hasExp = true
		}
	}
	if !hasDot && !hasExp {
		s += ".0"
	}
	return s
}

// jsonString quotes s for JSON. Uses encoding/json's MarshalString.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// jsonRawMarshal marshals v with the standard library encoder. Exposed here
// so other AST types (e.g. ArrayTag) can call into it without re-importing.
func jsonRawMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Loc is a pointer alias for source.Location so we can omit absent locations.
type Loc = *source.Location

// Node is any AST node. Nodes serialize to JSON with a "type" discriminator
// matching the upstream KaTeX shape.
type Node interface {
	NodeType() string
	NodeMode() Mode
}

// withType emits {"type": typeName, ...inner} where inner is v's normal JSON
// encoding without invoking v's own MarshalJSON.
func withType(typeName string, v any) ([]byte, error) {
	inner, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	// inner is {...}; replace leading '{' with '{"type":"<typeName>",'
	out := make([]byte, 0, len(inner)+len(typeName)+12)
	out = append(out, '{')
	out = append(out, `"type":`...)
	tn, _ := json.Marshal(typeName)
	out = append(out, tn...)
	if len(inner) > 2 { // not "{}"
		out = append(out, ',')
		out = append(out, inner[1:len(inner)-1]...)
	}
	out = append(out, '}')
	return out, nil
}

// =============================================================================
// Symbol nodes
// =============================================================================

// Atom is a symbol with a spacing family.
type Atom struct {
	Mode   Mode       `json:"mode"`
	Family AtomFamily `json:"family"`
	Text   string     `json:"text"`
	Loc    Loc        `json:"loc,omitempty"`
}

func (a *Atom) NodeType() string { return "atom" }
func (a *Atom) NodeMode() Mode   { return a.Mode }
func (a *Atom) MarshalJSON() ([]byte, error) {
	type alias Atom
	return withType("atom", (*alias)(a))
}

// MathOrd is an ordinary math symbol (variable letters, italic chars).
type MathOrd struct {
	Mode Mode   `json:"mode"`
	Text string `json:"text"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (m *MathOrd) NodeType() string { return "mathord" }
func (m *MathOrd) NodeMode() Mode   { return m.Mode }
func (m *MathOrd) MarshalJSON() ([]byte, error) {
	type alias MathOrd
	return withType("mathord", (*alias)(m))
}

// TextOrd is an ordinary text or numeric symbol (digits, upright text).
type TextOrd struct {
	Mode Mode   `json:"mode"`
	Text string `json:"text"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (t *TextOrd) NodeType() string { return "textord" }
func (t *TextOrd) NodeMode() Mode   { return t.Mode }
func (t *TextOrd) MarshalJSON() ([]byte, error) {
	type alias TextOrd
	return withType("textord", (*alias)(t))
}

// OpToken is a raw operator token before being wrapped into an Op node.
type OpToken struct {
	Mode Mode   `json:"mode"`
	Text string `json:"text"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (o *OpToken) NodeType() string { return "op-token" }
func (o *OpToken) NodeMode() Mode   { return o.Mode }
func (o *OpToken) MarshalJSON() ([]byte, error) {
	type alias OpToken
	return withType("op-token", (*alias)(o))
}

// AccentToken is a raw accent token before being wrapped into an Accent node.
type AccentToken struct {
	Mode Mode   `json:"mode"`
	Text string `json:"text"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (a *AccentToken) NodeType() string { return "accent-token" }
func (a *AccentToken) NodeMode() Mode   { return a.Mode }
func (a *AccentToken) MarshalJSON() ([]byte, error) {
	type alias AccentToken
	return withType("accent-token", (*alias)(a))
}

// Spacing is a fixed-size space command (\,, \;, \quad, …).
type Spacing struct {
	Mode Mode   `json:"mode"`
	Text string `json:"text"`
	Loc  Loc    `json:"loc,omitempty"`
}

func (s *Spacing) NodeType() string { return "spacing" }
func (s *Spacing) NodeMode() Mode   { return s.Mode }
func (s *Spacing) MarshalJSON() ([]byte, error) {
	type alias Spacing
	return withType("spacing", (*alias)(s))
}

// IsSymbol reports whether n is one of the symbol-bearing leaf node types.
func IsSymbol(n Node) bool {
	switch n.(type) {
	case *Atom, *MathOrd, *TextOrd, *OpToken, *AccentToken, *Spacing:
		return true
	}
	return false
}

// SymbolText returns the text of a symbol node, or ("", false) for non-symbols.
func SymbolText(n Node) (string, bool) {
	switch v := n.(type) {
	case *Atom:
		return v.Text, true
	case *MathOrd:
		return v.Text, true
	case *TextOrd:
		return v.Text, true
	case *OpToken:
		return v.Text, true
	case *AccentToken:
		return v.Text, true
	case *Spacing:
		return v.Text, true
	}
	return "", false
}

// =============================================================================
// Structural nodes
// =============================================================================

// OrdGroup is a `{ ... }`-bracketed sequence treated as a single ordinary atom.
type OrdGroup struct {
	Mode       Mode   `json:"mode"`
	Body       []Node `json:"body"`
	Semisimple *bool  `json:"semisimple,omitempty"`
	Loc        Loc    `json:"loc,omitempty"`
}

func (o *OrdGroup) NodeType() string { return "ordgroup" }
func (o *OrdGroup) NodeMode() Mode   { return o.Mode }
func (o *OrdGroup) MarshalJSON() ([]byte, error) {
	type alias OrdGroup
	// An empty body must serialise as `[]`, never `null` (matches upstream).
	a := *o
	if a.Body == nil {
		a.Body = []Node{}
	}
	return withType("ordgroup", (*alias)(&a))
}

// SupSub holds a base with optional super- and subscript.
type SupSub struct {
	Mode Mode `json:"mode"`
	Base Node `json:"base,omitempty"`
	Sup  Node `json:"sup,omitempty"`
	Sub  Node `json:"sub,omitempty"`
	Loc  Loc  `json:"loc,omitempty"`
}

func (s *SupSub) NodeType() string { return "supsub" }
func (s *SupSub) NodeMode() Mode   { return s.Mode }
func (s *SupSub) MarshalJSON() ([]byte, error) {
	type alias SupSub
	return withType("supsub", (*alias)(s))
}

// =============================================================================
// Helpers used by parser/functions
// =============================================================================

// NormalizeArgument: if arg is an OrdGroup with a single child, unwrap it.
func NormalizeArgument(arg Node) Node {
	if g, ok := arg.(*OrdGroup); ok && len(g.Body) == 1 {
		return g.Body[0]
	}
	return arg
}

// OrdArgument: if arg is an OrdGroup, return its body; otherwise wrap arg in
// a single-element slice.
func OrdArgument(arg Node) []Node {
	if g, ok := arg.(*OrdGroup); ok {
		return g.Body
	}
	return []Node{arg}
}
