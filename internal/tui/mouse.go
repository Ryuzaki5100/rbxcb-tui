package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type HitKind int

const (
	HitPuzzle HitKind = iota
	HitCell
	HitFaceArrow
	HitPalette
	HitAction
	HitMove
	HitTheme
)

type Hit struct {
	Kind      HitKind
	ID        string
	Rect      rect
	Puzzle    int
	Face      int
	Cell      int
	Color     Color
	Action    actionKind
	Move      int
	Direction int
	Enabled   bool
}

type hoverTarget struct {
	Kind HitKind
	ID   string
}

func (m *Model) rebuildHits() {
	m.hits = m.hits[:0]
	if m.themeMenuVisible {
		menu := m.themeMenuLayout()
		for i, r := range menu.rows {
			idx := m.themeScroll + i
			if idx >= len(m.themes) {
				break
			}
			m.hits = append(m.hits, Hit{Kind: HitTheme, ID: fmt.Sprintf("theme:%d", idx), Rect: r, Puzzle: idx})
		}
		return
	}
	l := m.layout()
	for _, row := range l.puzzles {
		m.hits = append(m.hits, Hit{Kind: HitPuzzle, ID: fmt.Sprintf("puzzle:%d", row.index), Rect: row.rect, Puzzle: row.index, Enabled: pickerOptions[row.index].enabled})
	}
	if m.selectedPuzzle >= 0 {
		g := l.cubeGeom
		for r := 0; r < 3; r++ {
			for c := 0; c < 3; c++ {
				cell := r*3 + c
				m.hits = append(m.hits, Hit{Kind: HitCell, ID: fmt.Sprintf("cell:%d:%d", r, c), Rect: g.cellRect(r, c), Face: m.editor.Face, Cell: cell})
			}
		}
		m.hits = append(m.hits,
			Hit{Kind: HitFaceArrow, ID: "arrow:up", Rect: rect{g.centerX, g.upY, 1, 1}, Direction: 0},
			Hit{Kind: HitFaceArrow, ID: "arrow:down", Rect: rect{g.centerX, g.downY, 1, 1}, Direction: 1},
			Hit{Kind: HitFaceArrow, ID: "arrow:left", Rect: rect{g.leftArrowX, g.arrowY, 1, 1}, Direction: 2},
			Hit{Kind: HitFaceArrow, ID: "arrow:right", Rect: rect{g.rightArrowX, g.arrowY, 1, 1}, Direction: 3},
		)
		pg := l.paletteGeom
		for i := 0; i < 6; i++ {
			row, col := i/2, i%2
			m.hits = append(m.hits, Hit{
				Kind:  HitPalette,
				ID:    fmt.Sprintf("palette:%d", i),
				Rect:  rect{pg.pane.inner.x + col*(pg.colW+pg.colGap), pg.firstY + row*pg.stride, pg.colW, 5},
				Color: Color(i),
			})
		}
	}
	for _, row := range l.actions {
		m.hits = append(m.hits, Hit{Kind: HitAction, ID: "action:" + actionID(row.kind), Rect: row.rect, Action: row.kind, Enabled: row.enabled})
	}
	for i := range m.solution.Moves {
		row := i / l.out.perRow
		col := i % l.out.perRow
		if row < m.outputScroll || row >= m.outputScroll+l.out.visible {
			continue
		}
		screenRow := row - m.outputScroll
		m.hits = append(m.hits, Hit{
			Kind: HitMove,
			ID:   fmt.Sprintf("move:%d", i+1),
			Rect: rect{l.out.inner.x + col*7, l.out.movesY + screenRow, 6, 1},
			Move: i + 1,
		})
	}
}

func (m *Model) focusAt(x, y int) {
	l := m.layout()
	switch {
	case l.left.rect.contains(x, y):
		m.focus = FocusPuzzles
	case l.cube.rect.contains(x, y):
		m.focus = FocusCube
	case l.output.rect.contains(x, y):
		m.focus = FocusOutput
	case l.right.rect.contains(x, y):
		m.focus = FocusActions
	}
}

