package mhchem

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	reDigitsOnly = regexp.MustCompile(`^[0-9]+$`)
	reCelsiusC   = regexp.MustCompile("°C|\\^oC|\\^\\{o\\}C")
	reCelsiusF   = regexp.MustCompile("°F|\\^oF|\\^\\{o\\}F")
	reHalf       = regexp.MustCompile(`^([0-9]+|\$[a-z]\$|[a-z])\/([0-9]+)(\$[a-z]\$|[a-z])?$`)
)

// applyAction dispatches one action spec against the buffer / match token.
// Mirrors upstream actions::apply.
func applyAction(ctx *parserCtx, machine string, buffer *Buffer, m *MatchToken, spec *ActionSpec) ([]Value, error) {
	t := spec.Type
	switch {
	case machine == "ce" && t == "output":
		return ceOutput(ctx, buffer, spec.Option)
	case machine == "ce" && t == "o after d":
		return ceOAfterD(ctx, buffer, m)
	case machine == "ce" && t == "d= kv":
		SetSlot(&buffer.D, tokenString(m))
		s := "kv"
		buffer.DType = &s
		return nil, nil
	case machine == "ce" && t == "charge or bond":
		return ceChargeOrBond(ctx, buffer, m)
	case machine == "ce" && t == "- after o/d":
		afterD := optBool(spec.Option)
		return ceAfterOD(ctx, buffer, m, afterD)
	case machine == "ce" && t == "a to o":
		buffer.O = buffer.A
		buffer.A = nil
		return nil, nil
	case machine == "ce" && t == "sb=true":
		buffer.Sb = true
		return nil, nil
	case machine == "ce" && t == "sb=false":
		buffer.Sb = false
		return nil, nil
	case machine == "ce" && t == "beginsWithBond=true":
		buffer.BeginsWithBond = true
		return nil, nil
	case machine == "ce" && t == "beginsWithBond=false":
		buffer.BeginsWithBond = false
		return nil, nil
	case machine == "ce" && t == "parenthesisLevel++":
		buffer.ParenthesisLevel++
		return nil, nil
	case machine == "ce" && t == "parenthesisLevel--":
		buffer.ParenthesisLevel--
		return nil, nil
	case machine == "ce" && t == "state of aggregation":
		inner, err := ctx.go_(tokenString(m), "o")
		if err != nil {
			return nil, err
		}
		return []Value{map[string]any{"type_": "state of aggregation", "p1": inner}}, nil
	case machine == "ce" && t == "comma":
		return ceComma(buffer, m)
	case machine == "ce" && t == "oxidation-output":
		inner, err := ctx.go_(tokenString(m), "oxidation")
		if err != nil {
			return nil, err
		}
		v := []Value{"{"}
		v = append(v, inner...)
		v = append(v, "}")
		return v, nil
	case machine == "ce" && t == "frac-output":
		if !m.IsArray || len(m.A) < 2 {
			return nil, errMsg("frac-output")
		}
		p1, err := ctx.go_(m.A[0], "ce")
		if err != nil {
			return nil, err
		}
		p2, err := ctx.go_(m.A[1], "ce")
		if err != nil {
			return nil, err
		}
		return []Value{map[string]any{"type_": "frac-ce", "p1": p1, "p2": p2}}, nil
	case machine == "ce" && (t == "overset-output" || t == "underset-output" || t == "underbrace-output"):
		if !m.IsArray || len(m.A) < 2 {
			return nil, errMsg("two-arg output")
		}
		ty := "underbrace"
		switch t {
		case "overset-output":
			ty = "overset"
		case "underset-output":
			ty = "underset"
		}
		p1, err := ctx.go_(m.A[0], "ce")
		if err != nil {
			return nil, err
		}
		p2, err := ctx.go_(m.A[1], "ce")
		if err != nil {
			return nil, err
		}
		return []Value{map[string]any{"type_": ty, "p1": p1, "p2": p2}}, nil
	case machine == "ce" && t == "color-output":
		if !m.IsArray || len(m.A) < 2 {
			return nil, errMsg("color-output")
		}
		c2, err := ctx.go_(m.A[1], "ce")
		if err != nil {
			return nil, err
		}
		return []Value{map[string]any{"type_": "color", "color1": m.A[0], "color2": c2}}, nil
	case machine == "ce" && t == "operator":
		kind := optString(spec.Option)
		if kind == "" {
			kind = matchStr(m)
		}
		return []Value{map[string]any{"type_": "operator", "kind_": kind}}, nil

	case machine == "text" && t == "output":
		if buffer.Text != nil {
			tx := *buffer.Text
			buffer.ClearAll()
			return []Value{map[string]any{"type_": "text", "p1": tx}}, nil
		}
		return nil, nil

	case machine == "pq" && t == "state of aggregation":
		inner, err := ctx.go_(tokenString(m), "o")
		if err != nil {
			return nil, err
		}
		return []Value{map[string]any{"type_": "state of aggregation subscript", "p1": inner}}, nil
	case (machine == "pq" || machine == "bd") && t == "color-output":
		if !m.IsArray {
			return nil, errMsg("color bd/pq")
		}
		sm := "bd"
		if machine == "pq" {
			sm = "pq"
		}
		c2, err := ctx.go_(m.A[1], sm)
		if err != nil {
			return nil, err
		}
		return []Value{map[string]any{"type_": "color", "color1": m.A[0], "color2": c2}}, nil

	case machine == "oxidation" && t == "roman-numeral":
		return []Value{map[string]any{"type_": "roman numeral", "p1": matchStr(m)}}, nil

	case (machine == "tex-math" || machine == "tex-math tight") && t == "output":
		if buffer.O != nil {
			o := *buffer.O
			buffer.ClearAll()
			return []Value{map[string]any{"type_": "tex-math", "p1": o}}, nil
		}
		return nil, nil
	case machine == "tex-math tight" && t == "tight operator":
		op := matchStr(m)
		cur := ""
		if buffer.O != nil {
			cur = *buffer.O
		}
		v := cur + "{" + op + "}"
		buffer.O = &v
		return nil, nil

	case machine == "9,9" && t == "comma":
		return []Value{map[string]any{"type_": "commaDecimal"}}, nil

	case machine == "pu" && t == "enumber":
		return puEnumber(ctx, m)
	case machine == "pu" && t == "number^":
		return puNumberPow(ctx, m)
	case machine == "pu" && t == "space":
		return []Value{map[string]any{"type_": "pu-space-1"}}, nil
	case machine == "pu" && t == "output":
		return puOutput(ctx, buffer)

	case machine == "pu-2" && t == "cdot":
		return []Value{map[string]any{"type_": "tight cdot"}}, nil
	case machine == "pu-2" && t == "^(-1)":
		pow := matchStr(m)
		cur := ""
		if buffer.Rm != nil {
			cur = *buffer.Rm
		}
		v := cur + "^{" + pow + "}"
		buffer.Rm = &v
		return nil, nil
	case machine == "pu-2" && t == "space":
		return []Value{map[string]any{"type_": "pu-space-2"}}, nil
	case machine == "pu-2" && t == "output":
		return pu2Output(ctx, buffer)

	case machine == "pu-9,9" && t == "comma":
		return []Value{map[string]any{"type_": "commaDecimal"}}, nil
	case machine == "pu-9,9" && t == "output-0":
		return pu99Out(buffer, false)
	case machine == "pu-9,9" && t == "output-o":
		return pu99Out(buffer, true)
	}
	return globalAction(ctx, machine, buffer, m, spec)
}

