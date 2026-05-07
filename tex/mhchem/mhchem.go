package mhchem

import "strings"

// ChemParseStr is the public entry point matching upstream's
// chem_parse_str(input, mode). Mode is "ce" (chemistry) or "pu"
// (physical units). On success it returns a TeX expansion string that the
// host parser can re-parse.
func ChemParseStr(input, mode string) (string, error) {
	d, err := LoadData()
	if err != nil {
		return "", err
	}
	var sm string
	switch mode {
	case "ce":
		sm = "ce"
	case "pu":
		sm = "pu"
	default:
		return "", errMsg("unknown mhchem mode (expected ce|pu): %s", mode)
	}
	ctx := &parserCtx{data: d}
	ast, err := goMachine(ctx, strings.TrimSpace(input), sm)
	if err != nil {
		return "", err
	}
	return texify(ast, false)
}