func (m Model) hoverAt(x, y int) hoverTarget {
	for _, h := range m.hits {
		if h.Rect.contains(x, y) {
			return hoverTarget{Kind: h.Kind, ID: h.ID}
		}
	}
	return hoverTarget{}
}

func (m Model) handleMouseClick(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Button != tea.MouseLeft {
		return m, nil
	}
	if m.themeMenuVisible {
		for _, h := range m.hits {
			if h.Kind == HitTheme && h.Rect.contains(mouse.X, mouse.Y) {
				m.themeCursor = h.Puzzle
				m.applyThemeIndex(h.Puzzle)
				return m, nil
			}
		}
		m.themeMenuVisible = false
		m.rebuildHits()
		return m, nil
	}
	m.focusAt(mouse.X, mouse.Y)
	for _, h := range m.hits {
		if !h.Rect.contains(mouse.X, mouse.Y) {
			continue
		}
		m.hover = hoverTarget{Kind: h.Kind, ID: h.ID}
		switch h.Kind {
		case HitPuzzle:
			m.puzzleCursor = h.Puzzle
			if h.Enabled {
				m.selectPuzzle(h.Puzzle)
			} else {
				m.status = pickerOptions[h.Puzzle].label + " is not supported yet"
			}
		case HitCell:
			m.showPlayback = false
			m.solution.Playing = false
			m.editor.Cell = h.Cell
			m.editor.PaintAt(h.Cell, m.editor.Current)
		case HitFaceArrow:
			m.editor.MoveFace(h.Direction)
		case HitPalette:
			m.editor.Current = h.Color
		case HitAction:
			for i, row := range m.layout().actions {
				if row.kind == h.Action && row.rect.y == h.Rect.y {
					m.actionCursor = i
					break
				}
			}
			if h.Enabled {
				m.activateAction(h.Action)
			}
		case HitMove:
			m.showPlayback = true
			m.solution.Step = h.Move
			m.solution.Playing = false
			m.ensureStepVisible()
		case HitTheme:
			m.themeCursor = h.Puzzle
			m.applyThemeIndex(h.Puzzle)
		}
		m.rebuildHits()
		cmd := m.pendingCmd
		m.pendingCmd = nil
		return m, cmd
	}
	m.rebuildHits()
	return m, nil
}

func (m Model) handleMouseMotion(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Button == tea.MouseLeft && m.selectedPuzzle >= 0 {
		for _, h := range m.hits {
			if h.Kind == HitCell && h.Rect.contains(mouse.X, mouse.Y) {
				m.focus = FocusCube
				m.showPlayback = false
				m.solution.Playing = false
				m.hover = hoverTarget{Kind: HitCell, ID: h.ID}
				m.editor.Cell = h.Cell
				m.editor.PaintAt(h.Cell, m.editor.Current)
				return m, nil
			}
		}
	}
	if !m.hoverEnabled {
		return m, nil
	}
	target := m.hoverAt(mouse.X, mouse.Y)
	if target != m.hover {
		m.hover = target
	}
	return m, nil
}

func (m Model) handleWheel(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if m.themeMenuVisible {
		switch mouse.Button {
		case tea.MouseWheelUp:
			m.themeCursor = clamp(m.themeCursor-1, 0, len(m.themes)-1)
		case tea.MouseWheelDown:
			m.themeCursor = clamp(m.themeCursor+1, 0, len(m.themes)-1)
		}
		m.ensureThemeVisible()
		return m, nil
	}
	l := m.layout()
	switch {
	case l.left.rect.contains(mouse.X, mouse.Y):
		m.focus = FocusPuzzles
		switch mouse.Button {
		case tea.MouseWheelUp:
			m.puzzleCursor = clamp(m.puzzleCursor-1, 0, len(pickerOptions)-1)
		case tea.MouseWheelDown:
			m.puzzleCursor = clamp(m.puzzleCursor+1, 0, len(pickerOptions)-1)
		}
	case l.output.rect.contains(mouse.X, mouse.Y):
		m.focus = FocusOutput
		switch mouse.Button {
		case tea.MouseWheelUp:
			m.scrollOutput(-1)
		case tea.MouseWheelDown:
			m.scrollOutput(1)
		}
	}
	return m, nil
}