func optString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

func optBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return false
	}
	return b
}

func optInt(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var i int
	if err := json.Unmarshal(raw, &i); err == nil {
		return i, true
	}
	return 0, false
}

func matchStr(m *MatchToken) string {
	if m == nil {
		return ""
	}
	if !m.IsArray {
		return m.S
	}
	if len(m.A) > 0 {
		return m.A[0]
	}
	return ""
}

func tokenString(m *MatchToken) string {
	if m == nil {
		return ""
	}
	if !m.IsArray {
		return m.S
	}
	return strings.Join(m.A, "")
}

func slotStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ceComma(buffer *Buffer, m *MatchToken) ([]Value, error) {
	raw := tokenString(m)
	trimmed := strings.TrimRight(raw, " \t\n")
	withSpace := trimmed != raw
	ty := "comma enumeration M"
	if withSpace && buffer.ParenthesisLevel == 0 {
		ty = "comma enumeration L"
	}
	return []Value{map[string]any{"type_": ty, "p1": trimmed}}, nil
}

func ceOAfterD(ctx *parserCtx, buffer *Buffer, m *MatchToken) ([]Value, error) {
	var ret []Value
	digitsOnly := buffer.D != nil && reDigitsOnly.MatchString(*buffer.D)
	if digitsOnly {
		tmp := *buffer.D
		buffer.D = nil
		out, err := ceOutput(ctx, buffer, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, out...)
		SetSlot(&buffer.B, tmp)
	} else {
		out, err := ceOutput(ctx, buffer, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, out...)
	}
	cat(&buffer.O, m)
	return ret, nil
}

