package mhchem

import (
	"strings"
	"unicode/utf8"
)

// Output value: a polymorphic JSON-like list. Either a string (a TeX
// fragment) or a tagged map ({type_: "...", ...}).
type Value any

// goMachine drives a state machine over input. Mirrors upstream
// engine::go_machine.
func goMachine(ctx *parserCtx, input, machine string) ([]Value, error) {
	input = normalizeInput(input)
	if input == "" {
		return nil, nil
	}
	mdef, ok := ctx.data.Machines[machine]
	if !ok {
		return nil, errMsg("unknown state machine %s", machine)
	}
	state := "0"
	buffer := NewBuffer()
	var output []Value
	lastInput := ""
	watchdog := 10

	for {
		if input != lastInput {
			watchdog = 10
			lastInput = input
		} else {
			watchdog--
		}
		if watchdog <= 0 {
			return nil, errMsg("mhchem bug U")
		}

		transitions := mdef.Transitions[state]
		if transitions == nil {
			transitions = mdef.Transitions["*"]
		}
		if transitions == nil {
			return nil, errMsg("no transitions for state %s", state)
		}

		consumed := false
	iter:
		for _, tr := range transitions {
			hit, err := matchPattern(ctx.data, tr.Pattern, input)
			if err != nil {
				return nil, err
			}
			if hit == nil {
				continue
			}
			consumed = true
			emptyAtMatch := input == ""
			for _, spec := range tr.Task.Action {
				piece, err := applyAction(ctx, machine, buffer, &hit.Token, &spec)
				if err != nil {
					return nil, err
				}
				output = append(output, piece...)
			}
			if tr.Task.NextState != nil {
				state = *tr.Task.NextState
			}
			if emptyAtMatch {
				return output, nil
			}
			if !tr.Task.Revisit {
				input = hit.Remainder
			}
			if !tr.Task.ToContinue {
				break iter
			}
		}
		if !consumed {
			return nil, errMsg("mhchem bug U")
		}
	}
}

func normalizeInput(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteByte(' ')
		case '−', '–', '—', '‐':
			b.WriteByte('-')
		case '…':
			b.WriteString("...")
		default:
			if r < 128 {
				b.WriteByte(byte(r))
				continue
			}
			var buf [4]byte
			n := utf8.EncodeRune(buf[:], r)
			b.Write(buf[:n])
		}
	}
	return b.String()
}

// parserCtx is the recursive driver context. Mirrors upstream ParserCtx.
type parserCtx struct {
	data *Data
}

func (c *parserCtx) go_(input, machine string) ([]Value, error) {
	return goMachine(c, input, machine)
}
