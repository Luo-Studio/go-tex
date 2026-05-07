// Package mhchem ports KaTeX/RaTeX's mhchem 3.3.0 chemistry parser.
//
// `\ce{...}` and `\pu{...}` invoke ChemParseStr to convert the chemistry
// source into a TeX expansion string. The host parser then re-parses the
// resulting TeX through the regular pipeline.
package mhchem

// Buffer mirrors the KaTeX mhchem `buffer` object used by the action
// handlers in actions.go. Slots a..rm are typed as *string so we can
// distinguish "unset" from "empty string".
type Buffer struct {
	ParenthesisLevel int
	BeginsWithBond   bool
	Sb               bool
	A                *string
	B                *string
	P                *string
	O                *string
	Q                *string
	D                *string
	DType            *string
	R                *string
	Rdt              *string
	Rd               *string
	Rqt              *string
	Rq               *string
	Text             *string
	Rm               *string
}

// NewBuffer returns a fresh, zeroed Buffer.
func NewBuffer() *Buffer { return &Buffer{} }

// ClearSoft clears all slots except ParenthesisLevel/BeginsWithBond.
func (b *Buffer) ClearSoft() {
	b.Sb = false
	b.A, b.B, b.P, b.O, b.Q = nil, nil, nil, nil, nil
	b.D, b.DType, b.R = nil, nil, nil
	b.Rdt, b.Rd, b.Rqt, b.Rq = nil, nil, nil, nil
	b.Text, b.Rm = nil, nil
}

// ClearAll resets the buffer to its zero value.
func (b *Buffer) ClearAll() { *b = Buffer{} }

// IsSlotEmpty reports whether s is nil or points to an empty string.
func IsSlotEmpty(s *string) bool { return s == nil || *s == "" }

// SetSlot writes v to *slot, allocating if nil.
func SetSlot(slot **string, v string) {
	if *slot == nil {
		s := v
		*slot = &s
		return
	}
	**slot = v
}
