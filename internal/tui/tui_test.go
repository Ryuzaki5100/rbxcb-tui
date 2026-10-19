package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/solver"
)

const sample = "ybgrbrrbbrgbooygorwyyrwyggboboorgwbryobyygyrwwwowgwowg"

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "rbxcb-tui-tests-*")
	if err == nil {
		os.Setenv("XDG_CONFIG_HOME", dir)
	}
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func press(text string) tea.KeyPressMsg {
	code := rune(0)
	if text != "" {
		code = rune(text[0])
	}
	if text == "space" {
		code = tea.KeySpace
	}
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}

func sized(t *testing.T, m Model) Model {
	t.Helper()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return updated.(Model)
}

func sampleModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := sized(t, New(Options{Facelets: sample}))
	if m.selectedPuzzle != 0 {
		t.Fatalf("puzzle not selected: %d", m.selectedPuzzle)
	}
	return m
}

func TestEmptyState(t *testing.T) {
	m := sized(t, New(Options{}))
	if m.selectedPuzzle != -1 {
		t.Fatalf("selected = %d", m.selectedPuzzle)
	}
	if m.focus != FocusPuzzles {
		t.Fatalf("focus = %v", m.focus)
	}
	content := stripANSI(m.View().Content)
	if !strings.Contains(content, "Select a puzzle") {
		t.Fatal("missing empty-state hint")
	}
	for _, row := range m.layout().actions {
		switch row.kind {
		case actionMouse, actionHelp, actionTheme:
			continue
		}
		if row.enabled {
			t.Fatalf("action %v enabled with no puzzle", row.kind)
		}
	}
}

func TestPuzzleKeyboardSelect(t *testing.T) {
	m := sized(t, New(Options{}))
	updated, _ := m.Update(press("enter"))
	m = updated.(Model)
	if m.selectedPuzzle != 0 {
		t.Fatalf("selected = %d", m.selectedPuzzle)
	}
	if m.focus != FocusCube {
		t.Fatalf("focus = %v", m.focus)
	}
}

