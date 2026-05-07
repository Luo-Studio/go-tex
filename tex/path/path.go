// Package path defines the SVG-style path commands used by the renderer to
// draw glyph outlines, radical signs, and large delimiters.
package path

// Kind is the discriminator for a Command.
type Kind uint8

const (
	KindMoveTo Kind = iota
	KindLineTo
	KindCubicTo
	KindQuadTo
	KindClose
)

// Command is a single drawing instruction in a path. The fields used depend on
// Kind:
//
//	MoveTo, LineTo:  X, Y
//	CubicTo:         X1, Y1, X2, Y2, X, Y
//	QuadTo:          X1, Y1, X, Y
//	Close:           (no fields)
//
// Using a single struct (rather than a sealed interface) keeps paths
// allocation-free for the renderer's hot loop.
type Command struct {
	Kind           Kind
	X, Y           float64
	X1, Y1, X2, Y2 float64
}

// MoveTo returns a "move pen to (x, y) without drawing" command.
func MoveTo(x, y float64) Command {
	return Command{Kind: KindMoveTo, X: x, Y: y}
}

// LineTo returns a "draw a line to (x, y)" command.
func LineTo(x, y float64) Command {
	return Command{Kind: KindLineTo, X: x, Y: y}
}

// CubicTo returns a cubic Bezier with control points (x1, y1), (x2, y2) and
// endpoint (x, y).
func CubicTo(x1, y1, x2, y2, x, y float64) Command {
	return Command{Kind: KindCubicTo, X1: x1, Y1: y1, X2: x2, Y2: y2, X: x, Y: y}
}

// QuadTo returns a quadratic Bezier with control point (x1, y1) and endpoint
// (x, y).
func QuadTo(x1, y1, x, y float64) Command {
	return Command{Kind: KindQuadTo, X1: x1, Y1: y1, X: x, Y: y}
}

// Close returns a "close current subpath" command.
func Close() Command {
	return Command{Kind: KindClose}
}