func ceChargeOrBond(ctx *parserCtx, buffer *Buffer, m *MatchToken) ([]Value, error) {
	if buffer.BeginsWithBond {
		ret, err := ceOutput(ctx, buffer, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, map[string]any{"type_": "bond", "kind_": "-"})
		return ret, nil
	}
	SetSlot(&buffer.D, tokenString(m))
	return nil, nil
}

func ceAfterOD(ctx *parserCtx, buffer *Buffer, m *MatchToken, isAfterD bool) ([]Value, error) {
	dash := tokenString(m)
	o := slotStr(buffer.O)
	c1, _ := matchPattern(ctx.data, "orbital", o)
	c2, _ := matchPattern(ctx.data, "one lowercase greek letter $", o)
	c3, _ := matchPattern(ctx.data, "one lowercase latin letter $", o)
	c4, _ := matchPattern(ctx.data, "$one lowercase latin letter$ $", o)
	hOrb := c1 != nil && c1.Remainder == ""
	hyphenFollows := dash == "-" && (hOrb || c2 != nil || c3 != nil || c4 != nil)

	if hyphenFollows &&
		IsSlotEmpty(buffer.A) && IsSlotEmpty(buffer.B) && IsSlotEmpty(buffer.P) &&
		IsSlotEmpty(buffer.D) && IsSlotEmpty(buffer.Q) &&
		c1 == nil && c3 != nil {
		oo := slotStr(buffer.O)
		v := "$" + oo + "$"
		buffer.O = &v
	}

	var ret []Value
	if hyphenFollows {
		out, err := ceOutput(ctx, buffer, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, out...)
		ret = append(ret, map[string]any{"type_": "hyphen"})
		return ret, nil
	}

	digitsD, _ := matchPattern(ctx.data, "digits", slotStr(buffer.D))
	dOnly := digitsD != nil && digitsD.Remainder == ""
	if isAfterD && dOnly {
		cat(&buffer.D, m)
		out, err := ceOutput(ctx, buffer, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, out...)
	} else {
		out, err := ceOutput(ctx, buffer, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, out...)
		ret = append(ret, map[string]any{"type_": "bond", "kind_": "-"})
	}
	return ret, nil
}

