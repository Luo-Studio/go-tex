package mhchem

import "errors"

// ErrNotImplemented is returned by ChemParseStr while the mhchem state-
// machine port is incomplete. The buffer/data scaffolding is in place
// (buffer.go, data.go); engine.go, patterns.go, actions.go and texify.go
// still need to be ported from upstream.
var ErrNotImplemented = errors.New("mhchem: state-machine port not yet complete")

// ChemParseStr is the public entry point matching upstream's
// chem_parse_str(input, mode). Mode is "ce" (chemistry) or "pu"
// (physical units). On success it returns a TeX expansion string that the
// host parser can re-parse.
//
// The current implementation returns ErrNotImplemented; callers should
// fall back to leaving \ce / \pu unhandled.
func ChemParseStr(input, mode string) (string, error) {
	_, err := LoadData()
	if err != nil {
		return "", err
	}
	return "", ErrNotImplemented
}
