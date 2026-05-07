package layout

import (
	"github.com/luo-studio/go-tex/tex/parser"
)

// layoutArray ports upstream parse_array layout: arstrut-based row
// metrics, axis-centred offset, col-sep-type-driven column gap.
func layoutArray(a *parser.Array, opts Options) *Box {
	if len(a.Body) == 0 {
		return NewEmpty()
	}
	if a.IsCD != nil && *a.IsCD {
		return layoutCD(a.Body, opts)
	}
	metrics := opts.Metrics()
	pt := 1.0 / metrics.PtPerEm
	baselineSkip := 12.0 * pt
	jot := 3.0 * pt
	arraystretch := a.ArrayStretch
	if arraystretch == 0 {
		arraystretch = 1.0
	}
	arrayskip := arraystretch * baselineSkip
	arstrutH := 0.7 * arrayskip
	arstrutD := 0.3 * arrayskip

	colGap := 2 * 5 * pt
	if a.ColSeparationType != nil {
		switch *a.ColSeparationType {
		case "align":
			colGap = muToEm(3.0, metrics.Quad)
		case "alignat":
			colGap = 0
		case "small":
			colGap = 2.0 * muToEm(5.0, metrics.Quad) * 0.7 / opts.SizeMultiplier()
		}
	}

	cellOpts := opts
	if a.ColSeparationType != nil {
		switch *a.ColSeparationType {
		case "align", "alignat":
			cap := 3.0
			cellOpts.AlignRelationSpacing = &cap
		}
	}

	numRows := len(a.Body)
	numCols := 0
	for _, row := range a.Body {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	cells := make([][]*Box, numRows)
	colWidths := make([]float64, numCols)
	rowHeights := make([]float64, numRows)
	rowDepths := make([]float64, numRows)

	addJot := false
	if a.AddJot != nil {
		addJot = *a.AddJot
	}

	for ri, row := range a.Body {
		cells[ri] = make([]*Box, numCols)
		rh := arstrutH
		rd := arstrutD
		for ci := 0; ci < numCols; ci++ {
			if ci < len(row) {
				cb := layoutNode(row[ci], cellOpts)
				cells[ri][ci] = cb
				if cb.Width > colWidths[ci] {
					colWidths[ci] = cb.Width
				}
				if cb.Height > rh {
					rh = cb.Height
				}
				if cb.Depth > rd {
					rd = cb.Depth
				}
			} else {
				cells[ri][ci] = NewEmpty()
			}
		}
		if addJot {
			rd += jot
		}
		rowHeights[ri] = rh
		rowDepths[ri] = rd
	}

	hlines := a.HLinesBeforeRow
	for len(hlines) < numRows+1 {
		hlines = append(hlines, []bool{})
	}

	totalH := 0.0
	for r := 0; r < numRows; r++ {
		totalH += rowHeights[r] + rowDepths[r]
	}
	offset := totalH/2 + metrics.AxisHeight

	hskip := false
	if a.HskipBeforeAndAft != nil {
		hskip = *a.HskipBeforeAndAft
	}
	contentXOffset := 0.0
	if hskip {
		contentXOffset = colGap / 2
	}

	var arrayInnerW float64
	for _, w := range colWidths {
		arrayInnerW += w
	}
	if numCols > 1 {
		arrayInnerW += colGap * float64(numCols-1)
	}
	arrayInnerW += 2 * contentXOffset

	colAligns := make([]byte, numCols)
	for i := range colAligns {
		colAligns[i] = 'c'
	}
	// col_separators[i] for i in [0, numCols]: nil = no rule, false = '|',
	// true = ':' (dashed). Mirrors upstream.
	colSeparators := make([]*bool, numCols+1)
	if a.Cols != nil {
		alignCount := 0
		for _, sp := range a.Cols {
			switch sp.Type {
			case parser.AlignTypeAlign:
				if alignCount < numCols && sp.Align != nil && len(*sp.Align) > 0 {
					colAligns[alignCount] = (*sp.Align)[0]
				}
				alignCount++
			case parser.AlignTypeSeparator:
				if sp.Align != nil && alignCount <= numCols {
					switch *sp.Align {
					case "|":
						v := false
						colSeparators[alignCount] = &v
					case ":":
						v := true
						colSeparators[alignCount] = &v
					}
				}
			}
		}
	}

	// Equation tags (e.g. (1) on the right of \begin{equation}/\begin{align}).
	rowTagBoxes := make([]*Box, numRows)
	tagColWidth := 0.0
	textOpts := opts.WithStyle(opts.Style.Text())
	if len(a.Tags) == numRows {
		for ri, t := range a.Tags {
			if t.Explicit != nil && len(t.Explicit) > 0 {
				tb := layoutExpression(t.Explicit, textOpts, true)
				if tb.Width > tagColWidth {
					tagColWidth = tb.Width
				}
				rowTagBoxes[ri] = tb
			}
		}
	}
	tagGapEm := 0.0
	if tagColWidth > 0 {
		tagGapEm = textOpts.Metrics().Quad
	}
	totalWidth := arrayInnerW + tagGapEm + tagColWidth
	tagsLeft := false
	if a.Leqno != nil {
		tagsLeft = *a.Leqno
	}

	return &Box{
		Width:  totalWidth,
		Height: offset,
		Depth:  totalH - offset,
		Color:  opts.Color,
		Content: Array{
			Cells:           cells,
			ColWidths:       colWidths,
			ColAligns:       colAligns,
			RowHeights:      rowHeights,
			RowDepths:       rowDepths,
			ColGap:          colGap,
			Offset:          offset,
			ContentXOffset:  contentXOffset,
			HLinesBeforeRow: hlines,
			ArrayInnerWidth: arrayInnerW,
			ColSeparators:   colSeparators,
			RuleThickness:   0.4 / metrics.PtPerEm,
			DoubleRuleSep:   metrics.DoubleRuleSep,
			TagGapEm:        tagGapEm,
			TagColWidth:     tagColWidth,
			RowTags:         rowTagBoxes,
			TagsLeft:        tagsLeft,
		},
	}
}