func ceOutput(ctx *parserCtx, buffer *Buffer, entity json.RawMessage) ([]Value, error) {
	entityFollows, hasEntity := optInt(entity)

	if IsSlotEmpty(buffer.R) {
		emptyPiece := IsSlotEmpty(buffer.A) && IsSlotEmpty(buffer.B) && IsSlotEmpty(buffer.P) &&
			IsSlotEmpty(buffer.O) && IsSlotEmpty(buffer.Q) && IsSlotEmpty(buffer.D) && !hasEntity
		if emptyPiece {
			buffer.ClearSoft()
			return nil, nil
		}

		var ret []Value
		if buffer.Sb {
			ret = append(ret, map[string]any{"type_": "entitySkip"})
		}

		var dType *string
		if buffer.DType != nil {
			s := *buffer.DType
			dType = &s
		}
		if !IsSlotEmpty(buffer.O) && dType != nil && *dType == "kv" {
			oxh, _ := matchPattern(ctx.data, "d-oxidation$", slotStr(buffer.D))
			if oxh != nil {
				s := "oxidation"
				dType = &s
			}
		} else if !IsSlotEmpty(buffer.O) && dType != nil && *dType == "kv" && IsSlotEmpty(buffer.Q) {
			dType = nil
		}

		if IsSlotEmpty(buffer.O) && IsSlotEmpty(buffer.Q) && IsSlotEmpty(buffer.D) &&
			IsSlotEmpty(buffer.B) && IsSlotEmpty(buffer.P) &&
			!(hasEntity && entityFollows == 2) {
			buffer.O = buffer.A
			buffer.A = nil
		} else if IsSlotEmpty(buffer.O) && IsSlotEmpty(buffer.Q) && IsSlotEmpty(buffer.D) &&
			(!IsSlotEmpty(buffer.B) || !IsSlotEmpty(buffer.P)) {
			buffer.O = buffer.A
			buffer.A = nil
			buffer.D = buffer.B
			buffer.B = nil
			buffer.Q = buffer.P
			buffer.P = nil
		}

		a, err := ctx.go_(slotStr(buffer.A), "a")
		if err != nil {
			return nil, err
		}
		b, err := ctx.go_(slotStr(buffer.B), "bd")
		if err != nil {
			return nil, err
		}
		p, err := ctx.go_(slotStr(buffer.P), "pq")
		if err != nil {
			return nil, err
		}
		o, err := ctx.go_(slotStr(buffer.O), "o")
		if err != nil {
			return nil, err
		}
		q, err := ctx.go_(slotStr(buffer.Q), "pq")
		if err != nil {
			return nil, err
		}
		dSm := "bd"
		if dType != nil && *dType == "oxidation" {
			dSm = "oxidation"
		}
		d, err := ctx.go_(slotStr(buffer.D), dSm)
		if err != nil {
			return nil, err
		}
		buffer.A, buffer.B, buffer.P, buffer.O, buffer.Q, buffer.D = nil, nil, nil, nil, nil, nil

		chem := map[string]any{
			"type_": "chemfive",
			"a":     a,
			"b":     b,
			"p":     p,
			"o":     o,
			"q":     q,
			"d":     d,
		}
		if dType != nil {
			chem["dType"] = *dType
		}
		ret = append(ret, chem)
		buffer.ClearSoft()
		return ret, nil
	}

	r := slotStr(buffer.R)
	rdt := slotStr(buffer.Rdt)
	rd := slotStr(buffer.Rd)
	var rdAst []Value
	switch rdt {
	case "M":
		var err error
		rdAst, err = ctx.go_(rd, "tex-math")
		if err != nil {
			return nil, err
		}
	case "T":
		rdAst = []Value{map[string]any{"type_": "text", "p1": rd}}
	default:
		var err error
		rdAst, err = ctx.go_(rd, "ce")
		if err != nil {
			return nil, err
		}
	}

	rqt := slotStr(buffer.Rqt)
	rq := slotStr(buffer.Rq)
	var rqAst []Value
	switch rqt {
	case "M":
		var err error
		rqAst, err = ctx.go_(rq, "tex-math")
		if err != nil {
			return nil, err
		}
	case "T":
		rqAst = []Value{map[string]any{"type_": "text", "p1": rq}}
	default:
		var err error
		rqAst, err = ctx.go_(rq, "ce")
		if err != nil {
			return nil, err
		}
	}

	node := map[string]any{
		"type_": "arrow",
		"r":     r,
		"rd":    rdAst,
		"rq":    rqAst,
	}
	buffer.ClearSoft()
	return []Value{node}, nil
}

func puEnumber(ctx *parserCtx, m *MatchToken) ([]Value, error) {
	if !m.IsArray {
		return nil, nil
	}
	parts := m.A
	get := func(i int) string {
		if i < len(parts) {
			return parts[i]
		}
		return ""
	}
	var ret []Value
	g0 := get(0)
	if g0 == "+-" || g0 == "+/-" {
		ret = append(ret, "\\pm ")
	} else if g0 != "" {
		ret = append(ret, g0)
	}
	if get(1) != "" {
		v, err := ctx.go_(parts[1], "pu-9,9")
		if err != nil {
			return nil, err
		}
		ret = append(ret, v...)
		if g2 := get(2); g2 != "" {
			if strings.ContainsAny(g2, ",.") {
				v2, err := ctx.go_(g2, "pu-9,9")
				if err != nil {
					return nil, err
				}
				ret = append(ret, v2...)
			} else {
				ret = append(ret, g2)
			}
		}
		gStar := strings.TrimSpace(get(4))
		gE := strings.TrimSpace(get(3))
		mult := gStar
		if mult == "" {
			mult = gE
		}
		if mult != "" {
			if mult == "e" || strings.HasPrefix(mult, "*") {
				ret = append(ret, map[string]any{"type_": "cdot"})
			} else {
				ret = append(ret, map[string]any{"type_": "times"})
			}
		}
	}
	if exp := get(5); exp != "" {
		ret = append(ret, "10^{"+exp+"}")
	}
	return ret, nil
}