func TestPuzzleMouseSelect(t *testing.T) {
	m := sized(t, New(Options{}))
	m.rebuildHits()
	var button Hit
	for _, h := range m.hits {
		if h.Kind == HitPuzzle && h.Puzzle == 0 {
			button = h
		}
	}
	updated, _ := m.handleMouseClick(tea.Mouse{X: button.Rect.x + 2, Y: button.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.selectedPuzzle != 0 || m.focus != FocusCube {
		t.Fatalf("selected=%d focus=%v", m.selectedPuzzle, m.focus)
	}
}

func TestDisabledPuzzleMouseIgnored(t *testing.T) {
	m := sized(t, New(Options{}))
	m.rebuildHits()
	var button Hit
	for _, h := range m.hits {
		if h.Kind == HitPuzzle && h.Puzzle == 1 {
			button = h
		}
	}
	updated, _ := m.handleMouseClick(tea.Mouse{X: button.Rect.x + 2, Y: button.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.selectedPuzzle != -1 {
		t.Fatalf("disabled puzzle selected: %d", m.selectedPuzzle)
	}
	if !strings.Contains(m.status, "not supported") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestEditorSerializesSample(t *testing.T) {
	m := sampleModel(t)
	got, err := m.editor.FaceletsString()
	if err != nil {
		t.Fatal(err)
	}
	if got != sample {
		t.Fatalf("got %q want %q", got, sample)
	}
}

func TestEditorPaintAdvances(t *testing.T) {
	m := New(Options{})
	m.editor.SetFace(2)
	m.editor.Cell = 0
	m.editor.Paint(Red)
	if m.editor.Faces[2][0] != Red || m.editor.Cell != 1 || m.editor.Current != Red {
		t.Fatal("paint state incorrect")
	}
}

func TestFaceNavigation(t *testing.T) {
	m := New(Options{})
	m.editor.SetFace(2) // F, Up becomes U
	startFace, startUp := m.editor.Face, m.editor.Up
	for i := 0; i < 4; i++ {
		m.editor.MoveFace(3)
	}
	if m.editor.Face != startFace || m.editor.Up != startUp {
		t.Fatalf("right x4 returned to %d/%d, want %d/%d", m.editor.Face, m.editor.Up, startFace, startUp)
	}
	for i := 0; i < 4; i++ {
		m.editor.MoveFace(0)
	}
	if m.editor.Face != startFace || m.editor.Up != startUp {
		t.Fatalf("up x4 returned to %d/%d, want %d/%d", m.editor.Face, m.editor.Up, startFace, startUp)
	}
}

func TestSolveFlowPreservesInput(t *testing.T) {
	m := sampleModel(t)
	before, _ := m.editor.FaceletsString()
	updated, _ := m.Update(press("s"))
	m = updated.(Model)
	if !m.solving || m.focus != FocusOutput {
		t.Fatalf("solving=%v focus=%v", m.solving, m.focus)
	}
	start, err := m.editor.Validate()
	if err != nil {
		t.Fatal(err)
	}
	moves, err := solver.Solve(context.Background(), start, m.stage, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ = m.Update(solveDoneMsg{gen: m.solveGen, stage: m.stage, moves: moves, start: start, elapsed: time.Second})
	m = updated.(Model)
	if len(m.solution.Moves) == 0 {
		t.Fatal("no solution")
	}
	after, _ := m.editor.FaceletsString()
	if before != after {
		t.Fatal("input changed after solve")
	}
	if !strings.Contains(stripANSI(m.View().Content), "SOLUTION") {
		t.Fatal("solution panel missing")
	}
}

func TestStageSelectionPreservesInput(t *testing.T) {
	m := sampleModel(t)
	before, _ := m.editor.FaceletsString()
	updated, _ := m.Update(press("2"))
	m = updated.(Model)
	if m.stage != solver.Stage2 {
		t.Fatalf("stage = %v", m.stage)
	}
	after, _ := m.editor.FaceletsString()
	if before != after {
		t.Fatal("input changed")
	}
}

func TestMouseCellHitsExactCell(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var target Hit
	for _, h := range m.hits {
		if h.Kind == HitCell && h.Cell == 4 {
			target = h
		}
	}
	before := m.editor.Faces[m.editor.Face][4]
	updated, _ := m.handleMouseClick(tea.Mouse{X: target.Rect.x + 1, Y: target.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.editor.Faces[m.editor.Face][4] == before {
		t.Fatal("clicked cell was not painted")
	}
	if m.editor.Faces[m.editor.Face][4] != m.editor.Current {
		t.Fatal("clicked cell did not receive current color")
	}
}

func TestArrowAlignment(t *testing.T) {
	m := sampleModel(t)
	l := m.layout()
	g := l.cubeGeom
	if g.centerX != g.cubeX+g.cubeW/2 {
		t.Fatalf("up/down arrow x = %d, cube center = %d", g.centerX, g.cubeX+g.cubeW/2)
	}
	if g.leftArrowX >= g.cubeX || g.rightArrowX <= g.cubeX+g.cubeW {
		t.Fatalf("side arrows not outside grid: left=%d right=%d cube=%d..%d", g.leftArrowX, g.rightArrowX, g.cubeX, g.cubeX+g.cubeW)
	}
	if g.arrowY != g.cubeY+g.cellH+g.rowGap+1 {
		t.Fatalf("side arrow row = %d", g.arrowY)
	}
	m.rebuildHits()
	for _, h := range m.hits {
		if h.Kind == HitFaceArrow && (h.Rect.y == g.upY || h.Rect.y == g.downY) && h.Rect.x != g.centerX {
			t.Fatalf("vertical arrow hitbox x = %d", h.Rect.x)
		}
	}
}

func TestHitboxesMatchRenderedCells(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, h := range m.hits {
		if h.Kind != HitCell {
			continue
		}
		if h.Rect.y < 0 || h.Rect.y >= len(lines) {
			t.Fatalf("cell hit y=%d outside view", h.Rect.y)
		}
		found := false
		for yy := h.Rect.y; yy < h.Rect.y+h.Rect.h; yy++ {
			if yy >= len(lines) {
				continue
			}
			runes := []rune(lines[yy])
			if h.Rect.x+h.Rect.w > len(runes) {
				continue
			}
			segment := string(runes[h.Rect.x : h.Rect.x+h.Rect.w])
			if strings.ContainsAny(segment, "█░") {
				found = true
			}
		}
		if !found {
			t.Fatalf("no glyph inside cell hit %d", h.Cell)
		}
	}
}

func TestNoBackgroundSGR(t *testing.T) {
	m := sampleModel(t)
	if strings.Contains(m.renderCubePane(m.layout()), "\x1b[48;") {
		t.Fatal("background SGR found in cube pane")
	}
}

func TestMousePalette(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var swatch Hit
	for _, h := range m.hits {
		if h.Kind == HitPalette && h.Color == Blue {
			swatch = h
		}
	}
	updated, _ := m.handleMouseClick(tea.Mouse{X: swatch.Rect.x + 1, Y: swatch.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.editor.Current != Blue {
		t.Fatalf("current = %v", m.editor.Current)
	}
}

func TestActionMouseStageAndSolve(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var stageHit, solveHit Hit
	for _, h := range m.hits {
		if h.Kind == HitAction && h.Action == actionStage2 {
			stageHit = h
		}
		if h.Kind == HitAction && h.Action == actionSolve {
			solveHit = h
		}
	}
	updated, _ := m.handleMouseClick(tea.Mouse{X: stageHit.Rect.x + 1, Y: stageHit.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.stage != solver.Stage2 {
		t.Fatalf("stage = %v", m.stage)
	}
	updated, _ = m.handleMouseClick(tea.Mouse{X: solveHit.Rect.x + 1, Y: solveHit.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if !m.solving {
		t.Fatal("solve action did not start")
	}
}

func TestPaneFocusMouse(t *testing.T) {
	m := sampleModel(t)
	l := m.layout()
	cases := []struct {
		p    rect
		want Focus
	}{
		{l.left.rect, FocusPuzzles},
		{l.cube.rect, FocusCube},
		{l.output.rect, FocusOutput},
		{l.right.rect, FocusActions},
	}
	for _, c := range cases {
		updated, _ := m.handleMouseClick(tea.Mouse{X: c.p.x + 2, Y: c.p.y + 2, Button: tea.MouseLeft})
		m = updated.(Model)
		if m.focus != c.want {
			t.Fatalf("click pane at %+v -> focus %v, want %v", c.p, m.focus, c.want)
		}
	}
}

func TestWheelOutput(t *testing.T) {
	m := sampleModel(t)
	m.solution = Solution{Moves: make([]cube.Move, 100), Step: 0}
	m.outputScroll = 0
	l := m.layout()
	updated, _ := m.handleWheel(tea.Mouse{X: l.output.rect.x + 2, Y: l.output.rect.y + 2, Button: tea.MouseWheelDown})
	m = updated.(Model)
	if m.outputScroll == 0 {
		t.Fatal("wheel did not scroll output")
	}
}

func TestRandomScramble(t *testing.T) {
	m := sampleModel(t)
	updated, _ := m.Update(press("n"))
	m = updated.(Model)
	if m.editor.Filled() != 54 {
		t.Fatalf("filled = %d", m.editor.Filled())
	}
	if _, err := m.editor.Validate(); err != nil {
		t.Fatalf("invalid scramble: %v", err)
	}
}

func TestHelpToggle(t *testing.T) {
	m := sized(t, New(Options{}))
	updated, _ := m.Update(press("?"))
	m = updated.(Model)
	if !m.helpVisible {
		t.Fatal("help not visible")
	}
	updated, _ = m.Update(press("esc"))
	m = updated.(Model)
	if m.helpVisible {
		t.Fatal("help still visible")
	}
}

func TestTinyTerminal(t *testing.T) {
	m := New(Options{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = updated.(Model)
	if !strings.Contains(m.View().Content, "too small") {
		t.Fatal("expected tiny-terminal message")
	}
}

func TestClearKeyOnlyClearsCellInCubePane(t *testing.T) {
	m := sampleModel(t)
	m.focus = FocusCube
	before := m.editor.Filled()
	updated, _ := m.Update(press("x"))
	m = updated.(Model)
	if m.editor.Filled() != before-1 {
		t.Fatalf("x cleared %d cells, want 1", before-m.editor.Filled())
	}
}

func TestClearActionWipesCube(t *testing.T) {
	m := sampleModel(t)
	m.focus = FocusActions
	updated, _ := m.Update(press("c"))
	m = updated.(Model)
	if m.editor.Filled() != 0 {
		t.Fatalf("c left %d cells filled", m.editor.Filled())
	}
}

func TestMouseFaceArrowChangesFace(t *testing.T) {
	m := sampleModel(t)
	m.editor.SetFace(2) // F, Up becomes U
	m.rebuildHits()
	var arrow Hit
	for _, h := range m.hits {
		if h.Kind == HitFaceArrow && h.Direction == 0 {
			arrow = h
		}
	}
	updated, _ := m.handleMouseClick(tea.Mouse{X: arrow.Rect.x, Y: arrow.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.editor.Face != 0 || m.editor.Up != 4 {
		t.Fatalf("face/up = %d/%d, want U/B", m.editor.Face, m.editor.Up)
	}
}

func TestLayoutAtMinSize(t *testing.T) {
	m := New(Options{Facelets: sample})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: minWidth, Height: minHeight})
	m = updated.(Model)
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	if len(lines) != minHeight {
		t.Fatalf("view has %d lines, want %d", len(lines), minHeight)
	}
	for _, line := range lines {
		if w := len([]rune(line)); w > minWidth {
			t.Fatalf("line width %d exceeds %d: %q", w, minWidth, line)
		}
	}
}

func TestOutputContainsSolutionMoves(t *testing.T) {
	m := sampleModel(t)
	start, _ := m.editor.Validate()
	moves, err := solver.Solve(context.Background(), start, solver.Stage4, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(solveDoneMsg{gen: m.solveGen, stage: solver.Stage4, moves: moves, start: start, elapsed: time.Millisecond})
	m = updated.(Model)
	content := stripANSI(m.View().Content)
	if len(moves) > 0 && !strings.Contains(content, moves[0].String()) {
		t.Fatal("first solution move not visible in output panel")
	}
	if !strings.Contains(content, "SOLUTION") {
		t.Fatal("solution panel missing")
	}
}

func TestPairedActionRowsShareLine(t *testing.T) {
	m := sampleModel(t)
	l := m.layout()
	var back, forward rect
	for _, row := range l.actions {
		if row.kind == actionStepBack {
			back = row.rect
		}
		if row.kind == actionStepForward {
			forward = row.rect
		}
	}
	if back.y != forward.y || back.x >= forward.x {
		t.Fatalf("paired actions not on one line: back=%+v forward=%+v", back, forward)
	}
}

func TestPuzzleHitboxesMatchRenderedRows(t *testing.T) {
	m := sized(t, New(Options{}))
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, h := range m.hits {
		if h.Kind != HitPuzzle {
			continue
		}
		found := false
		for yy := h.Rect.y; yy < h.Rect.y+h.Rect.h && yy < len(lines); yy++ {
			if strings.Contains(lines[yy], pickerOptions[h.Puzzle].label) {
				found = true
			}
		}
		if !found {
			t.Fatalf("puzzle %d hitbox rows do not contain label", h.Puzzle)
		}
	}
}

func TestActionHitboxesMatchRenderedRows(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, row := range m.layout().actions {
		if row.kind == actionView {
			continue
		}
		line := lines[row.rect.y]
		if !strings.Contains(line, strings.TrimSpace(row.label)) {
			t.Fatalf("action %q hitbox row %d does not contain label: %q", row.label, row.rect.y, line)
		}
	}
}

func TestArrowHitboxesMatchRenderedGlyphs(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, h := range m.hits {
		if h.Kind != HitFaceArrow {
			continue
		}
		runes := []rune(lines[h.Rect.y])
		if h.Rect.x >= len(runes) {
			t.Fatalf("arrow x=%d outside line", h.Rect.x)
		}
		if !strings.ContainsRune("↑↓←→", runes[h.Rect.x]) {
			t.Fatalf("no arrow glyph at %d,%d: %q", h.Rect.x, h.Rect.y, string(runes[h.Rect.x]))
		}
	}
}

func TestOutputMoveHitboxesMatchRenderedMoves(t *testing.T) {
	m := sampleModel(t)
	start, _ := m.editor.Validate()
	moves, err := solver.Solve(context.Background(), start, solver.Stage4, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(solveDoneMsg{gen: m.solveGen, stage: solver.Stage4, moves: moves, start: start, elapsed: time.Millisecond})
	m = updated.(Model)
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, h := range m.hits {
		if h.Kind != HitMove {
			continue
		}
		if h.Move-1 >= len(moves) {
			t.Fatalf("move hit index %d out of range", h.Move)
		}
		line := lines[h.Rect.y]
		runes := []rune(line)
		if h.Rect.x+h.Rect.w > len(runes) {
			t.Fatalf("move hit x=%d outside line", h.Rect.x)
		}
		segment := string(runes[h.Rect.x : h.Rect.x+h.Rect.w])
		if !strings.Contains(segment, moves[h.Move-1].String()) {
			t.Fatalf("move %d hitbox does not contain %q: %q", h.Move, moves[h.Move-1].String(), segment)
		}
	}
}

func TestMouseCellAtCornersMinSize(t *testing.T) {
	m := New(Options{Facelets: sample})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: minWidth, Height: minHeight})
	m = updated.(Model)
	m.rebuildHits()
	for _, cell := range []int{0, 8} {
		var hit Hit
		for _, h := range m.hits {
			if h.Kind == HitCell && h.Cell == cell {
				hit = h
			}
		}
		before := m.editor.Faces[m.editor.Face][cell]
		updated, _ := m.handleMouseClick(tea.Mouse{X: hit.Rect.x + hit.Rect.w/2, Y: hit.Rect.y + hit.Rect.h/2, Button: tea.MouseLeft})
		m = updated.(Model)
		if m.editor.Faces[m.editor.Face][cell] == before {
			t.Fatalf("cell %d was not painted at its rendered position", cell)
		}
	}
}

func TestSelectedCellDrawsHeavyBorder(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitCell && h.Cell == m.editor.Cell {
			hit = h
		}
	}
	top := []rune(lines[hit.Rect.y])
	mid := []rune(lines[hit.Rect.y+1])
	bot := []rune(lines[hit.Rect.y+3])
	if !strings.Contains(string(top[hit.Rect.x:hit.Rect.x+hit.Rect.w]), "┏") {
		t.Fatalf("top border missing: %q", string(top))
	}
	if !strings.Contains(string(mid[hit.Rect.x:hit.Rect.x+hit.Rect.w]), "┃") {
		t.Fatalf("side border missing: %q", string(mid))
	}
	if !strings.Contains(string(bot[hit.Rect.x:hit.Rect.x+hit.Rect.w]), "┗") {
		t.Fatalf("bottom border missing: %q", string(bot))
	}
}

func TestPuzzleButtonsDrawBoxes(t *testing.T) {
	m := sized(t, New(Options{}))
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, h := range m.hits {
		if h.Kind != HitPuzzle {
			continue
		}
		if !strings.Contains(lines[h.Rect.y], "╭") || !strings.Contains(lines[h.Rect.y+2], "╰") {
			t.Fatalf("puzzle %d is not boxed", h.Puzzle)
		}
	}
}

func TestHoverUpdatesOnMotion(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitAction && h.Action == actionSolve {
			hit = h
		}
	}
	updated, _ := m.Update(tea.MouseMotionMsg(tea.Mouse{X: hit.Rect.x + 1, Y: hit.Rect.y, Button: tea.MouseNone}))
	m = updated.(Model)
	if m.hover.ID != hit.ID {
		t.Fatalf("hover = %+v, want %s", m.hover, hit.ID)
	}
	updated, _ = m.Update(tea.MouseMotionMsg(tea.Mouse{X: 0, Y: 0, Button: tea.MouseNone}))
	m = updated.(Model)
	if m.hover.ID != "" {
		t.Fatalf("hover did not clear: %+v", m.hover)
	}
}

func TestHoverDoesNotPaint(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitCell && h.Cell == 4 {
			hit = h
		}
	}
	before := m.editor.Faces[m.editor.Face][4]
	updated, _ := m.Update(tea.MouseMotionMsg(tea.Mouse{X: hit.Rect.x + 1, Y: hit.Rect.y + 1, Button: tea.MouseNone}))
	m = updated.(Model)
	if m.editor.Faces[m.editor.Face][4] != before {
		t.Fatal("hover painted a cell")
	}
}

func TestMouseModeAllMotionWithHover(t *testing.T) {
	m := New(Options{Facelets: sample})
	if m.View().MouseMode != tea.MouseModeAllMotion {
		t.Fatalf("mouse mode = %v", m.View().MouseMode)
	}
	m2 := New(Options{Facelets: sample, NoHover: true})
	if m2.View().MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("no-hover mouse mode = %v", m2.View().MouseMode)
	}
}

func TestThemeMenuOpensAndSelects(t *testing.T) {
	m := sampleModel(t)
	start := m.theme.Name
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	if !m.themeMenuVisible {
		t.Fatal("theme menu did not open")
	}
	updated, _ = m.Update(press("down"))
	m = updated.(Model)
	updated, _ = m.Update(press("enter"))
	m = updated.(Model)
	if m.themeMenuVisible {
		t.Fatal("theme menu did not close")
	}
	if m.theme.Name == start {
		t.Fatal("theme did not change")
	}
}

func TestThemeMenuEscKeepsTheme(t *testing.T) {
	m := sampleModel(t)
	start := m.theme.Name
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	updated, _ = m.Update(press("down"))
	m = updated.(Model)
	updated, _ = m.Update(press("esc"))
	m = updated.(Model)
	if m.themeMenuVisible {
		t.Fatal("theme menu still visible")
	}
	if m.theme.Name != start {
		t.Fatalf("theme changed to %s", m.theme.Name)
	}
}

func TestThemeMenuMouseSelect(t *testing.T) {
	m := sampleModel(t)
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	m.rebuildHits()
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitTheme && h.Puzzle == 3 {
			hit = h
		}
	}
	updated, _ = m.handleMouseClick(tea.Mouse{X: hit.Rect.x + 1, Y: hit.Rect.y, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.theme.Name != m.themes[3].Name {
		t.Fatalf("theme = %s, want %s", m.theme.Name, m.themes[3].Name)
	}
}

func TestGruvboxIsDefaultTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(Options{})
	if m.theme.Name != "gruvbox" {
		t.Fatalf("default theme = %s", m.theme.Name)
	}
}

func TestHoverColourIsDarker(t *testing.T) {
	for _, th := range BuiltinThemes() {
		if th.Mono {
			continue
		}
		r1, g1, b1, ok1 := parseHex(th.Accent)
		r2, g2, b2, ok2 := parseHex(th.AccentHover)
		if !ok1 || !ok2 {
			t.Fatalf("%s: invalid hex", th.Name)
		}
		l1 := 0.2126*float64(r1) + 0.7152*float64(g1) + 0.0722*float64(b1)
		l2 := 0.2126*float64(r2) + 0.7152*float64(g2) + 0.0722*float64(b2)
		if l2 >= l1 {
			t.Fatalf("%s: hover colour is not darker", th.Name)
		}
	}
}

func TestThemeConfigAndCustomTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := dir + "/rbxcb-tui"
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"theme":"nord","hover":false,"themes":{"mine":{"accent":"#112233","stickers":["#111111","#222222","#333333","#444444","#555555","#666666"]}}}`
	if err := os.WriteFile(cfgDir+"/config.json", []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	m := New(Options{})
	if m.theme.Name != "nord" {
		t.Fatalf("theme = %s", m.theme.Name)
	}
	if m.hoverEnabled {
		t.Fatal("hover should be disabled by config")
	}
	if _, ok := findTheme(m.themes, "mine"); !ok {
		t.Fatal("custom theme missing")
	}
}

func TestThemePrecedence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := dir + "/rbxcb-tui"
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.WriteFile(cfgDir+"/config.json", []byte(`{"theme":"nord"}`), 0o644)
	t.Setenv("RBXCB_THEME", "gruvbox")
	if m := New(Options{}); m.theme.Name != "gruvbox" {
		t.Fatalf("env theme = %s", m.theme.Name)
	}
	if m := New(Options{Theme: "dracula"}); m.theme.Name != "dracula" {
		t.Fatalf("flag theme = %s", m.theme.Name)
	}
}

func TestNoColorForcesMono(t *testing.T) {
	m := New(Options{NoColor: true})
	if m.theme.Name != "mono" || !m.styles.Mono {
		t.Fatalf("theme = %s mono=%v", m.theme.Name, m.styles.Mono)
	}
	if strings.Contains(m.View().Content, "\x1b[38;") {
		t.Fatal("no-color still emitted truecolor")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := dir + "/rbxcb-tui"
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.WriteFile(cfgDir+"/config.json", []byte("{not json"), 0o644)
	m := New(Options{})
	if m.errMsg == "" {
		t.Fatal("expected config error status")
	}
	if m.theme.Name == "" {
		t.Fatal("theme not set after invalid config")
	}
}

func TestNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := New(Options{})
	if m.theme.Name != "mono" {
		t.Fatalf("theme = %s", m.theme.Name)
	}
}

func TestBuiltinThemeNames(t *testing.T) {
	want := []string{"gruvbox", "default", "nord", "dracula", "catppuccin", "tokyo-night", "onedark", "rose-pine", "kanagawa", "solarized-dark", "github-dark", "mono"}
	got := BuiltinThemes()
	if len(got) != len(want) {
		t.Fatalf("got %d themes", len(got))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("theme %d = %s, want %s", i, got[i].Name, name)
		}
	}
}

func TestHoverChangesRenderedStyles(t *testing.T) {
	m := sampleModel(t)
	before := m.View().Content
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitAction && h.Action == actionScramble {
			hit = h
		}
	}
	updated, _ := m.Update(tea.MouseMotionMsg(tea.Mouse{X: hit.Rect.x + 1, Y: hit.Rect.y, Button: tea.MouseNone}))
	m = updated.(Model)
	if m.View().Content == before {
		t.Fatal("hover did not change rendered output")
	}
}

func TestNoBackgroundSGRAllThemes(t *testing.T) {
	for _, th := range BuiltinThemes() {
		m := sized(t, New(Options{Facelets: sample, Theme: th.Name}))
		if strings.Contains(m.renderCubePane(m.layout()), "\x1b[48;") {
			t.Fatalf("%s: background SGR found in cube pane", th.Name)
		}
	}
}

func TestLayoutStableAcrossThemes(t *testing.T) {
	base := sized(t, New(Options{Facelets: sample, Theme: "default"}))
	baseL := base.layout()
	baseHits := len(base.hits)
	for _, th := range BuiltinThemes() {
		m := sized(t, New(Options{Facelets: sample, Theme: th.Name}))
		l := m.layout()
		if l.cubeGeom != baseL.cubeGeom {
			t.Fatalf("%s: cube geometry changed", th.Name)
		}
		if len(m.hits) != baseHits {
			t.Fatalf("%s: hitbox count changed", th.Name)
		}
	}
}

func hexBG(hex string) string {
	r, g, b, _ := parseHex(hex)
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

func rawLineFor(m Model, y int) string {
	lines := strings.Split(m.View().Content, "\n")
	if y < 0 || y >= len(lines) {
		return ""
	}
	return lines[y]
}

func injectSolution(t *testing.T, m Model, stage solver.Stage) Model {
	t.Helper()
	start, err := m.editor.Validate()
	if err != nil {
		t.Fatal(err)
	}
	moves, err := solver.Solve(context.Background(), start, stage, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(solveDoneMsg{gen: m.solveGen, stage: stage, moves: moves, start: start, elapsed: time.Millisecond})
	return updated.(Model)
}

func TestPlaybackUpdatesCubeColours(t *testing.T) {
	m := sampleModel(t)
	m = injectSolution(t, m, solver.Stage4)
	if !m.showPlayback {
		t.Fatal("playback view not enabled")
	}
	m.solution.Step = 0
	startFaces, _ := m.displayFaces()
	m.solution.Step = 1
	nextFaces, playback := m.displayFaces()
	if !playback {
		t.Fatal("not in playback view")
	}
	if startFaces == nextFaces {
		t.Fatal("cube colours did not change after playback step")
	}
	m.solution.Step = len(m.solution.Moves)
	solvedFaces, _ := m.displayFaces()
	for f := 0; f < 6; f++ {
		want := solvedFaces[f][4]
		for c := 0; c < 9; c++ {
			if solvedFaces[f][c] != want {
				t.Fatalf("face %d not solved at final step", f)
			}
		}
	}
}

func TestViewTogglePlaybackAndInput(t *testing.T) {
	m := sampleModel(t)
	m = injectSolution(t, m, solver.Stage4)
	m.solution.Step = 3
	input := m.editor.Faces
	updated, _ := m.Update(press("v"))
	m = updated.(Model)
	if m.showPlayback {
		t.Fatal("v did not switch to input view")
	}
	faces, playback := m.displayFaces()
	if playback || faces != input {
		t.Fatal("input view does not show editor colours")
	}
	updated, _ = m.Update(press("v"))
	m = updated.(Model)
	if !m.showPlayback {
		t.Fatal("v did not switch back to playback")
	}
}

func TestEditDuringPlaybackSwitchesToInput(t *testing.T) {
	m := sampleModel(t)
	m = injectSolution(t, m, solver.Stage4)
	m.focus = FocusCube
	updated, _ := m.Update(press("w"))
	m = updated.(Model)
	if m.showPlayback {
		t.Fatal("editing did not switch to input view")
	}
}

func TestReSolveSameInputAnotherStage(t *testing.T) {
	m := sampleModel(t)
	before, _ := m.editor.FaceletsString()
	m = injectSolution(t, m, solver.Stage4)
	if m.solution.Stage != solver.Stage4 {
		t.Fatal("expected stage 4")
	}
	updated, _ := m.Update(press("2"))
	m = updated.(Model)
	if m.stage != solver.Stage2 {
		t.Fatal("stage did not change")
	}
	updated, _ = m.Update(press("s"))
	m = updated.(Model)
	if !m.solving {
		t.Fatal("re-solve did not start")
	}
	m = injectSolution(t, m, solver.Stage2)
	if m.solution.Stage != solver.Stage2 {
		t.Fatalf("solution stage = %s", m.solution.Stage)
	}
	if m.solution.Step != 0 || !m.showPlayback {
		t.Fatal("playback did not reset")
	}
	after, _ := m.editor.FaceletsString()
	if before != after {
		t.Fatal("input changed during re-solve")
	}
	if !m.solution.States[len(m.solution.States)-1].IsSolved() {
		t.Fatal("re-solve did not finish solved")
	}
}

func TestActionButtonFullWidthBackground(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitAction && h.Action == actionScramble {
			hit = h
		}
	}
	raw := rawLineFor(m, hit.Rect.y)
	if !strings.Contains(raw, hexBG(m.theme.ButtonBG)) {
		t.Fatalf("normal button background missing: %q", raw)
	}
	plain := stripANSI(raw)
	runes := []rune(plain)
	if hit.Rect.x+hit.Rect.w > len(runes) {
		t.Fatal("button hitbox outside line")
	}
	segment := string(runes[hit.Rect.x : hit.Rect.x+hit.Rect.w])
	if !strings.Contains(segment, "Scramble") {
		t.Fatalf("button does not fill its row: %q", segment)
	}
}

func TestHoveredButtonUsesDarkerBackground(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitAction && h.Action == actionScramble {
			hit = h
		}
	}
	updated, _ := m.Update(tea.MouseMotionMsg(tea.Mouse{X: hit.Rect.x + 1, Y: hit.Rect.y, Button: tea.MouseNone}))
	m = updated.(Model)
	raw := rawLineFor(m, hit.Rect.y)
	if !strings.Contains(raw, hexBG(m.theme.ButtonBGHover)) {
		t.Fatalf("hover background missing: %q", raw)
	}
	if strings.Contains(raw, hexBG(m.theme.ButtonBG)) {
		t.Fatal("hover still uses normal background")
	}
}

func TestSelectedStageUsesAccentBackground(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var hit Hit
	for _, h := range m.hits {
		if h.Kind == HitAction && h.Action == actionStage4 {
			hit = h
		}
	}
	raw := rawLineFor(m, hit.Rect.y)
	if !strings.Contains(raw, hexBG(m.theme.Accent)) {
		t.Fatalf("selected stage accent background missing: %q", raw)
	}
}

func TestMonoButtonsUseNoColorSGR(t *testing.T) {
	m := sized(t, New(Options{Facelets: sample, NoColor: true}))
	content := m.View().Content
	if strings.Contains(content, "\x1b[38;") || strings.Contains(content, "\x1b[48;") {
		t.Fatal("mono mode emitted colour SGR")
	}
}

func TestCubeCellsAreSquare(t *testing.T) {
	m := sampleModel(t)
	g := m.layout().cubeGeom
	want := int(math.Round(float64(g.cellH) * m.cellAspect))
	if g.cellW != want {
		t.Fatalf("cell is %dx%d, want width %d for aspect %.1f", g.cellW, g.cellH, want, m.cellAspect)
	}
	physical := float64(g.cubeW) / (float64(g.cubeH) * m.cellAspect)
	if physical < 0.9 || physical > 1.1 {
		t.Fatalf("cube physical ratio %.2f not square", physical)
	}
}

func TestPaletteSwatchesAreSquare(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	count := 0
	for _, h := range m.hits {
		if h.Kind != HitPalette {
			continue
		}
		count++
		if h.Rect.w != 7 || h.Rect.h != 5 {
			t.Fatalf("palette hitbox = %dx%d, want 7x5", h.Rect.w, h.Rect.h)
		}
	}
	if count != 6 {
		t.Fatalf("palette hitboxes = %d", count)
	}
}

func TestPaletteSidebarRendersSwatches(t *testing.T) {
	m := sampleModel(t)
	pg := m.layout().paletteGeom
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for i := 0; i < 6; i++ {
		row, col := i/2, i%2
		y := pg.firstY + row*pg.stride
		if y+5 >= len(lines) {
			t.Fatalf("palette entry %d outside view", i)
		}
		x := pg.pane.inner.x + col*(pg.colW+pg.colGap)
		found := false
		for yy := y; yy < y+4; yy++ {
			runes := []rune(lines[yy])
			if x+pg.swatchW <= len(runes) && strings.TrimSpace(string(runes[x:x+pg.swatchW])) != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing palette swatch for entry %d", i)
		}
	}
}

func TestFaceNavigationAllDirectionsFourCycle(t *testing.T) {
	for _, dir := range []int{0, 1, 2, 3} {
		m := New(Options{})
		m.editor.SetFace(2) // F, Up=U
		face, up := m.editor.Face, m.editor.Up
		for i := 0; i < 4; i++ {
			m.editor.MoveFace(dir)
		}
		if m.editor.Face != face || m.editor.Up != up {
			t.Fatalf("dir %d four presses ended at %d/%d, want %d/%d", dir, m.editor.Face, m.editor.Up, face, up)
		}
	}
}

func TestMousePaintKeepsClickedCellSelected(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var target Hit
	for _, h := range m.hits {
		if h.Kind == HitCell && h.Cell == 4 {
			target = h
		}
	}
	updated, _ := m.handleMouseClick(tea.Mouse{X: target.Rect.x + 1, Y: target.Rect.y + 1, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.editor.Cell != 4 {
		t.Fatalf("selection moved to %d after mouse click", m.editor.Cell)
	}
}

func TestKeyboardPaintStillAdvances(t *testing.T) {
	m := sampleModel(t)
	m.focus = FocusCube
	m.editor.Cell = 2
	updated, _ := m.Update(press("w"))
	m = updated.(Model)
	if m.editor.Cell != 3 {
		t.Fatalf("keyboard paint cursor = %d, want 3", m.editor.Cell)
	}
}

func TestCubeCellHoverChangesRender(t *testing.T) {
	m := sampleModel(t)
	m.rebuildHits()
	var target Hit
	for _, h := range m.hits {
		if h.Kind == HitCell && h.Cell == 4 {
			target = h
		}
	}
	before := m.View().Content
	updated, _ := m.Update(tea.MouseMotionMsg(tea.Mouse{X: target.Rect.x + 1, Y: target.Rect.y + 1, Button: tea.MouseNone}))
	m = updated.(Model)
	if m.hover.ID != target.ID {
		t.Fatalf("hover = %+v, want %s", m.hover, target.ID)
	}
	if m.editor.Cell == 4 && m.View().Content == before {
		t.Fatal("hover did not change rendered cube")
	}
}

func TestAspectCycle(t *testing.T) {
	m := sampleModel(t)
	if m.cellAspect != 1.75 {
		t.Fatalf("default aspect = %.2f", m.cellAspect)
	}
	updated, _ := m.Update(press("A"))
	m = updated.(Model)
	if m.cellAspect != 1.8 {
		t.Fatalf("aspect after A = %.2f", m.cellAspect)
	}
}

func TestCellWidthFollowsAspect(t *testing.T) {
	m := sampleModel(t)
	m.cellAspect = 2.0
	g := m.layout().cubeGeom
	if g.cellW != 8 || g.cellH != 4 {
		t.Fatalf("cell = %dx%d for aspect 2.0", g.cellW, g.cellH)
	}
}

func TestPaletteIsSidebar(t *testing.T) {
	m := sampleModel(t)
	l := m.layout()
	if l.palette.rect.x <= l.cube.rect.x+l.cube.rect.w {
		t.Fatalf("palette is not to the right of cube: cube=%+v palette=%+v", l.cube.rect, l.palette.rect)
	}
	m.rebuildHits()
	for _, h := range m.hits {
		if h.Kind == HitPalette && h.Rect.y >= l.output.rect.y {
			t.Fatal("palette hitbox is below the cube/output boundary")
		}
	}
}

func TestOrientedPaintMapsToCanonicalCell(t *testing.T) {
	m := sampleModel(t)
	m.editor.SetFace(2)  // F, Up=U
	m.editor.MoveFace(3) // yaw right -> R
	if m.editor.Face != 3 {
		t.Fatalf("front = %d, want R", m.editor.Face)
	}
	display := 0
	want := orientationIndex[m.editor.Face][m.editor.Up][display]
	if want < 0 {
		t.Fatal("invalid orientation")
	}
	m.editor.PaintAt(display, Red)
	if got := m.editor.Faces[m.editor.Face][want]; got != Red {
		t.Fatalf("painted canonical cell %d, want %d", want, display)
	}
}

func TestOrientedDisplayUsesCanonicalGridRotation(t *testing.T) {
	m := sampleModel(t)
	m.editor.SetFace(2)
	m.editor.MoveFace(3)
	faces := m.editor.Faces
	front := m.editor.Face
	up := m.editor.Up
	for i := 0; i < 9; i++ {
		got := orientedColor(faces, front, up, i)
		want := faces[front][orientationIndex[front][up][i]]
		if got != want {
			t.Fatalf("display cell %d = %v, want %v", i, got, want)
		}
	}
}

func TestMonoUsesPatterns(t *testing.T) {
	m := sized(t, New(Options{Facelets: sample, NoColor: true}))
	content := m.View().Content
	for _, p := range []string{"█", "▓", "▒", "░", "▚", "▞"} {
		if !strings.Contains(content, p) {
			t.Fatalf("mono pattern %q missing", p)
		}
	}
	if strings.Contains(content, "wwwwww") {
		t.Fatal("mono still repeats colour letters")
	}
}

func TestMonoHasNoDoubleBorders(t *testing.T) {
	m := sized(t, New(Options{Facelets: sample, NoColor: true}))
	if strings.Contains(m.View().Content, "╔") || strings.Contains(m.View().Content, "╗") {
		t.Fatal("mono uses double borders")
	}
}

func TestThemeMenuListsConfiguredThemes(t *testing.T) {
	m := sampleModel(t)
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	content := stripANSI(m.View().Content)
	for _, th := range m.themes {
		if !strings.Contains(content, th.Name) {
			t.Fatalf("theme %q missing from menu", th.Name)
		}
	}
}

func TestAspectFromPixels(t *testing.T) {
	got, ok := aspectFromPixels(1000, 875, 100, 50)
	if !ok || math.Abs(got-1.75) > 0.01 {
		t.Fatalf("aspect = %.2f ok=%v, want 1.75", got, ok)
	}
	if _, ok := aspectFromPixels(0, 875, 100, 50); ok {
		t.Fatal("zero pixel width accepted")
	}
	if _, ok := aspectFromPixels(1000, 3000, 100, 50); ok {
		t.Fatal("out-of-range aspect accepted")
	}
}

func TestNoAutoAspectUsesConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := dir + "/rbxcb-tui"
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.WriteFile(cfgDir+"/config.json", []byte(`{"cell_aspect":1.9,"auto_aspect":false}`), 0o644)
	m := New(Options{NoAutoAspect: true})
	if math.Abs(m.cellAspect-1.9) > 0.01 {
		t.Fatalf("aspect = %.2f, want 1.9", m.cellAspect)
	}
}

func TestExplicitAspectOverridesAuto(t *testing.T) {
	m := New(Options{CellAspect: 2.0})
	if math.Abs(m.cellAspect-2.0) > 0.01 {
		t.Fatalf("aspect = %.2f", m.cellAspect)
	}
}

func TestThemeMenuHitboxesMatchRenderedRows(t *testing.T) {
	sizes := [][2]int{{100, 37}, {121, 41}, {160, 45}}
	for _, sz := range sizes {
		m := New(Options{Facelets: sample, NoAutoAspect: true})
		updated, _ := m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		m = updated.(Model)
		updated, _ = m.Update(press("t"))
		m = updated.(Model)
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		for _, h := range m.hits {
			if h.Kind != HitTheme {
				continue
			}
			if h.Rect.y < 0 || h.Rect.y >= len(lines) {
				t.Fatalf("%dx%d: theme row %d outside view", sz[0], sz[1], h.Puzzle)
			}
			if !strings.Contains(lines[h.Rect.y], m.themes[h.Puzzle].Name) {
				t.Fatalf("%dx%d: row for %s rendered as %q", sz[0], sz[1], m.themes[h.Puzzle].Name, lines[h.Rect.y])
			}
		}
	}
}

func TestThemeMenuClickSelectsRenderedTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := sized(t, New(Options{Facelets: sample, NoAutoAspect: true}))
	for _, idx := range []int{0, 3, 7, len(BuiltinThemes()) - 1} {
		updated, _ := m.Update(press("t"))
		m = updated.(Model)
		m.rebuildHits()
		var hit Hit
		for _, h := range m.hits {
			if h.Kind == HitTheme && h.Puzzle == idx {
				hit = h
			}
		}
		updated, _ = m.handleMouseClick(tea.Mouse{X: hit.Rect.x + 1, Y: hit.Rect.y, Button: tea.MouseLeft})
		m = updated.(Model)
		if m.theme.Name != m.themes[idx].Name {
			t.Fatalf("click row %d selected %s, want %s", idx, m.theme.Name, m.themes[idx].Name)
		}
	}
}

func TestThemeMenuHoverMatchesRenderedRow(t *testing.T) {
	m := sized(t, New(Options{Facelets: sample, NoAutoAspect: true}))
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	m.rebuildHits()
	for _, h := range m.hits {
		if h.Kind != HitTheme {
			continue
		}
		updated, _ = m.Update(tea.MouseMotionMsg(tea.Mouse{X: h.Rect.x + 1, Y: h.Rect.y, Button: tea.MouseNone}))
		m = updated.(Model)
		if m.hover.ID != fmt.Sprintf("theme:%d", h.Puzzle) {
			t.Fatalf("hover %s for row %d", m.hover.ID, h.Puzzle)
		}
	}
}

func TestThemeMenuOutsideClickCloses(t *testing.T) {
	m := sized(t, New(Options{Facelets: sample, NoAutoAspect: true}))
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	updated, _ = m.handleMouseClick(tea.Mouse{X: 0, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.themeMenuVisible {
		t.Fatal("outside click did not close theme menu")
	}
}

func TestThemeMenuScrolledRows(t *testing.T) {
	m := sized(t, New(Options{Facelets: sample, NoAutoAspect: true}))
	updated, _ := m.Update(press("t"))
	m = updated.(Model)
	m.themeScroll = 5
	m.rebuildHits()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for _, h := range m.hits {
		if h.Kind != HitTheme {
			continue
		}
		if !strings.Contains(lines[h.Rect.y], m.themes[h.Puzzle].Name) {
			t.Fatalf("scrolled row %d rendered incorrectly", h.Puzzle)
		}
	}
}

func themeIndex(m Model, name string) int {
	for i, th := range m.themes {
		if th.Name == name {
			return i
		}
	}
	return -1
}

func TestSelectedThemePersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	m := New(Options{Facelets: sample, NoAutoAspect: true})
	idx := themeIndex(m, "nord")
	if idx < 0 {
		t.Fatal("nord theme missing")
	}
	m.applyThemeIndex(idx)
	if m.theme.Name != "nord" {
		t.Fatalf("theme = %s", m.theme.Name)
	}
	m2 := New(Options{Facelets: sample, NoAutoAspect: true})
	if m2.theme.Name != "nord" {
		t.Fatalf("reloaded theme = %s, want nord", m2.theme.Name)
	}
}

func TestThemeSelectionPreservesOtherConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := dir + "/rbxcb-tui"
	_ = os.MkdirAll(cfgDir, 0o755)
	raw := `{"theme":"gruvbox","hover":false,"mouse":false,"cell_aspect":1.9,"auto_aspect":false,"themes":{"mine":{"accent":"#112233"}}}`
	_ = os.WriteFile(cfgDir+"/config.json", []byte(raw), 0o644)
	m := New(Options{NoAutoAspect: true})
	idx := themeIndex(m, "nord")
	m.applyThemeIndex(idx)
	saved, err := os.ReadFile(cfgDir + "/config.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(saved, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "nord" {
		t.Fatalf("theme = %s", cfg.Theme)
	}
	if cfg.Hover == nil || *cfg.Hover {
		t.Fatal("hover preference not preserved")
	}
	if cfg.Mouse == nil || *cfg.Mouse {
		t.Fatal("mouse preference not preserved")
	}
	if math.Abs(cfg.CellAspect-1.9) > 0.01 {
		t.Fatalf("cell_aspect = %.2f", cfg.CellAspect)
	}
	if cfg.AutoAspect == nil || *cfg.AutoAspect {
		t.Fatal("auto_aspect not preserved")
	}
	if _, ok := cfg.Themes["mine"]; !ok {
		t.Fatal("custom theme not preserved")
	}
}

func TestNoConfigWrittenOnStartup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	_ = New(Options{Facelets: sample, NoAutoAspect: true})
	if _, err := os.Stat(dir + "/rbxcb-tui/config.json"); !os.IsNotExist(err) {
		t.Fatalf("config written on startup: %v", err)
	}
}

func TestFlagThemeDoesNotPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	m := New(Options{Facelets: sample, Theme: "nord", NoAutoAspect: true})
	if m.theme.Name != "nord" {
		t.Fatalf("theme = %s", m.theme.Name)
	}
	if _, err := os.Stat(dir + "/rbxcb-tui/config.json"); !os.IsNotExist(err) {
		t.Fatal("flag theme persisted unexpectedly")
	}
}

func TestNoColorDoesNotPersistMono(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	m := New(Options{Facelets: sample, NoColor: true, NoAutoAspect: true})
	if m.theme.Name != "mono" {
		t.Fatalf("theme = %s", m.theme.Name)
	}
	if _, err := os.Stat(dir + "/rbxcb-tui/config.json"); !os.IsNotExist(err) {
		t.Fatal("NO_COLOR persisted mono")
	}
}

func TestSaveConfigFailureIsReported(t *testing.T) {
	dir := t.TempDir()
	blocker := dir + "/blocker"
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", blocker)
	if err := SaveConfig(Config{Theme: "nord"}); err == nil {
		t.Fatal("expected save error")
	}
}

func TestNoPuzzleHidesRightPaneAndPalette(t *testing.T) {
	m := sized(t, New(Options{NoAutoAspect: true}))
	l := m.layout()
	if l.right.rect.w != 0 {
		t.Fatalf("right pane width = %d, want 0", l.right.rect.w)
	}
	if l.palette.rect.w != 0 {
		t.Fatalf("palette width = %d, want 0", l.palette.rect.w)
	}
	if len(l.actions) != 0 || len(l.sections) != 0 {
		t.Fatal("actions/sections exist without a puzzle")
	}
	content := stripANSI(m.View().Content)
	for _, hidden := range []string{"Solver", "Actions", "Playback", "COLOURS"} {
		if strings.Contains(content, hidden) {
			t.Fatalf("%q visible without a puzzle", hidden)
		}
	}
}

func TestNoPuzzleKeepsOutputPanel(t *testing.T) {
	m := sized(t, New(Options{NoAutoAspect: true}))
	content := stripANSI(m.View().Content)
	if !strings.Contains(content, "SOLUTION") {
		t.Fatal("output panel missing")
	}
	if !strings.Contains(content, "Select a puzzle") {
		t.Fatal("empty-state hint missing")
	}
}

func TestSelectingPuzzleShowsFullWorkspace(t *testing.T) {
	m := sized(t, New(Options{NoAutoAspect: true}))
	updated, _ := m.Update(press("enter"))
	m = updated.(Model)
	l := m.layout()
	if l.right.rect.w == 0 || l.palette.rect.w == 0 {
		t.Fatal("right pane or palette missing after selection")
	}
	if len(l.actions) == 0 || len(l.sections) != 3 {
		t.Fatalf("sections=%d actions=%d", len(l.sections), len(l.actions))
	}
	content := stripANSI(m.View().Content)
	for _, visible := range []string{"Solver", "Actions", "Playback", "COLOURS"} {
		if !strings.Contains(content, visible) {
			t.Fatalf("%q missing after selecting a puzzle", visible)
		}
	}
}

func TestTabSkipsHiddenPanesWithoutPuzzle(t *testing.T) {
	m := sized(t, New(Options{NoAutoAspect: true}))
	if m.focus != FocusPuzzles {
		t.Fatalf("focus = %v", m.focus)
	}
	updated, _ := m.Update(press("tab"))
	m = updated.(Model)
	if m.focus != FocusOutput {
		t.Fatalf("tab focus = %v, want output", m.focus)
	}
	updated, _ = m.Update(press("tab"))
	m = updated.(Model)
	if m.focus != FocusPuzzles {
		t.Fatalf("second tab focus = %v, want puzzles", m.focus)
	}
}

func TestNoActionHitboxesWithoutPuzzle(t *testing.T) {
	m := sized(t, New(Options{NoAutoAspect: true}))
	for _, h := range m.hits {
		switch h.Kind {
		case HitAction, HitPalette, HitCell, HitFaceArrow:
			t.Fatalf("unexpected hitbox kind %v without a puzzle", h.Kind)
		}
	}
}

func TestClickWhereRightPaneWouldBeDoesNothing(t *testing.T) {
	m := sized(t, New(Options{NoAutoAspect: true}))
	updated, _ := m.handleMouseClick(tea.Mouse{X: m.width - 5, Y: 10, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.focus == FocusActions {
		t.Fatal("focus moved to hidden actions pane")
	}
	if m.selectedPuzzle >= 0 {
		t.Fatal("puzzle selected by background click")
	}
}
