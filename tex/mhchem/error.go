package mhchem

import "fmt"

// Error is a user-facing mhchem parse error.
type Error struct {
	Msg   string
	Extra bool // ExtraClose: stray '}' encountered
}

func (e *Error) Error() string {
	if e.Extra {
		return "mhchem: extra closing brace"
	}
	return "mhchem: " + e.Msg
}

func errMsg(format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}

var errExtraClose = &Error{Extra: true}