func puNumberPow(ctx *parserCtx, m *MatchToken) ([]Value, error) {
	if !m.IsArray {
		return nil, nil
	}
	parts := m.A
	var ret []Value
	get := func(i int) string {
		if i < len(parts) {
			return parts[i]
		}
		return ""
	}
	g0 := get(0)
	if g0 == "+-" || g0 == "+/-" {
		ret = append(ret, "\\pm ")
	} else if g0 != "" {
		ret = append(ret, g0)
	}
	if base := get(1); base != "" || len(parts) > 1 {
		v, err := ctx.go_(base, "pu-9,9")
		if err != nil {
			return nil, err
		}
		ret = append(ret, v...)
	}
	if exp := get(2); exp != "" || len(parts) > 2 {
		ret = append(ret, "^{"+exp+"}")
	}
	return ret, nil
}

func puOutput(ctx *parserCtx, buffer *Buffer) ([]Value, error) {
	d := slotStr(buffer.D)
	if md, _ := matchPattern(ctx.data, "{(...)}", d); md != nil && md.Remainder == "" && !md.Token.IsArray {
		d = md.Token.S
	}
	qv := slotStr(buffer.Q)
	if mq, _ := matchPattern(ctx.data, "{(...)}", qv); mq != nil && mq.Remainder == "" && !mq.Token.IsArray {
		qv = mq.Token.S
	}
	d = reCelsiusC.ReplaceAllString(d, "{}^{\\circ}C")
	qv = reCelsiusC.ReplaceAllString(qv, "{}^{\\circ}C")
	d = reCelsiusF.ReplaceAllString(d, "{}^{\\circ}F")
	qv = reCelsiusF.ReplaceAllString(qv, "{}^{\\circ}F")

	var res []Value
	if qv != "" {
		b5d, err := ctx.go_(d, "pu")
		if err != nil {
			return nil, err
		}
		b5q, err := ctx.go_(qv, "pu")
		if err != nil {
			return nil, err
		}
		if buffer.O != nil && *buffer.O == "//" {
			res = []Value{map[string]any{"type_": "pu-frac", "p1": b5d, "p2": b5q}}
		} else {
			res = b5d
			if len(b5d) > 1 || len(b5q) > 1 {
				res = append(res, map[string]any{"type_": " / "})
			} else {
				res = append(res, map[string]any{"type_": "/"})
			}
			res = append(res, b5q...)
		}
	} else {
		v, err := ctx.go_(d, "pu-2")
		if err != nil {
			return nil, err
		}
		res = v
	}
	buffer.ClearAll()
	return res, nil
}

func pu2Output(ctx *parserCtx, buffer *Buffer) ([]Value, error) {
	var res []Value
	if buffer.Rm != nil {
		rm := *buffer.Rm
		buffer.Rm = nil
		mrm, _ := matchPattern(ctx.data, "{(...)}", rm)
		if mrm != nil && mrm.Remainder == "" {
			if !mrm.Token.IsArray {
				inner, err := ctx.go_(mrm.Token.S, "pu")
				if err != nil {
					return nil, err
				}
				res = inner
			}
		} else {
			res = []Value{map[string]any{"type_": "rm", "p1": rm}}
		}
	}
	buffer.ClearAll()
	return res, nil
}

func pu99Out(buffer *Buffer, reverseTriplets bool) ([]Value, error) {
	t := slotStr(buffer.Text)
	buffer.Text = nil
	var ret []Value
	if len(t) > 4 {
		if reverseTriplets {
			a := len(t) % 3
			if a == 0 {
				a = 3
			}
			i := len(t) - 3
			for i > 0 {
				ret = append(ret, t[i:i+3])
				ret = append(ret, map[string]any{"type_": "1000 separator"})
				i -= 3
			}
			ret = append(ret, t[:a])
			// reverse
			for l, r := 0, len(ret)-1; l < r; l, r = l+1, r-1 {
				ret[l], ret[r] = ret[r], ret[l]
			}
		} else {
			a := len(t) - 3
			i := 0
			for i < a {
				ret = append(ret, t[i:i+3])
				ret = append(ret, map[string]any{"type_": "1000 separator"})
				i += 3
			}
			ret = append(ret, t[i:])
		}
	} else {
		ret = []Value{t}
	}
	buffer.ClearAll()
	return ret, nil
}

