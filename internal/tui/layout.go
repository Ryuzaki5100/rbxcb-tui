package tui

import (
	"math"
	"strings"

	"github.com/Ryuzaki5100/rbxcb-tui/internal/solver"
)

type rect struct{ x, y, w, h int }

func (r rect) inner() rect { return rect{r.x + 2, r.y + 1, r.w - 4, r.h - 2} }

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

type paneLayout struct {
	rect  rect
	inner rect
}

func newPane(r rect) paneLayout { return paneLayout{rect: r, inner: r.inner()} }

type cubeGeometry struct {
	inner rect

	titleY int
	upY    int
	cubeY  int
	downY  int

	cubeX, cubeW int
	cubeH        int
	cellW        int
	cellH        int
	gapX         int
	rowGap       int

	centerX     int
	leftArrowX  int
	rightArrowX int
	arrowY      int

	paletteY0 int
	paletteY1 int
	infoY     int
	centersY  int
}

func (g cubeGeometry) cellRect(row, col int) rect {
	return rect{
		x: g.cubeX + col*(g.cellW+g.gapX),
		y: g.cubeY + row*(g.cellH+g.rowGap),
		w: g.cellW,
		h: g.cellH,
	}
}

func computeCubeGeometry(pane rect, aspect float64) cubeGeometry {
	inner := pane.inner()
	cellH := 4
	cellW := clampInt(int(math.Round(float64(cellH)*aspect)), 5, 10)
	gapX := clampInt(int(math.Round(aspect)), 1, 4)
	if maxCell := (inner.w - 2*gapX) / 3; cellW > maxCell && maxCell >= 5 {
		cellW = maxCell
	}
	g := cubeGeometry{
		inner:  inner,
		cellW:  cellW,
		cellH:  cellH,
		gapX:   gapX,
		rowGap: 1,
	}
	g.cubeW = 3*g.cellW + 2*g.gapX
	g.cubeH = 3*g.cellH + 2*g.rowGap

	blockH := g.cubeH + 5
	topPad := (inner.h - blockH) / 2
	if topPad < 0 {
		topPad = 0
	}
	g.titleY = inner.y + topPad
	g.upY = g.titleY + 1
	g.cubeY = g.upY + 1
	g.downY = g.cubeY + g.cubeH
	g.infoY = g.downY + 1
	g.centersY = g.infoY + 1
	g.paletteY0 = 0
	g.paletteY1 = 0

	g.cubeX = inner.x + (inner.w-g.cubeW)/2
	if g.cubeX < inner.x {
		g.cubeX = inner.x
	}
	g.centerX = g.cubeX + g.cubeW/2
	g.arrowY = g.cubeY + g.cellH + g.rowGap + 1
	g.leftArrowX = g.cubeX - 2
	g.rightArrowX = g.cubeX + g.cubeW + 1
	return g
}

type paletteGeometry struct {
	pane    paneLayout
	colW    int
	chipW   int
	chipH   int
	swatchW int
	swatchH int
	firstY  int
	stride  int
	colGap  int
}

func paletteOuterWidth(aspect float64) int {
	swatchW := clampInt(int(math.Round(2*aspect)), 3, 6)
	colW := maxInt(swatchW+2, 7)
	return 2*colW + 3 + 4
}

func computePaletteGeometry(pane rect, aspect float64) paletteGeometry {
	swatchW := clampInt(int(math.Round(2*aspect)), 3, 6)
	colW := maxInt(swatchW+2, 7)
	g := paletteGeometry{
		pane:    newPane(pane),
		colW:    colW,
		chipW:   swatchW + 2,
		chipH:   4,
		swatchW: swatchW,
		swatchH: 2,
		firstY:  pane.inner().y + 2,
		stride:  6,
		colGap:  3,
	}
	return g
}

type puzzleRow struct {
	rect  rect
	index int
}

type actionKind int

const (
	actionStage2 actionKind = iota
	actionStage3
	actionStage4
	actionSolve
	actionScramble
	actionClear
	actionMouse
	actionTheme
	actionView
	actionHelp
	actionPlay
	actionStepBack
	actionStepForward
	actionFirst
	actionLast
)

type actionRow struct {
	rect    rect
	kind    actionKind
	label   string
	enabled bool
}

type section struct {
	rect  rect
	title string
}

type outputLayout struct {
	inner   rect
	titleY  int
	statusY int
	movesY  int
	perRow  int
	visible int
}

type appLayout struct {
	left    paneLayout
	cube    paneLayout
	palette paneLayout
	output  paneLayout
	right   paneLayout

	cubeGeom    cubeGeometry
	paletteGeom paletteGeometry

	puzzles  []puzzleRow
	sections []section
	actions  []actionRow
	out      outputLayout
}

