// Package source defines the location type shared by the lexer, parser, and
// downstream pipeline stages. A Location is a half-open byte range [Start, End)
// into the original LaTeX input string.
package source

// Location is a half-open byte range into the original input.
//
// Positions are byte offsets, not rune offsets, matching the upstream RaTeX
// (and KaTeX) convention. Tools that need column numbers should re-walk the
// input themselves.
type Location struct {
	Start int
	End   int
}

// Range returns the smallest location covering both a and b: it spans from
// a.Start to b.End. Callers must ensure a precedes b in the input.
func Range(a, b Location) Location {
	return Location{Start: a.Start, End: b.End}
}