func globalAction(ctx *parserCtx, machine string, buffer *Buffer, m *MatchToken, spec *ActionSpec) ([]Value, error) {
	switch spec.Type {
	case "a=":
		cat(&buffer.A, m)
		return nil, nil
	case "b=":
		cat(&buffer.B, m)
		return nil, nil
	case "p=":
		cat(&buffer.P, m)
		return nil, nil
	case "o=":
		cat(&buffer.O, m)
		return nil, nil
	case "q=":
		cat(&buffer.Q, m)
		return nil, nil
	case "d=":
		cat(&buffer.D, m)
		return nil, nil
	case "rm=":
		cat(&buffer.Rm, m)
		return nil, nil
	case "text=":
		cat(&buffer.Text, m)
		return nil, nil
	case "r=":
		SetSlot(&buffer.R, tokenString(m))
		return nil, nil
	case "rdt=":
		SetSlot(&buffer.Rdt, tokenString(m))
		return nil, nil
	case "rqt=":
		SetSlot(&buffer.Rqt, tokenString(m))
		return nil, nil
	case "rd=":
		SetSlot(&buffer.Rd, tokenString(m))
		return nil, nil
	case "rq=":
		SetSlot(&buffer.Rq, tokenString(m))
		return nil, nil
	case "insert":
		opt := optString(spec.Option)
		return []Value{map[string]any{"type_": opt}}, nil
	case "insert+p1":
		return []Value{map[string]any{"type_": optString(spec.Option), "p1": matchStr(m)}}, nil
	case "insert+p1+p2":
		if !m.IsArray || len(m.A) < 2 {
			return nil, errMsg("insert+p1+p2")
		}
		return []Value{map[string]any{
			"type_": optString(spec.Option),
			"p1":    m.A[0],
			"p2":    m.A[1],
		}}, nil
	case "copy":
		return []Value{tokenString(m)}, nil
	case "rm":
		return []Value{map[string]any{"type_": "rm", "p1": matchStr(m)}}, nil
	case "text":
		return ctx.go_(tokenString(m), "text")
	case "{text}":
		v := []Value{"{"}
		out, err := ctx.go_(tokenString(m), "text")
		if err != nil {
			return nil, err
		}
		v = append(v, out...)
		v = append(v, "}")
		return v, nil
	case "tex-math":
		return ctx.go_(tokenString(m), "tex-math")
	case "tex-math tight":
		return ctx.go_(tokenString(m), "tex-math tight")
	case "bond":
		kind := optString(spec.Option)
		if kind == "" {
			kind = matchStr(m)
		}
		if kind == "" {
			kind = "-"
		}
		return []Value{map[string]any{"type_": "bond", "kind_": kind}}, nil
	case "color0-output":
		if !m.IsArray {
			return nil, errMsg("color0-output")
		}
		c := ""
		if len(m.A) > 0 {
			c = m.A[0]
		}
		return []Value{map[string]any{"type_": "color0", "color": c}}, nil
	case "ce":
		return ctx.go_(tokenString(m), "ce")
	case "1/2":
		return halfAction(m)
	case "9,9":
		return ctx.go_(tokenString(m), "9,9")
	case "operator":
		kind := optString(spec.Option)
		if kind == "" {
			kind = matchStr(m)
		}
		return []Value{map[string]any{"type_": "operator", "kind_": kind}}, nil
	}
	return nil, errMsg("mhchem bug A: %s / %s", machine, spec.Type)
}

func cat(slot **string, m *MatchToken) {
	t := tokenString(m)
	if *slot == nil {
		s := t
		*slot = &s
		return
	}
	v := **slot + t
	*slot = &v
}

func halfAction(m *MatchToken) ([]Value, error) {
	s := tokenString(m)
	var ret []Value
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		ret = append(ret, s[:1])
		s = s[1:]
	}
	mm := reHalf.FindStringSubmatch(s)
	if mm == nil {
		return nil, nil
	}
	n1 := strings.ReplaceAll(mm[1], "$", "")
	n2 := mm[2]
	ret = append(ret, map[string]any{"type_": "frac", "p1": n1, "p2": n2})
	if len(mm) > 3 && mm[3] != "" {
		t3 := strings.ReplaceAll(mm[3], "$", "")
		ret = append(ret, map[string]any{"type_": "tex-math", "p1": t3})
	}
	return ret, nil
}