func (m Model) layout() appLayout {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		width, height = 120, 40
	}
	const headerH, footerH = 1, 2
	bodyY := headerH
	bodyH := height - headerH - footerH
	if bodyH < 1 {
		bodyH = 1
	}

	hasPuzzle := m.selectedPuzzle >= 0
	leftW := clampInt(width*18/100, 20, 26)
	rightW := 0
	midW := width - leftW
	if hasPuzzle {
		rightW = clampInt(width*20/100, 24, 30)
		midW = width - leftW - rightW
		if midW < 40 {
			midW = 40
			leftW = clampInt(width-midW-rightW, 16, 26)
			rightW = width - leftW - midW
		}
		if rightW < 20 {
			rightW = 20
		}
	}

	leftRect := rect{0, bodyY, leftW, bodyH}
	midRect := rect{leftW, bodyY, midW, bodyH}
	rightRect := rect{leftW + midW, bodyY, rightW, bodyH}

	outputH := clampInt(bodyH*28/100, 7, 14)
	if outputH > bodyH-8 {
		outputH = bodyH - 8
	}
	if outputH < 6 {
		outputH = 6
	}
	cubeH := bodyH - outputH
	paletteW := 0
	cubeW := midRect.w
	if hasPuzzle {
		paletteW = paletteOuterWidth(m.cellAspect)
		if paletteW > midRect.w/2 {
			paletteW = midRect.w / 2
		}
		cubeW = midRect.w - paletteW - 1
		if cubeW < 30 {
			cubeW = midRect.w
			paletteW = 0
		}
	}
	cubeRect := rect{midRect.x, midRect.y, cubeW, cubeH}
	paletteRect := rect{midRect.x + cubeW + 1, midRect.y, paletteW, cubeH}
	outputRect := rect{midRect.x, midRect.y + cubeH, midRect.w, outputH}

	l := appLayout{
		left:    newPane(leftRect),
		cube:    newPane(cubeRect),
		palette: newPane(paletteRect),
		output:  newPane(outputRect),
		right:   newPane(rightRect),
	}
	l.cubeGeom = computeCubeGeometry(cubeRect, m.cellAspect)
	l.paletteGeom = computePaletteGeometry(paletteRect, m.cellAspect)

	for i := range pickerOptions {
		y := l.left.inner.y + 2 + i*4
		l.puzzles = append(l.puzzles, puzzleRow{
			rect:  rect{l.left.inner.x, y, l.left.inner.w, 3},
			index: i,
		})
	}

	l.out = outputLayout{
		inner:   l.output.inner,
		titleY:  l.output.inner.y,
		statusY: l.output.inner.y + 1,
		movesY:  l.output.inner.y + 2,
		perRow:  maxInt(1, l.output.inner.w/7),
		visible: maxInt(1, l.output.inner.h-2),
	}

	if !hasPuzzle {
		return l
	}

	ix, iy := l.right.inner.x, l.right.inner.y
	hasSolution := len(m.solution.Moves) > 0

	solverRect := rect{ix, iy, l.right.inner.w, 5}
	actionsRect := rect{ix, iy + 6, l.right.inner.w, 9}
	playbackRect := rect{ix, iy + 16, l.right.inner.w, 5}
	l.sections = []section{
		{solverRect, "Solver"},
		{actionsRect, "Actions"},
		{playbackRect, "Playback"},
	}

	pillX := func(r rect) (int, int) { return r.x + 1, r.w - 2 }
	add := func(kind actionKind, label string, y int, enabled bool, r rect) {
		x, w := pillX(r)
		l.actions = append(l.actions, actionRow{rect: rect{x, y, w, 1}, kind: kind, label: label, enabled: enabled})
	}
	add(actionStage2, stageRadio(m.stage, solver.Stage2)+" 2-stage", solverRect.y+1, hasPuzzle, solverRect)
	add(actionStage3, stageRadio(m.stage, solver.Stage3)+" 3-stage", solverRect.y+2, hasPuzzle, solverRect)
	add(actionStage4, stageRadio(m.stage, solver.Stage4)+" 4-stage", solverRect.y+3, hasPuzzle, solverRect)

	add(actionSolve, "[s] Solve", actionsRect.y+1, hasPuzzle, actionsRect)
	add(actionScramble, "[n] Scramble", actionsRect.y+2, hasPuzzle, actionsRect)
	add(actionClear, "[c] Clear", actionsRect.y+3, hasPuzzle, actionsRect)
	add(actionMouse, fmtMouse(m.mouseOn), actionsRect.y+4, true, actionsRect)
	add(actionTheme, "[t] Themes", actionsRect.y+5, true, actionsRect)
	add(actionView, "[v] Input", actionsRect.y+6, hasSolution, actionsRect)
	add(actionHelp, "[?] Help", actionsRect.y+7, true, actionsRect)

	add(actionPlay, "[space] Play", playbackRect.y+1, hasSolution, playbackRect)
	add(actionStepBack, "[h]", playbackRect.y+2, hasSolution, playbackRect)
	add(actionStepForward, "[l]", playbackRect.y+2, hasSolution, playbackRect)
	add(actionFirst, "[g]", playbackRect.y+3, hasSolution, playbackRect)
	add(actionLast, "[G]", playbackRect.y+3, hasSolution, playbackRect)

	// Split the paired playback rows into two pills.
	for i := range l.actions {
		row := &l.actions[i]
		switch row.kind {
		case actionStepBack, actionStepForward, actionFirst, actionLast:
			x, w := pillX(playbackRect)
			row.rect.w = w/2 - 1
			if row.kind == actionStepForward || row.kind == actionLast {
				row.rect.x = x + w/2
			}
		}
	}
	return l
}

func stageRadio(current, stage solver.Stage) string {
	if current == stage {
		return "(•)"
	}
	return "( )"
}

func fmtMouse(on bool) string {
	if on {
		return "[m] Mouse: on"
	}
	return "[m] Mouse: off"
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

type themeMenuLayout struct {
	rect    rect
	rows    []rect
	visible int
}

func (m Model) themeMenuLayout() themeMenuLayout {
	w := 36
	h := len(m.themes) + 4
	if h > m.height-2 {
		h = m.height - 2
	}
	if h < 6 {
		h = 6
	}
	x := (m.width - w) / 2
	y := (m.height - h) / 2
	visible := h - 4
	if visible < 1 {
		visible = 1
	}
	l := themeMenuLayout{rect: rect{x, y, w, h}, visible: visible}
	for i := 0; i < visible; i++ {
		l.rows = append(l.rows, rect{x + 2, y + 2 + i, w - 4, 1})
	}
	return l
}

func (m Model) themeMenuVisibleRows() int { return m.themeMenuLayout().visible }
