// Package symbols provides KaTeX/RaTeX's symbol table: a lookup from a TeX
// command name (or single Unicode character) to its math/text classification
// (mathord, textord, atom-family, …) and Unicode codepoint.
//
// The data is mechanically derived from upstream's symbols_data.rs.
package symbols

// Mode is the parser's current mode when looking up a symbol.
type Mode uint8

const (
	ModeMath Mode = 0
	ModeText Mode = 1
)

// Font is which symbol-font the entry lives in (Main = KaTeX_Main, Ams =
// KaTeX_AMS). Mostly relevant for the layout engine; included here for
// completeness.
type Font uint8

const (
	FontMain Font = 0
	FontAMS  Font = 1
)

// Group is the group-name string KaTeX assigns to a symbol; it determines
// the AST node type or atom family the parser emits.
type Group string

const (
	GroupBin         Group = "bin"
	GroupClose       Group = "close"
	GroupInner       Group = "inner"
	GroupOpen        Group = "open"
	GroupPunct       Group = "punct"
	GroupRel         Group = "rel"
	GroupAccentToken Group = "accent-token"
	GroupMathOrd     Group = "mathord"
	GroupOpToken     Group = "op-token"
	GroupSpacing     Group = "spacing"
	GroupTextOrd     Group = "textord"
)

// IsAtom reports whether the group corresponds to a TeX atom family
// (bin/close/inner/open/punct/rel).
func (g Group) IsAtom() bool {
	switch g {
	case GroupBin, GroupClose, GroupInner, GroupOpen, GroupPunct, GroupRel:
		return true
	}
	return false
}

// Info is the resolved symbol information returned by Lookup.
type Info struct {
	Name      string
	Mode      Mode
	Font      Font
	Group     Group
	Codepoint rune // 0 if no associated codepoint
}

type modeKey struct {
	mode uint8
	key  string
}
type modeRune struct {
	mode uint8
	r    rune
}

var (
	byName     map[modeKey]int
	byCodepnt  map[modeRune]int
	indexBuilt bool
)

func buildIndex() {
	if indexBuilt {
		return
	}
	byName = make(map[modeKey]int, len(Entries))
	byCodepnt = make(map[modeRune]int, len(Entries))
	for i := range Entries {
		e := &Entries[i]
		k := modeKey{mode: e.Mode, key: e.Name}
		if _, ok := byName[k]; !ok {
			byName[k] = i
		}
		if e.Codepoint != 0 {
			r := modeRune{mode: e.Mode, r: e.Codepoint}
			if _, ok := byCodepnt[r]; !ok {
				byCodepnt[r] = i
			}
		}
	}
	indexBuilt = true
}

func indexToInfo(i int, mode Mode) Info {
	e := Entries[i]
	return Info{
		Name:      e.Name,
		Mode:      mode,
		Font:      Font(e.Font),
		Group:     Group(e.Group),
		Codepoint: e.Codepoint,
	}
}

// Lookup resolves a symbol by name in the given mode, falling back to a
// codepoint lookup when name is exactly one Unicode character (matching
// KaTeX's `acceptUnicodeChar`). Returns ok=false if not found.
func Lookup(name string, mode Mode) (Info, bool) {
	buildIndex()
	if i, ok := byName[modeKey{mode: uint8(mode), key: name}]; ok {
		return indexToInfo(i, mode), true
	}
	// Single-character name: try codepoint lookup.
	r, second := decodeOneRune(name)
	if r != 0 && !second {
		if i, ok := byCodepnt[modeRune{mode: uint8(mode), r: r}]; ok {
			return indexToInfo(i, mode), true
		}
	}
	return Info{}, false
}

// LookupByCodepoint resolves a symbol by codepoint in the given mode.
func LookupByCodepoint(r rune, mode Mode) (Info, bool) {
	buildIndex()
	if i, ok := byCodepnt[modeRune{mode: uint8(mode), r: r}]; ok {
		return indexToInfo(i, mode), true
	}
	return Info{}, false
}

// decodeOneRune returns (r, hasMore) where r is the first rune in s and
// hasMore is true if s contains more than one rune.
func decodeOneRune(s string) (rune, bool) {
	if len(s) == 0 {
		return 0, false
	}
	var r rune
	var sz int
	for i, c := range s {
		if i == 0 {
			r = c
			sz = i
			_ = sz
			continue
		}
		_ = c
		return r, true
	}
	return r, false
}
