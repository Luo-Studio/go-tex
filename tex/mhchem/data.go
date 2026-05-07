package mhchem

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
)

//go:embed data/machines.json
var machinesJSON []byte

//go:embed data/patterns_regex.json
var patternsRegexJSON []byte

// Machines is the parsed top-level state-machine map.
type Machines map[string]MachineDef

// MachineDef is one named state machine.
type MachineDef struct {
	Transitions     map[string][]Transition `json:"transitions"`
	HasLocalActions bool                    `json:"hasLocalActions"`
}

// Transition is one input-pattern -> task pair within a state.
type Transition struct {
	Pattern string `json:"pattern"`
	Task    Task   `json:"task"`
}

// Task is the action+next-state to apply when a transition fires.
type Task struct {
	NextState  *string      `json:"nextState,omitempty"`
	Revisit    bool         `json:"revisit,omitempty"`
	ToContinue bool         `json:"toContinue,omitempty"`
	Action     []ActionSpec `json:"action_,omitempty"`
}

// ActionSpec names an action handler and its option payload.
type ActionSpec struct {
	Type   string          `json:"type_"`
	Option json.RawMessage `json:"option,omitempty"`
}

// Data holds the loaded JSON tables and compiled regex patterns.
type Data struct {
	Machines Machines
	Regexes  map[string]*regexp.Regexp
}

var (
	loadedData     *Data
	loadedDataOnce sync.Once
	loadedDataErr  error
)

// LoadData parses the embedded JSON and compiles patterns. Cached on first
// call. The Data is read-only after construction.
func LoadData() (*Data, error) {
	loadedDataOnce.Do(func() {
		loadedData, loadedDataErr = loadDataImpl()
	})
	return loadedData, loadedDataErr
}

// MustData returns the singleton Data, panicking on load failure.
func MustData() *Data {
	d, err := LoadData()
	if err != nil {
		panic(err)
	}
	return d
}

func loadDataImpl() (*Data, error) {
	machines := Machines{}
	if err := json.Unmarshal(machinesJSON, &machines); err != nil {
		return nil, fmt.Errorf("mhchem: parse machines.json: %w", err)
	}
	var pr struct {
		Regex map[string]string `json:"regex"`
	}
	if err := json.Unmarshal(patternsRegexJSON, &pr); err != nil {
		return nil, fmt.Errorf("mhchem: parse patterns_regex.json: %w", err)
	}
	regexes := make(map[string]*regexp.Regexp, len(pr.Regex))
	for name, src := range pr.Regex {
		// The upstream patterns assume Perl/JS regex semantics with
		// look-around. Go's RE2 doesn't support look-around. We compile
		// the patterns we can; ones that fail are stored as nil and
		// fall back at runtime to a slow string match (or skip the
		// pattern, accepting reduced coverage).
		re, err := regexp.Compile(translateRegex(src))
		if err != nil {
			regexes[name] = nil
			continue
		}
		regexes[name] = re
	}
	return &Data{Machines: machines, Regexes: regexes}, nil
}

// translateRegex strips look-around constructs that RE2 doesn't accept.
// This is a best-effort translation: many upstream patterns rely on
// look-ahead and won't match identically. Patterns that can't be
// translated stay as nil regexes (see loadDataImpl), and the engine
// falls back to literal matching where possible.
func translateRegex(src string) string {
	// (?=...) and (?!...) and (?<=...) etc. — drop the look-around so
	// the regex compiles. This loses semantic accuracy.
	out := []byte{}
	for i := 0; i < len(src); i++ {
		if i+2 < len(src) && src[i] == '(' && src[i+1] == '?' &&
			(src[i+2] == '=' || src[i+2] == '!' || src[i+2] == '<') {
			// Skip the look-around to its matching ')'.
			depth := 1
			j := i + 3
			if src[i+2] == '<' && j < len(src) && (src[j] == '=' || src[j] == '!') {
				j++
			}
			for j < len(src) && depth > 0 {
				switch src[j] {
				case '(':
					depth++
				case ')':
					depth--
				case '\\':
					if j+1 < len(src) {
						j++
					}
				}
				j++
			}
			i = j - 1
			continue
		}
		out = append(out, src[i])
	}
	return string(out)
}
