package mhchem

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dlclark/regexp2"
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
//
// Patterns use github.com/dlclark/regexp2 which supports the JS/Perl-style
// look-around and back-reference constructs the upstream patterns rely on
// (Go's stdlib regexp / RE2 doesn't).
type Data struct {
	Machines Machines
	Regexes  map[string]*regexp2.Regexp
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
	regexes := make(map[string]*regexp2.Regexp, len(pr.Regex))
	for name, src := range pr.Regex {
		// dlclark/regexp2 accepts JS/Perl/.NET-style syntax (look-around,
		// back-references) which the upstream patterns rely on. The
		// ECMAScript flag relaxes escape handling so e.g. \_ — accepted
		// in JS but rejected by .NET — also compiles.
		re, err := regexp2.Compile(src, regexp2.ECMAScript)
		if err != nil {
			return nil, fmt.Errorf("mhchem: compile pattern %q: %w", name, err)
		}
		regexes[name] = re
	}
	return &Data{Machines: machines, Regexes: regexes}, nil
}
