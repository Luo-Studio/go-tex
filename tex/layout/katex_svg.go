package layout

import (
	"strconv"
	"strings"

	"github.com/luo-studio/go-tex/tex/path"
)

// This file ports upstream's katex_svg.rs — the KaTeX SVG path data
// machinery used for stretchy arrows (\xrightarrow et al.), wide
// accents (\widehat, \widetilde, \overgroup, …), \overbrace /
// \underbrace, the synthetic \vec arrow, and tall vertical delimiters.

// =============================================================================
// SVG path parser
// =============================================================================

type svgToken struct {
	isCmd bool
	cmd   byte
	num   float64
}

func tokenizeSvg(d string) []svgToken {
	var tokens []svgToken
	bytes := []byte(d)
	i := 0
	for i < len(bytes) {
		b := bytes[i]
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
			tokens = append(tokens, svgToken{isCmd: true, cmd: b})
			i++
			continue
		}
		if b == '-' || b == '.' || (b >= '0' && b <= '9') {
			start := i
			if b == '-' {
				i++
			}
			hasDot := false
			for i < len(bytes) {
				c := bytes[i]
				if c >= '0' && c <= '9' {
					i++
				} else if c == '.' && !hasDot {
					hasDot = true
					i++
				} else {
					break
				}
			}
			if n, err := strconv.ParseFloat(string(bytes[start:i]), 64); err == nil {
				tokens = append(tokens, svgToken{num: n})
			}
			continue
		}
		i++
	}
	return tokens
}

func svgRead1(tokens []svgToken, i *int) float64 {
	if *i < len(tokens) && !tokens[*i].isCmd {
		v := tokens[*i].num
		*i++
		return v
	}
	return 0
}

func svgRead2(tokens []svgToken, i *int) (float64, float64) {
	a := svgRead1(tokens, i)
	b := svgRead1(tokens, i)
	return a, b
}

// parseSVGPath parses an SVG-style "d" string into path commands.
// Supports M/m/L/l/H/h/V/v/C/c/S/s/Q/q/Z/z.
func parseSVGPath(d string) []path.Command {
	tokens := tokenizeSvg(d)
	var cmds []path.Command
	i := 0
	cx, cy := 0.0, 0.0
	subpathStart := [2]float64{0, 0}
	lastCmd := byte('M')
	for i < len(tokens) {
		var cmdByte byte
		if tokens[i].isCmd {
			cmdByte = tokens[i].cmd
			i++
		} else {
			cmdByte = lastCmd
		}
		switch cmdByte {
		case 'M':
			x, y := svgRead2(tokens, &i)
			cx, cy = x, y
			subpathStart = [2]float64{cx, cy}
			cmds = append(cmds, path.MoveTo(x, y))
			lastCmd = 'L'
		case 'm':
			dx, dy := svgRead2(tokens, &i)
			cx += dx
			cy += dy
			subpathStart = [2]float64{cx, cy}
			cmds = append(cmds, path.MoveTo(cx, cy))
			lastCmd = 'l'
		case 'L':
			x, y := svgRead2(tokens, &i)
			cx, cy = x, y
			cmds = append(cmds, path.LineTo(x, y))
			lastCmd = 'L'
		case 'l':
			dx, dy := svgRead2(tokens, &i)
			cx += dx
			cy += dy
			cmds = append(cmds, path.LineTo(cx, cy))
			lastCmd = 'l'
		case 'H':
			x := svgRead1(tokens, &i)
			cx = x
			cmds = append(cmds, path.LineTo(cx, cy))
			lastCmd = 'H'
		case 'h':
			dx := svgRead1(tokens, &i)
			cx += dx
			cmds = append(cmds, path.LineTo(cx, cy))
			lastCmd = 'h'
		case 'V':
			y := svgRead1(tokens, &i)
			cy = y
			cmds = append(cmds, path.LineTo(cx, cy))
			lastCmd = 'V'
		case 'v':
			dy := svgRead1(tokens, &i)
			cy += dy
			cmds = append(cmds, path.LineTo(cx, cy))
			lastCmd = 'v'
		case 'C':
			x1, y1 := svgRead2(tokens, &i)
			x2, y2 := svgRead2(tokens, &i)
			x, y := svgRead2(tokens, &i)
			cx, cy = x, y
			cmds = append(cmds, path.CubicTo(x1, y1, x2, y2, x, y))
			lastCmd = 'C'
		case 'c':
			dx1, dy1 := svgRead2(tokens, &i)
			dx2, dy2 := svgRead2(tokens, &i)
			dx, dy := svgRead2(tokens, &i)
			x1, y1 := cx+dx1, cy+dy1
			x2, y2 := cx+dx2, cy+dy2
			cx += dx
			cy += dy
			cmds = append(cmds, path.CubicTo(x1, y1, x2, y2, cx, cy))
			lastCmd = 'c'
		case 'S':
			x2, y2 := svgRead2(tokens, &i)
			x, y := svgRead2(tokens, &i)
			x1, y1 := cx, cy
			cx, cy = x, y
			cmds = append(cmds, path.CubicTo(x1, y1, x2, y2, x, y))
			lastCmd = 'S'
		case 's':
			dx2, dy2 := svgRead2(tokens, &i)
			dx, dy := svgRead2(tokens, &i)
			x1, y1 := cx, cy
			x2, y2 := cx+dx2, cy+dy2
			cx += dx
			cy += dy
			cmds = append(cmds, path.CubicTo(x1, y1, x2, y2, cx, cy))
			lastCmd = 's'
		case 'Q':
			x1, y1 := svgRead2(tokens, &i)
			x, y := svgRead2(tokens, &i)
			cx, cy = x, y
			cmds = append(cmds, path.QuadTo(x1, y1, x, y))
			lastCmd = 'Q'
		case 'q':
			dx1, dy1 := svgRead2(tokens, &i)
			dx, dy := svgRead2(tokens, &i)
			x1, y1 := cx+dx1, cy+dy1
			cx += dx
			cy += dy
			cmds = append(cmds, path.QuadTo(x1, y1, cx, cy))
			lastCmd = 'q'
		case 'Z', 'z':
			cmds = append(cmds, path.Command{Kind: path.KindClose})
			cx, cy = subpathStart[0], subpathStart[1]
			lastCmd = 'M'
		default:
			// skip unknown command byte
		}
	}
	return cmds
}

// =============================================================================
// Path scaling helpers
// =============================================================================

func scaleSVGPathThousandths(cmds []path.Command) []path.Command {
	const s = 0.001
	return scaleSVGPathXY(cmds, s, s)
}

func scaleSVGPathXY(cmds []path.Command, sx, sy float64) []path.Command {
	out := make([]path.Command, len(cmds))
	for i, c := range cmds {
		out[i] = c
		switch c.Kind {
		case path.KindMoveTo, path.KindLineTo:
			out[i].X = c.X * sx
			out[i].Y = c.Y * sy
		case path.KindCubicTo:
			out[i].X1 = c.X1 * sx
			out[i].Y1 = c.Y1 * sy
			out[i].X2 = c.X2 * sx
			out[i].Y2 = c.Y2 * sy
			out[i].X = c.X * sx
			out[i].Y = c.Y * sy
		case path.KindQuadTo:
			out[i].X1 = c.X1 * sx
			out[i].Y1 = c.Y1 * sy
			out[i].X = c.X * sx
			out[i].Y = c.Y * sy
		}
	}
	return out
}

