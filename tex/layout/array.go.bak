package layout

import (
	"github.com/luo-studio/go-tex/tex/parser"
)

// layoutArray is a starting port of upstream's parse_array layout. It
// produces an Array Box with row/col dimensions; the displaylist emit
// then walks the cells in grid order.
func layoutArray(a *parser.Array, opts Options) *Box {
	if len(a.Body) == 0 {
		return NewEmpty()
	}
	numCols := 0
	for _, row := range a.Body {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	cells := make([][]*Box, len(a.Body))
	colWidths := make([]float64, numCols)
	rowHeights := make([]float64, len(a.Body))
	rowDepths := make([]float64, len(a.Body))
	for ri, row := range a.Body {
		cells[ri] = make([]*Box, numCols)
		var h, d float64
		for ci := 0; ci < numCols; ci++ {
			if ci < len(row) {
				cb := layoutNode(row[ci], opts)
				cells[ri][ci] = cb
				if cb.Width > colWidths[ci] {
					colWidths[ci] = cb.Width
				}
				if cb.Height > h {
					h = cb.Height
				}
				if cb.Depth > d {
					d = cb.Depth
				}
			} else {
				cells[ri][ci] = NewEmpty()
			}
		}
		rowHeights[ri] = h
		rowDepths[ri] = d
	}

	// Column gap (arraycolsep). Upstream uses 5pt = 0.5em for non-align.
	colGap := 0.5
	if a.ColSeparationType != nil {
		switch *a.ColSeparationType {
		case "small":
			colGap = 0.16667
		case "align":
			colGap = 0
		case "alignat":
			colGap = 0.5
		case "gather":
			colGap = 0
		case "CD":
			colGap = 0.5
		}
	}
	hskipBefore := false
	if a.HskipBeforeAndAft != nil {
		hskipBefore = *a.HskipBeforeAndAft
	}

	totalW := 0.0
	for ci, w := range colWidths {
		if ci > 0 {
			totalW += 2 * colGap
		}
		totalW += w
	}
	if hskipBefore {
		totalW += 2 * colGap // padding on outer edges
	}

	// Row stacking: arraystretch * 12pt baseline-skip equivalent.
	stretch := a.ArrayStretch
	if stretch == 0 {
		stretch = 1.0
	}
	baselineSkip := 1.2 * stretch // 12pt line height

	totalH := 0.0
	totalD := 0.0
	for ri := range a.Body {
		totalH += rowHeights[ri]
		totalD += rowDepths[ri]
		if ri > 0 {
			gap := baselineSkip - rowHeights[ri-1] - rowDepths[ri-1]
			if gap < 0 {
				gap = 0
			}
			totalD += gap
		}
	}

	xOffset := 0.0
	if hskipBefore {
		xOffset = colGap
	}

	colAligns := make([]byte, numCols)
	for i := range colAligns {
		colAligns[i] = 'c'
		if a.Cols != nil && i < len(a.Cols) && a.Cols[i].Align != nil {
			colAligns[i] = (*a.Cols[i].Align)[0]
		}
	}

	return &Box{
		Width:  totalW,
		Height: totalH/2 + 0.25, // approximate axis-centred placement
		Depth:  totalD/2 + 0.25,
		Color:  opts.Color,
		Content: Array{
			Cells:           cells,
			ColWidths:       colWidths,
			ColAligns:       colAligns,
			RowHeights:      rowHeights,
			RowDepths:       rowDepths,
			ColGap:          colGap,
			ContentXOffset:  xOffset,
			ArrayInnerWidth: totalW,
		},
	}
}
