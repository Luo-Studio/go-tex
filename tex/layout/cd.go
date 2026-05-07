package layout

import (
	"github.com/luo-studio/go-tex/tex/parser"
)

// cd.go: Layout for \begin{CD}...\end{CD} commutative diagrams.
// Mirrors upstream layout_cd.

const cdVertSideKernEm = 0.11

// layoutCD does a two-pass layout for a CD diagram. Mirrors upstream layout_cd.
func layoutCD(body [][]parser.Node, opts Options) *Box {
	m := opts.Metrics()
	pt := 1.0 / m.PtPerEm
	baselineSkip := 3.0 * m.XHeight
	arstrutH := 0.7 * baselineSkip
	arstrutD := 0.3 * baselineSkip

	numRows := len(body)
	if numRows == 0 {
		return NewEmpty()
	}
	numCols := 0
	for _, row := range body {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	if numCols == 0 {
		return NewEmpty()
	}
	jot := 3.0 * pt

	// Pass 1: layout cells at natural size.
	cellBoxes := make([][]*Box, numRows)
	colWidths := make([]float64, numCols)
	rowHeights := make([]float64, numRows)
	rowDepths := make([]float64, numRows)
	for r := range rowHeights {
		rowHeights[r] = arstrutH
		rowDepths[r] = arstrutD
	}
	for r, row := range body {
		rowBoxes := make([]*Box, 0, numCols)
		for c, cell := range row {
			var cb *Box
			switch v := cell.(type) {
			case *parser.CdArrow:
				cb = layoutCDArrow(v.Direction, v.LabelAbove, v.LabelBelow, 0, 0, opts)
			case *parser.OrdGroup:
				cb = layoutExpression(v.Body, opts, false)
			default:
				cb = layoutNode(cell, opts)
			}
			if cb.Height > rowHeights[r] {
				rowHeights[r] = cb.Height
			}
			if cb.Depth > rowDepths[r] {
				rowDepths[r] = cb.Depth
			}
			if cb.Width > colWidths[c] {
				colWidths[c] = cb.Width
			}
			rowBoxes = append(rowBoxes, cb)
		}
		for len(rowBoxes) < numCols {
			rowBoxes = append(rowBoxes, NewEmpty())
		}
		cellBoxes[r] = rowBoxes
	}

	colTargetW := make([]float64, numCols)
	copy(colTargetW, colWidths)

	// Pass 2: stretch arrow cells to column targets / row heights.
	for r, row := range body {
		isArrowRow := r%2 == 1
		for c, cell := range row {
			ca, ok := cell.(*parser.CdArrow)
			if !ok {
				continue
			}
			isHoriz := ca.Direction == "right" || ca.Direction == "left" || ca.Direction == "horiz_eq"
			if !isArrowRow && c%2 == 1 && isHoriz {
				cb := layoutCDArrow(ca.Direction, ca.LabelAbove, ca.LabelBelow, cellBoxes[r][c].Width, colTargetW[c], opts)
				if cb.Width > colWidths[c] {
					colWidths[c] = cb.Width
				}
				cellBoxes[r][c] = cb
			} else if isArrowRow && c%2 == 0 {
				vSpan := rowHeights[r] + rowDepths[r]
				cb := layoutCDArrow(ca.Direction, ca.LabelAbove, ca.LabelBelow, vSpan, colWidths[c], opts)
				if cb.Width > colWidths[c] {
					colWidths[c] = cb.Width
				}
				cellBoxes[r][c] = cb
			}
		}
	}

	// addJot: 3pt added to every row depth.
	for i := range rowDepths {
		rowDepths[i] += jot
	}

	colGap := 0.5
	colAligns := make([]byte, numCols)
	for i := range colAligns {
		colAligns[i] = 'c'
	}
	colSeparators := make([]*bool, numCols+1)

	totalH := 0.0
	for r := 0; r < numRows; r++ {
		totalH += rowHeights[r] + rowDepths[r]
	}
	offset := totalH/2 + m.AxisHeight
	totalW := 0.0
	for _, w := range colWidths {
		totalW += w
	}
	if numCols > 1 {
		totalW += colGap * float64(numCols-1)
	}

	hlines := make([][]bool, numRows+1)

	return &Box{
		Width:  totalW,
		Height: offset,
		Depth:  totalH - offset,
		Color:  opts.Color,
		Content: Array{
			Cells:           cellBoxes,
			ColWidths:       colWidths,
			ColAligns:       colAligns,
			RowHeights:      rowHeights,
			RowDepths:       rowDepths,
			ColGap:          colGap,
			Offset:          offset,
			ContentXOffset:  0,
			HLinesBeforeRow: hlines,
			ColSeparators:   colSeparators,
			RuleThickness:   0.04 * pt,
			DoubleRuleSep:   m.DoubleRuleSep,
			ArrayInnerWidth: totalW,
			RowTags:         make([]*Box, numRows),
		},
	}
}

// layoutCDArrow renders a single CdArrow cell. Mirrors upstream layout_cd_arrow.
func layoutCDArrow(direction string, labelAbove, labelBelow parser.Node, targetSize, targetColWidth float64, opts Options) *Box {
	switch direction {
	case "right", "left", "horiz_eq":
		return layoutCDHorizArrow(direction, labelAbove, labelBelow, targetSize, targetColWidth, opts)
	case "down", "up", "vert_eq":
		return layoutCDVertArrow(direction, labelAbove, labelBelow, targetSize, targetColWidth, opts)
	}
	return NewEmpty()
}

func layoutCDHorizArrow(direction string, labelAbove, labelBelow parser.Node, targetSize, targetColWidth float64, opts Options) *Box {
	axis := opts.Metrics().AxisHeight
	supStyle := opts.Style.Superscript()
	subStyle := opts.Style.Subscript()
	supRatio := supStyle.SizeMultiplier() / opts.Style.SizeMultiplier()
	subRatio := subStyle.SizeMultiplier() / opts.Style.SizeMultiplier()

	var aboveBox, belowBox *Box
	if labelAbove != nil {
		aboveBox = layoutNode(labelAbove, opts.WithStyle(supStyle))
	}
	if labelBelow != nil {
		belowBox = layoutNode(labelBelow, opts.WithStyle(subStyle))
	}
	aboveW := 0.0
	belowW := 0.0
	if aboveBox != nil {
		aboveW = aboveBox.Width * supRatio
	}
	if belowBox != nil {
		belowW = belowBox.Width * subRatio
	}

	pathLabel := `\cdrightarrow`
	if direction == "left" {
		pathLabel = `\cdleftarrow`
	} else if direction == "horiz_eq" {
		pathLabel = `\cdlongequal`
	}
	minShaftW, _ := katexStretchyMinWidthEm(pathLabel)
	if minShaftW == 0 {
		minShaftW = 1.0
	}
	const cdLabelPadL = 0.22
	const cdLabelPadR = 0.48
	cdPadSup := (cdLabelPadL + cdLabelPadR) * supRatio
	cdPadSub := (cdLabelPadL + cdLabelPadR) * subRatio
	upperNeed := 0.0
	if aboveBox != nil {
		upperNeed = aboveW + cdPadSup
	}
	lowerNeed := 0.0
	if belowBox != nil {
		lowerNeed = belowW + cdPadSub
	}
	naturalW := upperNeed
	if lowerNeed > naturalW {
		naturalW = lowerNeed
	}
	shaftW := targetSize
	if shaftW <= 0 {
		shaftW = naturalW
		if minShaftW > shaftW {
			shaftW = minShaftW
		}
	}
	cmds, actualH, ok := katexStretchyPath(pathLabel, shaftW)
	if !ok {
		return NewEmpty()
	}
	arrowHalf := actualH / 2
	arrowBox := &Box{
		Width: shaftW, Height: arrowHalf, Depth: arrowHalf, Color: opts.Color,
		Content: SvgPath{Commands: cmds, Fill: true},
	}
	gap := 0.111
	supH := 0.0
	supD := 0.0
	supDContrib := 0.0
	if aboveBox != nil {
		supH = aboveBox.Height * supRatio
		supD = aboveBox.Depth * supRatio
		if aboveBox.Depth > 0.25 {
			supDContrib = supD
		}
	}
	height := axis + arrowHalf + gap + supH + supDContrib
	subHRaw := 0.0
	subDRaw := 0.0
	if belowBox != nil {
		subHRaw = belowBox.Height * subRatio
		subDRaw = belowBox.Depth * subRatio
	}
	depth := arrowHalf - axis
	if depth < 0 {
		depth = 0
	}
	if belowBox != nil {
		depth += gap + subHRaw + subDRaw
	}
	inner := &Box{
		Width: shaftW, Height: height, Depth: depth, Color: opts.Color,
		Content: OpLimits{
			Base:      arrowBox,
			Sup:       aboveBox,
			Sub:       belowBox,
			BaseShift: -axis,
			SupKern:   gap,
			SubKern:   gap,
			Slant:     0,
			SupScale:  supRatio,
			SubScale:  subRatio,
		},
	}
	if targetColWidth > inner.Width+1e-6 {
		extra := targetColWidth - inner.Width
		kl := extra / 2
		kr := extra - kl
		return cdWrapHPad(inner, kl, kr, opts.Color)
	}
	return inner
}

func layoutCDVertArrow(direction string, labelAbove, labelBelow parser.Node, targetSize, targetColWidth float64, opts Options) *Box {
	bigTotal := sizeToMaxHeight[2] // 1.8em
	var shaftBox *Box
	switch {
	case direction == "vert_eq" && targetSize > 0:
		ts := targetSize
		if bigTotal > ts {
			ts = bigTotal
		}
		shaftBox = makeVertDelimBox(ts, true, opts)
	case direction == "vert_eq":
		shaftBox = makeStretchyDelim(`\Vert`, bigTotal, opts)
	case direction == "down" && targetSize > 0:
		ts := targetSize
		if 1.0 > ts {
			ts = 1.0
		}
		shaftBox = cdStretchVertArrowBox(ts, true, opts)
	case direction == "up" && targetSize > 0:
		ts := targetSize
		if 1.0 > ts {
			ts = 1.0
		}
		shaftBox = cdStretchVertArrowBox(ts, false, opts)
	case direction == "down":
		shaftBox = cdStretchVertArrowBox(bigTotal, true, opts)
	case direction == "up":
		shaftBox = cdStretchVertArrowBox(bigTotal, false, opts)
	default:
		shaftBox = cdStretchVertArrowBox(bigTotal, true, opts)
	}
	boxH := shaftBox.Height
	boxD := shaftBox.Depth
	shaftW := shaftBox.Width

	var leftBox, rightBox *Box
	if labelAbove != nil {
		leftBox = cdVCenterSideLabel(cdSideLabelScaled(labelAbove, opts), boxH, boxD, opts.Color)
	}
	if labelBelow != nil {
		rightBox = cdVCenterSideLabel(cdSideLabelScaled(labelBelow, opts), boxH, boxD, opts.Color)
	}
	leftW := 0.0
	rightW := 0.0
	if leftBox != nil {
		leftW = leftBox.Width
	}
	if rightBox != nil {
		rightW = rightBox.Width
	}
	leftPart := leftW
	if leftW > 0 {
		leftPart += cdVertSideKernEm
	}
	rightPart := rightW
	if rightW > 0 {
		rightPart += cdVertSideKernEm
	}
	innerW := leftPart + shaftW + rightPart
	kernLeft, kernRight, totalW := 0.0, 0.0, innerW
	if targetColWidth > innerW {
		extra := targetColWidth - innerW
		kernLeft = extra / 2
		kernRight = extra - kernLeft
		totalW = targetColWidth
	}

	var children []*Box
	if kernLeft > 0 {
		children = append(children, NewKern(kernLeft))
	}
	if leftBox != nil {
		children = append(children, leftBox)
		children = append(children, NewKern(cdVertSideKernEm))
	}
	children = append(children, shaftBox)
	if rightBox != nil {
		children = append(children, NewKern(cdVertSideKernEm))
		children = append(children, rightBox)
	}
	if kernRight > 0 {
		children = append(children, NewKern(kernRight))
	}
	return &Box{
		Width: totalW, Height: boxH, Depth: boxD, Color: opts.Color,
		Content: HBox{Children: children},
	}
}

func cdWrapHPad(inner *Box, padL, padR float64, color Color) *Box {
	w := inner.Width + padL + padR
	var children []*Box
	if padL > 0 {
		children = append(children, NewKern(padL))
	}
	children = append(children, inner)
	if padR > 0 {
		children = append(children, NewKern(padR))
	}
	return &Box{
		Width: w, Height: inner.Height, Depth: inner.Depth, Color: color,
		Content: HBox{Children: children},
	}
}

func cdVCenterSideLabel(label *Box, boxH, boxD float64, color Color) *Box {
	shift := (boxH - boxD + label.Depth - label.Height) / 2
	return &Box{
		Width: label.Width, Height: boxH, Depth: boxD, Color: color,
		Content: RaiseBox{Body: label, Shift: shift},
	}
}

func cdSideLabelScaled(node parser.Node, opts Options) *Box {
	supStyle := opts.Style.Superscript()
	supRatio := supStyle.SizeMultiplier() / opts.Style.SizeMultiplier()
	inner := layoutNode(node, opts.WithStyle(supStyle))
	if supRatio > 0.999 && supRatio < 1.001 {
		return inner
	}
	return &Box{
		Width: inner.Width * supRatio, Height: inner.Height * supRatio, Depth: inner.Depth * supRatio,
		Color:   opts.Color,
		Content: Scaled{Body: inner, ChildScale: supRatio},
	}
}

func cdStretchVertArrowBox(totalH float64, down bool, opts Options) *Box {
	axis := opts.Metrics().AxisHeight
	depth := totalH/2 - axis
	if depth < 0 {
		depth = 0
	}
	height := totalH - depth
	if cmds, w, ok := katexCDVertArrowFromRightarrow(down, totalH, axis); ok {
		return &Box{
			Width: w, Height: height, Depth: depth, Color: opts.Color,
			Content: SvgPath{Commands: cmds, Fill: true},
		}
	}
	if down {
		return makeStretchyDelim(`\downarrow`, sizeToMaxHeight[2], opts)
	}
	return makeStretchyDelim(`\uparrow`, sizeToMaxHeight[2], opts)
}