// scaleCmdTwoheadUniform scales by `s`, shifts y by -vbCy (centering on
// midline), and shifts x by xShift. Used for stretchy paths.
func scaleCmdTwoheadUniform(cmd path.Command, s, vbCy, xShift float64) path.Command {
	out := cmd
	switch cmd.Kind {
	case path.KindMoveTo, path.KindLineTo:
		out.X = cmd.X*s + xShift
		out.Y = (cmd.Y - vbCy) * s
	case path.KindCubicTo:
		out.X1 = cmd.X1*s + xShift
		out.Y1 = (cmd.Y1 - vbCy) * s
		out.X2 = cmd.X2*s + xShift
		out.Y2 = (cmd.Y2 - vbCy) * s
		out.X = cmd.X*s + xShift
		out.Y = (cmd.Y - vbCy) * s
	case path.KindQuadTo:
		out.X1 = cmd.X1*s + xShift
		out.Y1 = (cmd.Y1 - vbCy) * s
		out.X = cmd.X*s + xShift
		out.Y = (cmd.Y - vbCy) * s
	}
	return out
}

// =============================================================================
// Path clipping (Liang-Barsky on flattened contours)
// =============================================================================

func flattenPathToContours(commands []path.Command) [][][2]float64 {
	var contours [][][2]float64
	var current [][2]float64
	var last [2]float64
	const N = 16
	flush := func() {
		if len(current) > 0 {
			contours = append(contours, current)
			current = nil
		}
	}
	for _, cmd := range commands {
		switch cmd.Kind {
		case path.KindMoveTo:
			flush()
			last = [2]float64{cmd.X, cmd.Y}
			current = append(current, last)
		case path.KindLineTo:
			last = [2]float64{cmd.X, cmd.Y}
			current = append(current, last)
		case path.KindCubicTo:
			x0, y0 := last[0], last[1]
			x1, y1 := cmd.X1, cmd.Y1
			x2, y2 := cmd.X2, cmd.Y2
			x, y := cmd.X, cmd.Y
			for k := 1; k <= N; k++ {
				t := float64(k) / float64(N)
				u := 1 - t
				bx := u*u*u*x0 + 3*u*u*t*x1 + 3*u*t*t*x2 + t*t*t*x
				by := u*u*u*y0 + 3*u*u*t*y1 + 3*u*t*t*y2 + t*t*t*y
				last = [2]float64{bx, by}
				current = append(current, last)
			}
		case path.KindQuadTo:
			x0, y0 := last[0], last[1]
			x1, y1 := cmd.X1, cmd.Y1
			x, y := cmd.X, cmd.Y
			for k := 1; k <= N; k++ {
				t := float64(k) / float64(N)
				u := 1 - t
				bx := u*u*x0 + 2*u*t*x1 + t*t*x
				by := u*u*y0 + 2*u*t*y1 + t*t*y
				last = [2]float64{bx, by}
				current = append(current, last)
			}
		case path.KindClose:
			flush()
		}
	}
	flush()
	return contours
}

func clipSegmentToRect(p0, p1 [2]float64, xMin, xMax, yMin, yMax float64) ([2]float64, [2]float64, bool) {
	x0, y0 := p0[0], p0[1]
	x1, y1 := p1[0], p1[1]
	dx, dy := x1-x0, y1-y0
	t0, t1 := 0.0, 1.0
	clip := func(p, q float64) bool {
		if p < 0 {
			t := q / p
			if t > t1 {
				return false
			}
			if t > t0 {
				t0 = t
			}
		} else if p > 0 {
			t := q / p
			if t < t0 {
				return false
			}
			if t < t1 {
				t1 = t
			}
		} else {
			return q >= 0
		}
		return true
	}
	if !clip(-dx, x0-xMin) {
		return p0, p1, false
	}
	if !clip(dx, xMax-x0) {
		return p0, p1, false
	}
	if !clip(-dy, y0-yMin) {
		return p0, p1, false
	}
	if !clip(dy, yMax-y0) {
		return p0, p1, false
	}
	if t0 >= t1-1e-9 {
		return p0, p1, false
	}
	a := [2]float64{x0 + t0*dx, y0 + t0*dy}
	b := [2]float64{x0 + t1*dx, y0 + t1*dy}
	return a, b, true
}

func clipPathToRect(commands []path.Command, xMin, xMax, yMin, yMax float64) []path.Command {
	contours := flattenPathToContours(commands)
	var out []path.Command
	for _, contour := range contours {
		if len(contour) < 2 {
			continue
		}
		needMove := true
		var lastPt [2]float64
		n := len(contour)
		for i := 0; i < n; i++ {
			p0 := contour[i]
			p1 := contour[(i+1)%n]
			a, b, ok := clipSegmentToRect(p0, p1, xMin, xMax, yMin, yMax)
			if !ok {
				continue
			}
			if needMove {
				out = append(out, path.MoveTo(a[0], a[1]))
				needMove = false
			} else if absF(a[0]-lastPt[0]) > 1e-9 || absF(a[1]-lastPt[1]) > 1e-9 {
				out = append(out, path.LineTo(a[0], a[1]))
			}
			out = append(out, path.LineTo(b[0], b[1]))
			lastPt = b
		}
	}
	return out
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// =============================================================================
// KaTeX SVG path data (from svgGeometry.js)
// =============================================================================

const (
	katexVecPath          = `M377 20c0-5.333 1.833-10 5.5-14S391 0 397 0c4.667 0 8.667 1.667 12 5 3.333 2.667 6.667 9 10 19 6.667 24.667 20.333 43.667 41 57 7.333 4.667 11 10.667 11 18 0 6-1 10-3 12s-6.667 5-14 9c-28.667 14.667-53.667 35.667-75 63 -1.333 1.333-3.167 3.5-5.5 6.5s-4 4.833-5 5.5c-1 .667-2.5 1.333-4.5 2s-4.333 1 -7 1c-4.667 0-9.167-1.833-13.5-5.5S337 184 337 178c0-12.667 15.667-32.333 47-59 H213l-171-1c-8.667-6-13-12.333-13-19 0-4.667 4.333-11.333 13-20h359 c-16-25.333-24-45-24-59z`
	twoheadleftarrowPath  = `M0 167c68 40 115.7 95.7 143 167h22c15.3 0 23-.3 23-1 0-1.3-5.3-13.7-16-37-18-35.3-41.3-69-70-101l-7-8h125l9 7c50.7 39.3 85 86 103 140h46c0-4.7-6.3-18.7-19-42-18-35.3-40-67.3-66-96l-9-9h399716v-40H284l9-9c26-28.7 48-60.7 66-96 12.7-23.333 19-37.333 19-42h-46c-18 54-52.3 100.7-103 140l-9 7H95l7-8c28.7-32 52-65.7 70-101 10.7-23.333 16-35.7 16-37 0-.7-7.7-1-23-1h-22C115.7 71.3 68 127 0 167z`
	rightarrowPath        = `M0 241v40h399891c-47.3 35.3-84 78-110 128 -16.7 32-27.7 63.7-33 95 0 1.3-.2 2.7-.5 4-.3 1.3-.5 2.3-.5 3 0 7.3 6.7 11 20 11 8 0 13.2-.8 15.5-2.5 2.3-1.7 4.2-5.5 5.5-11.5 2-13.3 5.7-27 11-41 14.7-44.7 39-84.5 73-119.5s73.7-60.2 119-75.5c6-2 9-5.7 9-11s-3-9-9-11c-45.3-15.3-85 -40.5-119-75.5s-58.3-74.8-73-119.5c-4.7-14-8.3-27.3-11-40-1.3-6.7-3.2-10.8-5.5 -12.5-2.3-1.7-7.5-2.5-15.5-2.5-14 0-21 3.7-21 11 0 2 2 10.3 6 25 20.7 83.3 67 151.7 139 205zm0 0v40h399900v-40z`
	leftarrowPath         = `M400000 241H110l3-3c68.7-52.7 113.7-120 135-202 4-14.7 6-23 6-25 0-7.3-7-11-21-11-8 0-13.2.8-15.5 2.5-2.3 1.7-4.2 5.8 -5.5 12.5-1.3 4.7-2.7 10.3-4 17-12 48.7-34.8 92-68.5 130S65.3 228.3 18 247 c-10 4-16 7.7-18 11 0 8.7 6 14.3 18 17 47.3 18.7 87.8 47 121.5 85S196 441.3 208 490c.7 2 1.3 5 2 9s1.2 6.7 1.5 8c.3 1.3 1 3.3 2 6s2.2 4.5 3.5 5.5c1.3 1 3.3 1.8 6 2.5s6 1 10 1c14 0 21-3.7 21-11 0-2-2-10.3-6-25-20-79.3-65-146.7-135-202 l-3-3h399890zM100 241v40h399900v-40z`
	leftmapstoPath        = `M40 281 V448H0V74H40V241H400000v40z`
	twoheadrightarrowPath = `M400000 167c-68-40-115.7-95.7-143-167h-22c-15.3 0-23 .3-23 1 0 1.3 5.3 13.7 16 37 18 35.3 41.3 69 70 101l7 8h-125l-9-7c-50.7-39.3-85-86-103-140h-46c0 4.7 6.3 18.7 19 42 18 35.3 40 67.3 66 96l9 9H0v40h399716l-9 9c-26 28.7-48 60.7-66 96-12.7 23.333-19 37.333-19 42h46c18-54 52.3-100.7 103-140l9-7h125l-7 8c-28.7 32-52 65.7-70 101-10.7 23.333-16 35.7-16 37 0 .7 7.7 1 23 1h22c27.3-71.3 75-127 143-167z`
	doubleleftarrowPath   = `M262 157 l10-10c34-36 62.7-77 86-123 3.3-8 5-13.3 5-16 0-5.3-6.7-8-20-8-7.3 0-12.2.5-14.5 1.5-2.3 1-4.8 4.5-7.5 10.5-49.3 97.3-121.7 169.3-217 216-28 14-57.3 25-88 33-6.7 2-11 3.8-13 5.5-2 1.7-3 4.2-3 7.5s1 5.8 3 7.5 c2 1.7 6.3 3.5 13 5.5 68 17.3 128.2 47.8 180.5 91.5 52.3 43.7 93.8 96.2 124.5 157.5 9.3 8 15.3 12.3 18 13h6c12-.7 18-4 18-10 0-2-1.7-7-5-15-23.3-46-52-87-86-123l-10-10h399738v-40H218c328 0 0 0 0 0l-10-8c-26.7-20-65.7-43-117-69 2.7-2 6-3.7 10-5 36.7-16 72.3-37.3 107-64l10-8h399782v-40z m8 0v40h399730v-40zm0 194v40h399730v-40z`
	doublerightarrowPath  = `M399738 392l -10 10c-34 36-62.7 77-86 123-3.3 8-5 13.3-5 16 0 5.3 6.7 8 20 8 7.3 0 12.2-.5 14.5-1.5 2.3-1 4.8-4.5 7.5-10.5 49.3-97.3 121.7-169.3 217-216 28-14 57.3-25 88-33 6.7-2 11-3.8 13-5.5 2-1.7 3-4.2 3-7.5s-1-5.8-3-7.5c-2-1.7-6.3-3.5-13-5.5-68-17.3-128.2-47.8-180.5-91.5-52.3-43.7-93.8-96.2-124.5-157.5-9.3-8-15.3-12.3-18-13h-6c-12 .7-18 4-18 10 0 2 1.7 7 5 15 23.3 46 52 87 86 123l10 10H0v40h399782 c-328 0 0 0 0 0l10 8c26.7 20 65.7 43 117 69-2.7 2-6 3.7-10 5-36.7 16-72.3 37.3-107 64l-10 8H0v40zM0 157v40h399730v-40zm0 194v40h399730v-40z`
	leftharpoonPath       = `M0 267c.7 5.3 3 10 7 14h399993v-40H93c3.3-3.3 10.2-9.5 20.5-18.5s17.8-15.8 22.5-20.5c50.7-52 88-110.3 112-175 4-11.3 5-18.3 3-21-1.3-4-7.3-6-18-6-8 0-13 .7-15 2s-4.7 6.7-8 16c-42 98.7-107.3 174.7-196 228-6.7 4.7-10.7 8-12 10-1.3 2-2 5.7-2 11zm100-26v40h399900v-40z`
	leftharpoonplusPath   = `M0 267c.7 5.3 3 10 7 14h399993v-40H93c3.3-3.3 10.2-9.5 20.5-18.5s17.8-15.8 22.5-20.5c50.7-52 88-110.3 112-175 4-11.3 5-18.3 3-21-1.3-4-7.3-6-18-6-8 0-13 .7-15 2s-4.7 6.7-8 16c-42 98.7-107.3 174.7-196 228-6.7 4.7-10.7 8-12 10-1.3 2-2 5.7-2 11zm100-26v40h399900v-40zM0 435v40h400000v-40z m0 0v40h400000v-40z`
	leftharpoondownPath   = `M7 241c-4 4-6.333 8.667-7 14 0 5.333.667 9 2 11s5.333 5.333 12 10c90.667 54 156 130 196 228 3.333 10.667 6.333 16.333 9 17 2 .667 5 1 9 1h5c10.667 0 16.667-2 18-6 2-2.667 1-9.667-3-21-32-87.333-82.667-157.667-152-211l-3-3h399907v-40zM93 281 H400000 v-40L7 241z`
	leftharpoondownplusPath = `M7 435c-4 4-6.3 8.7-7 14 0 5.3.7 9 2 11s5.3 5.3 12 10c90.7 54 156 130 196 228 3.3 10.7 6.3 16.3 9 17 2 .7 5 1 9 1h5c10.7 0 16.7-2 18-6 2-2.7 1-9.7-3-21-32-87.3-82.7-157.7-152-211l-3-3h399907v-40H7zm93 0 v40h399900v-40zM0 241v40h399900v-40zm0 0v40h399900v-40z`
	rightharpoonPath      = `M0 241v40h399993c4.7-4.7 7-9.3 7-14 0-9.3-3.7-15.3-11-18-92.7-56.7-159-133.7-199-231-3.3-9.3-6-14.7-8-16-2-1.3-7-2-15-2-10.7 0-16.7 2-18 6-2 2.7-1 9.7 3 21 15.3 42 36.7 81.8 64 119.5 27.3 37.7 58 69.2 92 94.5zm0 0v40h399900v-40z`
	rightharpoonplusPath  = `M0 241v40h399993c4.7-4.7 7-9.3 7-14 0-9.3-3.7-15.3-11-18-92.7-56.7-159-133.7-199-231-3.3-9.3-6-14.7-8-16-2-1.3-7-2-15-2-10.7 0-16.7 2-18 6-2 2.7-1 9.7 3 21 15.3 42 36.7 81.8 64 119.5 27.3 37.7 58 69.2 92 94.5z m0 0v40h399900v-40z m100 194v40h399900v-40zm0 0v40h399900v-40z`
	rightharpoondownPath  = `M399747 511c0 7.3 6.7 11 20 11 8 0 13-.8 15-2.5s4.7-6.8 8-15.5c40-94 99.3-166.3 178-217 13.3-8 20.3-12.3 21-13 5.3-3.3 8.5-5.8 9.5-7.5 1-1.7 1.5-5.2 1.5-10.5s-2.3-10.3-7-15H0v40h399908c-34 25.3-64.7 57-92 95-27.3 38-48.7 77.7-64 119-3.3 8.7-5 14-5 16zM0 241v40h399900v-40z`
	rightharpoondownplusPath = `M399747 705c0 7.3 6.7 11 20 11 8 0 13-.8 15-2.5s4.7-6.8 8-15.5c40-94 99.3-166.3 178-217 13.3-8 20.3-12.3 21-13 5.3-3.3 8.5-5.8 9.5-7.5 1-1.7 1.5-5.2 1.5-10.5s-2.3-10.3-7-15H0v40h399908c-34 25.3-64.7 57-92 95-27.3 38-48.7 77.7-64 119-3.3 8.7-5 14-5 16zM0 435v40h399900v-40z m0-194v40h400000v-40zm0 0v40h400000v-40z`
	lefthookPath          = `M400000 281 H103s-33-11.2-61-33.5S0 197.3 0 164s14.2-61.2 42.5-83.5C70.8 58.2 104 47 142 47 c16.7 0 25 6.7 25 20 0 12-8.7 18.7-26 20-40 3.3-68.7 15.7-86 37-10 12-15 25.3-15 40 0 22.7 9.8 40.7 29.5 54 19.7 13.3 43.5 21 71.5 23h399859zM103 281v-40h399897v40z`
	righthookPath         = `M399859 241c-764 0 0 0 0 0 40-3.3 68.7-15.7 86-37 10-12 15-25.3 15-40 0-22.7-9.8-40.7-29.5-54-19.7-13.3-43.5-21-71.5-23-17.3-1.3-26-8-26-20 0-13.3 8.7-20 26-20 38 0 71 11.2 99 33.5 0 0 7 5.6 21 16.7 14 11.2 21 33.5 21 66.8s-14 61.2-42 83.5c-28 22.3-61 33.5-99 33.5L0 241z M0 281v-40h399859v40z`
	leftbracePath         = `M6 548l-6-6v-35l6-11c56-104 135.3-181.3 238-232 57.3-28.7 117-45 179-50h399577v120H403c-43.3 7-81 15-113 26-100.7 33-179.7 91-237 174-2.7 5-6 9-10 13-.7 1-7.3 1-20 1H6z`
	midbracePath          = `M200428 334 c-100.7-8.3-195.3-44-280-108-55.3-42-101.7-93-139-153l-9-14c-2.7 4-5.7 8.7-9 14-53.3 86.7-123.7 153-211 199-66.7 36-137.3 56.3-212 62H0V214h199568c178.3-11.7 311.7-78.3 403-201 6-8 9.7-12 11-12 .7-.7 6.7-1 18-1s17.3.3 18 1c1.3 0 5 4 11 12 44.7 59.3 101.3 106.3 170 141s145.3 54.3 229 60h199572v120z`
	rightbracePath        = `M400000 542l -6 6h-17c-12.7 0-19.3-.3-20-1-4-4-7.3-8.3-10-13-35.3-51.3-80.8-93.8-136.5-127.5s-117.2-55.8-184.5-66.5c-.7 0-2-.3-4-1-18.7-2.7-76-4.3-172-5H0V214h399571l6 1 c124.7 8 235 61.7 331 161 31.3 33.3 59.7 72.7 85 118l7 13v35z`
	leftbraceunderPath    = `M0 6l6-6h17c12.688 0 19.313.3 20 1 4 4 7.313 8.3 10 13 35.313 51.3 80.813 93.8 136.5 127.5 55.688 33.7 117.188 55.8 184.5 66.5.688 0 2 .3 4 1 18.688 2.7 76 4.3 172 5h399450v120H429l-6-1c-124.688-8-235-61.7-331-161C60.687 138.7 32.312 99.3 7 54L0 41V6z`
	midbraceunderPath     = `M199572 214 c100.7 8.3 195.3 44 280 108 55.3 42 101.7 93 139 153l9 14c2.7-4 5.7-8.7 9-14 53.3-86.7 123.7-153 211-199 66.7-36 137.3-56.3 212-62h199568v120H200432c-178.3 11.7-311.7 78.3-403 201-6 8-9.7 12-11 12-.7.7-6.7 1-18 1s-17.3-.3-18-1c-1.3 0-5-4-11-12-44.7-59.3-101.3-106.3-170-141s-145.3-54.3-229-60H0V214z`
	rightbraceunderPath   = `M399994 0l6 6v35l-6 11c-56 104-135.3 181.3-238 232-57.3 28.7-117 45-179 50H-300V214h399897c43.3-7 81-15 113-26 100.7-33 179.7-91 237-174 2.7-5 6-9 10-13 .7-1 7.3-1 20-1h17z`
	leftToFromPath        = `M0 147h400000v40H0zm0 214c68 40 115.7 95.7 143 167h22c15.3 0 23-.3 23-1 0-1.3-5.3-13.7-16-37-18-35.3-41.3-69-70-101l-7-8h399905v-40H95l7-8 c28.7-32 52-65.7 70-101 10.7-23.3 16-35.7 16-37 0-.7-7.7-1-23-1h-22C115.7 265.3 68 321 0 361zm0-174v-40h399900v40zm100 154v40h399900v-40z`
	rightToFromPath       = `M400000 167c-70.7-42-118-97.7-142-167h-23c-15.3 0-23 .3-23 1 0 1.3 5.3 13.7 16 37 18 35.3 41.3 69 70 101l7 8H0v40h399905l-7 8c-28.7 32-52 65.7-70 101-10.7 23.3-16 35.7-16 37 0 .7 7.7 1 23 1h23c24-69.3 71.3-125 142-167z M100 147v40h399900v-40zM0 341v40h399900v-40z`
	longequalPath         = `M0 50 h400000 v40H0z m0 194h400000v40H0z M0 50 h400000 v40H0z m0 194h400000v40H0z`
	leftlinesegmentPath   = `M40 281 V428 H0 V94 H40 V241 H400000 v40z M40 281 V428 H0 V94 H40 V241 H400000 v40z`
	rightlinesegmentPath  = `M399960 241 V94 h40 V428 h-40 V281 H0 v-40z M399960 241 V94 h40 V428 h-40 V281 H0 v-40z`
	leftgroupPath         = `M400000 80 H435C64 80 168.3 229.4 21 260c-5.9 1.2-18 0-18 0-2 0-3-1-3-3v-38C76 61 257 0 435 0h399565z`
	leftgroupunderPath    = `M400000 262 H435C64 262 168.3 112.6 21 82c-5.9-1.2-18 0-18 0-2 0-3 1-3 3v38c76 158 257 219 435 219h399565z`
	rightgroupPath        = `M0 80h399565c371 0 266.7 149.4 414 180 5.9 1.2 18 0 18 0 2 0 3-1 3-3v-38c-76-158-257-219-435-219H0z`
	rightgroupunderPath   = `M0 262h399565c371 0 266.7-149.4 414-180 5.9-1.2 18 0 18 0 2 0 3 1 3 3v38c-76 158-257 219-435 219H0z`
	leftbracketoverPath   = `M0 440 h120 V150 H399995 v-120 H0z`
	rightbracketoverPath  = `M399995 440 h-120 V150 H0 v-120 H399995z`
	leftbracketunderPath  = `M0 0 h120 V290 H399995 v120 H0z`
	rightbracketunderPath = `M399995 0 h-120 V290 H0 v120 H400000z`
	baraboveleftarrowPath = `M400000 620h-399890l3-3c68.7-52.7 113.7-120 135-202 c4-14.7 6-23 6-25c0-7.3-7-11-21-11c-8 0-13.2 0.8-15.5 2.5c-2.3 1.7-4.2 5.8-5.5 12.5c-1.3 4.7-2.7 10.3-4 17c-12 48.7-34.8 92-68.5 130 s-74.2 66.3-121.5 85c-10 4-16 7.7-18 11c0 8.7 6 14.3 18 17c47.3 18.7 87.8 47 121.5 85s56.5 81.3 68.5 130c0.7 2 1.3 5 2 9s1.2 6.7 1.5 8c0.3 1.3 1 3.3 2 6 s2.2 4.5 3.5 5.5c1.3 1 3.3 1.8 6 2.5s6 1 10 1c14 0 21-3.7 21-11 c0-2-2-10.3-6-25c-20-79.3-65-146.7-135-202l-3-3h399890z M100 620v40h399900v-40z M0 241v40h399900v-40zM0 241v40h399900v-40z`
	rightarrowabovebarPath = `M0 241v40h399891c-47.3 35.3-84 78-110 128-16.7 32-27.7 63.7-33 95 0 1.3-.2 2.7-.5 4-.3 1.3-.5 2.3-.5 3 0 7.3 6.7 11 20 11 8 0 13.2-.8 15.5-2.5 2.3-1.7 4.2-5.5 5.5-11.5 2-13.3 5.7-27 11-41 14.7-44.7 39-84.5 73-119.5s73.7-60.2 119-75.5c6-2 9-5.7 9-11s-3-9-9-11c-45.3-15.3-85-40.5-119-75.5s-58.3-74.8-73-119.5c-4.7-14-8.3-27.3-11-40-1.3-6.7-3.2-10.8-5.5-12.5-2.3-1.7-7.5-2.5-15.5-2.5-14 0-21 3.7-21 11 0 2 2 10.3 6 25 20.7 83.3 67 151.7 139 205zm96 379h399894v40H0zm0 0h399904v40H0z`
	baraboveshortleftharpoonPath  = `M507,435c-4,4,-6.3,8.7,-7,14c0,5.3,0.7,9,2,11c1.3,2,5.3,5.3,12,10c90.7,54,156,130,196,228c3.3,10.7,6.3,16.3,9,17c2,0.7,5,1,9,1c0,0,5,0,5,0c10.7,0,16.7,-2,18,-6c2,-2.7,1,-9.7,-3,-21c-32,-87.3,-82.7,-157.7,-152,-211c0,0,-3,-3,-3,-3l399351,0l0,-40c-398570,0,-399437,0,-399437,0z M593 435 v40 H399500 v-40zM0 281 v-40 H399908 v40z M0 281 v-40 H399908 v40z`
	rightharpoonaboveshortbarPath = `M0,241 l0,40c399126,0,399993,0,399993,0c4.7,-4.7,7,-9.3,7,-14c0,-9.3,-3.7,-15.3,-11,-18c-92.7,-56.7,-159,-133.7,-199,-231c-3.3,-9.3,-6,-14.7,-8,-16c-2,-1.3,-7,-2,-15,-2c-10.7,0,-16.7,2,-18,6c-2,2.7,-1,9.7,3,21c15.3,42,36.7,81.8,64,119.5c27.3,37.7,58,69.2,92,94.5zM0 241 v40 H399908 v-40z M0 475 v-40 H399500 v40z M0 475 v-40 H399500 v40z`
	shortbaraboveleftharpoonPath  = `M7,435c-4,4,-6.3,8.7,-7,14c0,5.3,0.7,9,2,11c1.3,2,5.3,5.3,12,10c90.7,54,156,130,196,228c3.3,10.7,6.3,16.3,9,17c2,0.7,5,1,9,1c0,0,5,0,5,0c10.7,0,16.7,-2,18,-6c2,-2.7,1,-9.7,-3,-21c-32,-87.3,-82.7,-157.7,-152,-211c0,0,-3,-3,-3,-3l399907,0l0,-40c-399126,0,-399993,0,-399993,0zM93 435 v40 H400000 v-40z M500 241 v40 H400000 v-40z M500 241 v40 H400000 v-40z`
	shortrightharpoonabovebarPath = `M53,241l0,40c398570,0,399437,0,399437,0c4.7,-4.7,7,-9.3,7,-14c0,-9.3,-3.7,-15.3,-11,-18c-92.7,-56.7,-159,-133.7,-199,-231c-3.3,-9.3,-6,-14.7,-8,-16c-2,-1.3,-7,-2,-15,-2c-10.7,0,-16.7,2,-18,6c-2,2.7,-1,9.7,3,21c15.3,42,36.7,81.8,64,119.5c27.3,37.7,58,69.2,92,94.5zM500 241 v40 H399408 v-40z M500 435 v40 H400000 v-40z`
)

var widehatPaths = [4]string{
	`M529 0h5l519 115c5 1 9 5 9 10 0 1-1 2-1 3l-4 22c-1 5-5 9-11 9h-2L532 67 19 159h-2c-5 0-9-4-11-9l-5-22c-1-6 2-12 8-13z`,
	`M1181 0h2l1171 176c6 0 10 5 10 11l-2 23c-1 6-5 10-11 10h-1L1182 67 15 220h-1c-6 0-10-4-11-10l-2-23c-1-6 4-11 10-11z`,
	`M1181 0h2l1171 236c6 0 10 5 10 11l-2 23c-1 6-5 10-11 10h-1L1182 67 15 280h-1c-6 0-10-4-11-10l-2-23c-1-6 4-11 10-11z`,
	`M1181 0h2l1171 296c6 0 10 5 10 11l-2 23c-1 6-5 10-11 10h-1L1182 67 15 340h-1c-6 0-10-4-11-10l-2-23c-1-6 4-11 10-11z`,
}

var widecheckPaths = [4]string{
	`M529,159h5l519,-115c5,-1,9,-5,9,-10c0,-1,-1,-2,-1,-3l-4,-22c-1,-5,-5,-9,-11,-9h-2l-512,92l-513,-92h-2c-5,0,-9,4,-11,9l-5,22c-1,6,2,12,8,13z`,
	`M1181,220h2l1171,-176c6,0,10,-5,10,-11l-2,-23c-1,-6,-5,-10,-11,-10h-1l-1168,153l-1167,-153h-1c-6,0,-10,4,-11,10l-2,23c-1,6,4,11,10,11z`,
	`M1181,280h2l1171,-236c6,0,10,-5,10,-11l-2,-23c-1,-6,-5,-10,-11,-10h-1l-1168,213l-1167,-213h-1c-6,0,-10,4,-11,10l-2,23c-1,6,4,11,10,11z`,
	`M1181,340h2l1171,-296c6,0,10,-5,10,-11l-2,-23c-1,-6,-5,-10,-11,-10h-1l-1168,273l-1167,-273h-1c-6,0,-10,4,-11,10l-2,23c-1,6,4,11,10,11z`,
}

var tildePaths = [4]string{
	`M200 55.538c-77 0-168 73.953-177 73.953-3 0-7-2.175-9-5.437L2 97c-1-2-2-4-2-6 0-4 2-7 5-9l20-12C116 12 171 0 207 0c86 0 114 68 191 68 78 0 168-68 177-68 4 0 7 2 9 5l12 19c1 2.175 2 4.35 2 6.525 0 4.35-2 7.613-5 9.788l-19 13.05c-92 63.077-116.937 75.308-183 76.128 -68.267.847-113-73.952-191-73.952z`,
	`M344 55.266c-142 0-300.638 81.316-311.5 86.418 -8.01 3.762-22.5 10.91-23.5 5.562L1 120c-1-2-1-3-1-4 0-5 3-9 8-10l18.4-9C160.9 31.9 283 0 358 0c148 0 188 122 331 122s314-97 326-97c4 0 8 2 10 7l7 21.114 c1 2.14 1 3.21 1 4.28 0 5.347-3 9.626-7 10.696l-22.3 12.622C852.6 158.372 751  181.476 676 181.476c-149 0-189-126.21-332-126.21z`,
	`M786 59C457 59 32 175.242 13 175.242c-6 0-10-3.457-11-10.37L.15 138c-1-7 3-12 10-13l19.2-6.4C378.4 40.7 634.3 0 804.3 0c337 0 411.8 157 746.8 157 328 0 754-112 773-112 5 0 10 3 11 9l1 14.075c1 8.066-.697 16.595-6.697 17.492l-21.052 7.31c-367.9 98.146-609.15 122.696-778.15 122.696 -338 0-409-156.573-744-156.573z`,
	`M786 58C457 58 32 177.487 13 177.487c-6 0-10-3.345-11-10.035L.15 143c-1-7 3-12 10-13l22-6.7C381.2 35 637.15 0 807.15 0c337 0 409 177 744 177 328 0 754-127 773-127 5 0 10 3 11 9l1 14.794c1 7.805-3 13.38-9 14.495l-20.7 5.574c-366.85 99.79-607.3 139.372-776.3 139.372-338 0-409 -175.236-744-175.236z`,
}

func pathForName(name string) (string, bool) {
	switch name {
	case "rightarrow":
		return rightarrowPath, true
	case "leftarrow":
		return leftarrowPath, true
	case "doubleleftarrow":
		return doubleleftarrowPath, true
	case "doublerightarrow":
		return doublerightarrowPath, true
	case "leftharpoon":
		return leftharpoonPath, true
	case "leftharpoonplus":
		return leftharpoonplusPath, true
	case "leftharpoondown":
		return leftharpoondownPath, true
	case "leftharpoondownplus":
		return leftharpoondownplusPath, true
	case "rightharpoon":
		return rightharpoonPath, true
	case "rightharpoonplus":
		return rightharpoonplusPath, true
	case "rightharpoondown":
		return rightharpoondownPath, true
	case "rightharpoondownplus":
		return rightharpoondownplusPath, true
	case "baraboveshortleftharpoon":
		return baraboveshortleftharpoonPath, true
	case "rightharpoonaboveshortbar":
		return rightharpoonaboveshortbarPath, true
	case "shortbaraboveleftharpoon":
		return shortbaraboveleftharpoonPath, true
	case "shortrightharpoonabovebar":
		return shortrightharpoonabovebarPath, true
	case "lefthook":
		return lefthookPath, true
	case "righthook":
		return righthookPath, true
	case "leftbrace":
		return leftbracePath, true
	case "midbrace":
		return midbracePath, true
	case "rightbrace":
		return rightbracePath, true
	case "leftbraceunder":
		return leftbraceunderPath, true
	case "midbraceunder":
		return midbraceunderPath, true
	case "rightbraceunder":
		return rightbraceunderPath, true
	case "leftToFrom":
		return leftToFromPath, true
	case "rightToFrom":
		return rightToFromPath, true
	case "baraboveleftarrow":
		return baraboveleftarrowPath, true
	case "rightarrowabovebar":
		return rightarrowabovebarPath, true
	case "longequal":
		return longequalPath, true
	case "leftlinesegment":
		return leftlinesegmentPath, true
	case "rightlinesegment":
		return rightlinesegmentPath, true
	case "leftmapsto":
		return leftmapstoPath, true
	case "twoheadleftarrow":
		return twoheadleftarrowPath, true
	case "twoheadrightarrow":
		return twoheadrightarrowPath, true
	case "leftgroup":
		return leftgroupPath, true
	case "leftgroupunder":
		return leftgroupunderPath, true
	case "rightgroup":
		return rightgroupPath, true
	case "rightgroupunder":
		return rightgroupunderPath, true
	case "leftbracketover":
		return leftbracketoverPath, true
	case "rightbracketover":
		return rightbracketoverPath, true
	case "leftbracketunder":
		return leftbracketunderPath, true
	case "rightbracketunder":
		return rightbracketunderPath, true
	}
	return "", false
}

// =============================================================================
// katexImagesData lookup
// =============================================================================

type katexImageData struct {
	paths    []string
	minWidth float64
	vbHeight float64
	align    string
}

func katexImageInfo(label string) (katexImageData, bool) {
	key := strings.TrimPrefix(label, `\`)
	switch key {
	case "overrightarrow":
		return katexImageData{paths: []string{"rightarrow"}, minWidth: 0.888, vbHeight: 522, align: "xMaxYMin"}, true
	case "overleftarrow":
		return katexImageData{paths: []string{"leftarrow"}, minWidth: 0.888, vbHeight: 522, align: "xMinYMin"}, true
	case "underrightarrow":
		return katexImageData{paths: []string{"rightarrow"}, minWidth: 0.888, vbHeight: 522, align: "xMaxYMin"}, true
	case "underleftarrow":
		return katexImageData{paths: []string{"leftarrow"}, minWidth: 0.888, vbHeight: 522, align: "xMinYMin"}, true
	case "xrightarrow":
		return katexImageData{paths: []string{"rightarrow"}, minWidth: 1.469, vbHeight: 522, align: "xMaxYMin"}, true
	case "cdrightarrow":
		return katexImageData{paths: []string{"rightarrow"}, minWidth: 3.0, vbHeight: 522, align: "xMaxYMin"}, true
	case "cdleftarrow":
		return katexImageData{paths: []string{"leftarrow"}, minWidth: 3.0, vbHeight: 522, align: "xMinYMin"}, true
	case "xleftarrow":
		return katexImageData{paths: []string{"leftarrow"}, minWidth: 1.469, vbHeight: 522, align: "xMinYMin"}, true
	case "Overrightarrow":
		return katexImageData{paths: []string{"doublerightarrow"}, minWidth: 0.888, vbHeight: 560, align: "xMaxYMin"}, true
	case "xRightarrow":
		return katexImageData{paths: []string{"doublerightarrow"}, minWidth: 1.526, vbHeight: 560, align: "xMaxYMin"}, true
	case "xLeftarrow":
		return katexImageData{paths: []string{"doubleleftarrow"}, minWidth: 1.526, vbHeight: 560, align: "xMinYMin"}, true
	case "overleftharpoon", "xleftharpoonup":
		return katexImageData{paths: []string{"leftharpoon"}, minWidth: 0.888, vbHeight: 522, align: "xMinYMin"}, true
	case "xleftharpoondown":
		return katexImageData{paths: []string{"leftharpoondown"}, minWidth: 0.888, vbHeight: 522, align: "xMinYMin"}, true
	case "overrightharpoon", "xrightharpoonup":
		return katexImageData{paths: []string{"rightharpoon"}, minWidth: 0.888, vbHeight: 522, align: "xMaxYMin"}, true
	case "xrightharpoondown":
		return katexImageData{paths: []string{"rightharpoondown"}, minWidth: 0.888, vbHeight: 522, align: "xMaxYMin"}, true
	case "xlongequal":
		return katexImageData{paths: []string{"longequal"}, minWidth: 0.888, vbHeight: 334, align: "xMinYMin"}, true
	case "cdlongequal":
		return katexImageData{paths: []string{"longequal"}, minWidth: 3.0, vbHeight: 334, align: "xMinYMin"}, true
	case "xtwoheadleftarrow":
		return katexImageData{paths: []string{"twoheadleftarrow"}, minWidth: 0.888, vbHeight: 334, align: "xMinYMin"}, true
	case "xtwoheadrightarrow":
		return katexImageData{paths: []string{"twoheadrightarrow"}, minWidth: 0.888, vbHeight: 334, align: "xMaxYMin"}, true
	case "overleftrightarrow":
		return katexImageData{paths: []string{"leftarrow", "rightarrow"}, minWidth: 0.888, vbHeight: 522}, true
	case "underleftrightarrow":
		return katexImageData{paths: []string{"leftarrow", "rightarrow"}, minWidth: 0.888, vbHeight: 522}, true
	case "xleftrightarrow":
		return katexImageData{paths: []string{"leftarrow", "rightarrow"}, minWidth: 1.75, vbHeight: 522}, true
	case "xLeftrightarrow":
		return katexImageData{paths: []string{"doubleleftarrow", "doublerightarrow"}, minWidth: 1.75, vbHeight: 560}, true
	case "xrightleftharpoons":
		return katexImageData{paths: []string{"leftharpoondownplus", "rightharpoonplus"}, minWidth: 1.75, vbHeight: 716}, true
	case "xleftrightharpoons":
		return katexImageData{paths: []string{"leftharpoonplus", "rightharpoondownplus"}, minWidth: 1.75, vbHeight: 716}, true
	case "xrightequilibrium":
		return katexImageData{paths: []string{"baraboveshortleftharpoon", "rightharpoonaboveshortbar"}, minWidth: 1.75, vbHeight: 716}, true
	case "xleftequilibrium":
		return katexImageData{paths: []string{"shortbaraboveleftharpoon", "shortrightharpoonabovebar"}, minWidth: 1.75, vbHeight: 716}, true
	case "xhookleftarrow":
		return katexImageData{paths: []string{"leftarrow", "righthook"}, minWidth: 1.08, vbHeight: 522}, true
	case "xhookrightarrow":
		return katexImageData{paths: []string{"lefthook", "rightarrow"}, minWidth: 1.08, vbHeight: 522}, true
	case "overlinesegment":
		return katexImageData{paths: []string{"leftlinesegment", "rightlinesegment"}, minWidth: 0.888, vbHeight: 522}, true
	case "underlinesegment":
		return katexImageData{paths: []string{"leftlinesegment", "rightlinesegment"}, minWidth: 0.888, vbHeight: 522}, true
	case "overgroup":
		return katexImageData{paths: []string{"leftgroup", "rightgroup"}, minWidth: 0.888, vbHeight: 342}, true
	case "undergroup":
		return katexImageData{paths: []string{"leftgroupunder", "rightgroupunder"}, minWidth: 0.888, vbHeight: 342}, true
	case "xmapsto":
		return katexImageData{paths: []string{"leftmapsto", "rightarrow"}, minWidth: 1.5, vbHeight: 522}, true
	case "xtofrom":
		return katexImageData{paths: []string{"leftToFrom", "rightToFrom"}, minWidth: 1.75, vbHeight: 528}, true
	case "xrightleftarrows":
		return katexImageData{paths: []string{"baraboveleftarrow", "rightarrowabovebar"}, minWidth: 1.75, vbHeight: 901}, true
	case "overbrace":
		return katexImageData{paths: []string{"leftbrace", "midbrace", "rightbrace"}, minWidth: 0.888, vbHeight: 548}, true
	case "underbrace":
		return katexImageData{paths: []string{"leftbraceunder", "midbraceunder", "rightbraceunder"}, minWidth: 0.888, vbHeight: 548}, true
	case "overbracket":
		return katexImageData{paths: []string{"leftbracketover", "rightbracketover"}, minWidth: 1.6, vbHeight: 440}, true
	case "underbracket":
		return katexImageData{paths: []string{"leftbracketunder", "rightbracketunder"}, minWidth: 1.6, vbHeight: 410}, true
	}
	return katexImageData{}, false
}

// katexStretchyMinWidthEm returns the minWidth in em for a stretchy-arrow label.
func katexStretchyMinWidthEm(label string) (float64, bool) {
	d, ok := katexImageInfo(label)
	if !ok {
		return 0, false
	}
	return d.minWidth, true
}

// katexStretchyPath builds the path commands for a stretchy element.
// Returns (commands, height_em, ok). The path is centered at y=0 (shaft
// at midline, extending ±height_em/2).
func katexStretchyPath(label string, widthEm float64) ([]path.Command, float64, bool) {
	data, ok := katexImageInfo(label)
	if !ok {
		return nil, 0, false
	}
	const s = 1.0 / 1000.0
	heightEm := data.vbHeight * s
	vbCy := data.vbHeight / 2
	yMin := -heightEm / 2
	yMax := heightEm / 2

	makeCmds := func(name string, xShift float64) ([]path.Command, bool) {
		svgStr, ok := pathForName(name)
		if !ok {
			return nil, false
		}
		raw := parseSVGPath(svgStr)
		out := make([]path.Command, len(raw))
		for i, c := range raw {
			out[i] = scaleCmdTwoheadUniform(c, s, vbCy, xShift)
		}
		return out, true
	}

	switch len(data.paths) {
	case 1:
		xShift := 0.0
		if data.align == "xMaxYMin" {
			xShift = widthEm - 400000*s
		}
		cmds, ok := makeCmds(data.paths[0], xShift)
		if !ok {
			return nil, 0, false
		}
		return clipPathToRect(cmds, 0, widthEm, yMin, yMax), heightEm, true
	case 2:
		xR := widthEm - 400000*s
		lc, ok1 := makeCmds(data.paths[0], 0)
		rc, ok2 := makeCmds(data.paths[1], xR)
		if !ok1 || !ok2 {
			return nil, 0, false
		}
		isXmapsto := data.paths[0] == "leftmapsto" && data.paths[1] == "rightarrow"
		var out []path.Command
		if isXmapsto {
			leftBarXMax := 40 * s
			shaftOverlap := 0.1
			shaftXStart := leftBarXMax - shaftOverlap
			if shaftXStart < 0 {
				shaftXStart = 0
			}
			shaftYTop := (241 - vbCy) * s
			shaftYBot := (281 - vbCy) * s
			shaftXEnd := xR
			if leftBarXMax > shaftXEnd {
				shaftXEnd = leftBarXMax
			}
			out = clipPathToRect(lc, 0, leftBarXMax, yMin, yMax)
			if shaftXEnd > shaftXStart {
				out = append(out, path.MoveTo(shaftXStart, shaftYTop))
				out = append(out, path.LineTo(shaftXEnd, shaftYTop))
				out = append(out, path.LineTo(shaftXEnd, shaftYBot))
				out = append(out, path.LineTo(shaftXStart, shaftYBot))
				out = append(out, path.Command{Kind: path.KindClose})
			}
			out = append(out, clipPathToRect(rc, 0, widthEm, yMin, yMax)...)
		} else {
			mid := widthEm / 2
			out = clipPathToRect(lc, 0, mid, yMin, yMax)
			out = append(out, clipPathToRect(rc, mid, widthEm, yMin, yMax)...)
		}
		return out, heightEm, true
	case 3:
		xM := widthEm/2 - 200000*s
		xR := widthEm - 400000*s
		lc, ok1 := makeCmds(data.paths[0], 0)
		mc, ok2 := makeCmds(data.paths[1], xM)
		rc, ok3 := makeCmds(data.paths[2], xR)
		if !ok1 || !ok2 || !ok3 {
			return nil, 0, false
		}
		key := strings.TrimPrefix(label, `\`)
		var out []path.Command
		if key == "overbrace" || key == "underbrace" {
			leftMax := widthEm * 0.251
			centerMin := widthEm * 0.25
			centerMax := widthEm * 0.75
			rightMin := widthEm * (1 - 0.251)
			out = clipPathToRect(lc, 0, leftMax, yMin, yMax)
			out = append(out, clipPathToRect(mc, centerMin, centerMax, yMin, yMax)...)
			out = append(out, clipPathToRect(rc, rightMin, widthEm, yMin, yMax)...)
		} else {
			out = clipPathToRect(lc, 0, widthEm, yMin, yMax)
			out = append(out, clipPathToRect(mc, 0, widthEm, yMin, yMax)...)
			out = append(out, clipPathToRect(rc, 0, widthEm, yMin, yMax)...)
		}
		return out, heightEm, true
	}
	return nil, 0, false
}

// =============================================================================
// Wide-accent paths (\widehat / \widetilde / \widecheck / \overgroup / \undergroup)
// =============================================================================

func wideAccentImgIndex(groupLen int) int {
	if groupLen > 5 {
		return 4
	}
	idxs := [6]int{1, 1, 2, 2, 3, 3}
	return idxs[groupLen]
}

func selectHatCheck(isHat bool, groupLen int) (svgPath string, vbW, vbH, hEm float64) {
	imgIndex := wideAccentImgIndex(groupLen)
	var prefix [4]string
	if isHat {
		prefix = widehatPaths
	} else {
		prefix = widecheckPaths
	}
	vbWArr := [5]float64{0, 1062, 2364, 2364, 2364}
	vbHArr := [5]float64{0, 239, 300, 360, 420}
	vbW = vbWArr[imgIndex]
	vbH = vbHArr[imgIndex]
	if groupLen > 5 {
		hEm = 0.42
	} else {
		hArr := [6]float64{0, 0.24, 0.3, 0.3, 0.36, 0.42}
		hEm = hArr[imgIndex]
	}
	svgPath = prefix[imgIndex-1]
	return
}

func selectTilde(groupLen int) (svgPath string, vbW, vbH, hEm float64) {
	imgIndex := wideAccentImgIndex(groupLen)
	vbWArr := [5]float64{0, 600, 1033, 2339, 2340}
	vbHArr := [5]float64{0, 260, 286, 306, 312}
	vbW = vbWArr[imgIndex]
	vbH = vbHArr[imgIndex]
	if groupLen > 5 {
		hEm = 0.34
	} else {
		hArr := [6]float64{0, 0.26, 0.286, 0.3, 0.306, 0.34}
		hEm = hArr[imgIndex]
	}
	svgPath = tildePaths[imgIndex-1]
	return
}

func parseAndFitNonuniform(svgPath string, vbW, vbH, targetW, targetH float64) []path.Command {
	raw := parseSVGPath(svgPath)
	sx := targetW / vbW
	sy := targetH / vbH
	out := make([]path.Command, len(raw))
	for i, c := range raw {
		out[i] = c
		switch c.Kind {
		case path.KindMoveTo, path.KindLineTo:
			out[i].X = c.X * sx
			out[i].Y = c.Y * sy
		case path.KindCubicTo:
			out[i].X1 = c.X1 * sx
			out[i].Y1 = c.Y1 * sy
			out[i].X2 = c.X2 * sx
			out[i].Y2 = c.Y2 * sy
			out[i].X = c.X * sx
			out[i].Y = c.Y * sy
		case path.KindQuadTo:
			out[i].X1 = c.X1 * sx
			out[i].Y1 = c.Y1 * sy
			out[i].X = c.X * sx
			out[i].Y = c.Y * sy
		}
	}
	return out
}

func buildOvergroup(widthEm, hEm float64, isOver bool) []path.Command {
	const vbH = 342.0
	sy := hEm / vbH
	naturalCorner := 0.435
	cornerEm := naturalCorner
	if widthEm/2 < cornerEm {
		cornerEm = widthEm / 2
	}
	cx := cornerEm / 435.0
	ly := func(y float64) float64 { return y * sy }
	if isOver {
		return []path.Command{
			path.MoveTo(cornerEm, ly(80)),
			path.CubicTo(64*cx, ly(80), 168.3*cx, ly(229.4), 21*cx, ly(260)),
			path.CubicTo(15.1*cx, ly(261.2), 3*cx, ly(260), 3*cx, ly(260)),
			path.CubicTo(1*cx, ly(260), 0, ly(259), 0, ly(257)),
			path.LineTo(0, ly(219)),
			path.CubicTo(76*cx, ly(61), 257*cx, ly(0), cornerEm, ly(0)),
			path.LineTo(widthEm-cornerEm, ly(0)),
			path.CubicTo(widthEm-257*cx, ly(0), widthEm-76*cx, ly(61), widthEm, ly(219)),
			path.LineTo(widthEm, ly(257)),
			path.CubicTo(widthEm, ly(259), widthEm-1*cx, ly(260), widthEm-3*cx, ly(260)),
			path.CubicTo(widthEm-3*cx, ly(260), widthEm-15.1*cx, ly(261.2), widthEm-21*cx, ly(260)),
			path.CubicTo(widthEm-168.3*cx, ly(229.4), widthEm-64*cx, ly(80), widthEm-cornerEm, ly(80)),
			path.Command{Kind: path.KindClose},
		}
	}
	return []path.Command{
		path.MoveTo(cornerEm, ly(262)),
		path.CubicTo(64*cx, ly(262), 168.3*cx, ly(112.6), 21*cx, ly(82)),
		path.CubicTo(15.1*cx, ly(80.8), 3*cx, ly(82), 3*cx, ly(82)),
		path.CubicTo(1*cx, ly(82), 0, ly(83), 0, ly(85)),
		path.LineTo(0, ly(123)),
		path.CubicTo(76*cx, ly(281), 257*cx, ly(342), cornerEm, ly(342)),
		path.LineTo(widthEm-cornerEm, ly(342)),
		path.CubicTo(widthEm-257*cx, ly(342), widthEm-76*cx, ly(281), widthEm, ly(123)),
		path.LineTo(widthEm, ly(85)),
		path.CubicTo(widthEm, ly(83), widthEm-1*cx, ly(82), widthEm-3*cx, ly(82)),
		path.CubicTo(widthEm-3*cx, ly(82), widthEm-15.1*cx, ly(80.8), widthEm-21*cx, ly(82)),
		path.CubicTo(widthEm-168.3*cx, ly(112.6), widthEm-64*cx, ly(262), widthEm-cornerEm, ly(262)),
		path.Command{Kind: path.KindClose},
	}
}

// katexAccentPath returns (commands, width_em, height_em, fill_ok) for a
// KaTeX-style wide accent. Returns ok=false if label isn't a wide
// accent.
func katexAccentPath(label string, baseWidthEm float64, groupLen int) (cmds []path.Command, wEm, hEm float64, fill, ok bool) {
	switch label {
	case `\vec`:
		const vecW = 0.471
		const vecH = 0.714
		raw := parseSVGPath(katexVecPath)
		return scaleSVGPathThousandths(raw), vecW, vecH, true, true
	case `\widehat`, `\widecheck`:
		isHat := label == `\widehat`
		svgPath, vbW, vbH, h := selectHatCheck(isHat, groupLen)
		c := parseAndFitNonuniform(svgPath, vbW, vbH, baseWidthEm, h)
		return c, baseWidthEm, h, true, true
	case `\widetilde`, `\utilde`:
		svgPath, vbW, vbH, h := selectTilde(groupLen)
		c := parseAndFitNonuniform(svgPath, vbW, vbH, baseWidthEm, h)
		return c, baseWidthEm, h, true, true
	case `\overgroup`:
		const h = 0.342
		c := buildOvergroup(baseWidthEm, h, true)
		return c, baseWidthEm, h, true, true
	case `\undergroup`:
		const h = 0.342
		c := buildOvergroup(baseWidthEm, h, false)
		return c, baseWidthEm, h, true, true
	}
	return nil, 0, 0, false, false
}
