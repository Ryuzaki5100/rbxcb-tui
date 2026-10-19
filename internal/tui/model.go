package tui

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/solver"
	"github.com/charmbracelet/x/ansi"
)

// Focus identifies the active pane.
type Focus int

const (
	FocusPuzzles Focus = iota
	FocusCube
	FocusActions
	FocusOutput
)

// Options configures the initial model.
type Options struct {
	Demo         bool
	NoMouse      bool
	NoHover      bool
	NoColor      bool
	Theme        string
	CellAspect   float64
	NoAutoAspect bool
	Facelets     string
	Scramble     string
	Stage        solver.Stage
}

type Solution struct {
	Stage   solver.Stage
	Moves   []cube.Move
	States  []cube.Cube
	Step    int
	Playing bool
	Elapsed time.Duration
}

type Model struct {
	keys KeyMap
	opts Options

	width, height  int
	focus          Focus
	selectedPuzzle int
	puzzleCursor   int
	actionCursor   int
	outputScroll   int

	editor           Editor
	showPlayback     bool
	stage            solver.Stage
	mouseOn          bool
	hoverEnabled     bool
	hover            hoverTarget
	helpVisible      bool
	themeMenuVisible bool
	themeCursor      int
	themeScroll      int

	config     Config
	theme      Theme
	styles     Styles
	themes     []Theme
	themeIndex int
	cellAspect float64

	status   string
	errMsg   string
	spinner  int
	solving  bool
	solveGen int
	cancel   context.CancelFunc

	solution   Solution
	hits       []Hit
	pendingCmd tea.Cmd
}

type solveDoneMsg struct {
	gen     int
	stage   solver.Stage
	moves   []cube.Move
	start   cube.Cube
	err     error
	elapsed time.Duration
}

type tickMsg struct {
	gen  int
	kind string
}

const (
	minWidth  = 100
	minHeight = 37
)

func New(opts Options) Model {
	cfg, cfgErr := LoadConfig()
	themes := ThemesWithCustom(cfg)

	name := opts.Theme
	if name == "" {
		name = os.Getenv("RBXCB_THEME")
	}
	if name == "" {
		name = cfg.Theme
	}
	noColor := opts.NoColor || os.Getenv("NO_COLOR") != ""
	if noColor {
		name = "mono"
	}
	theme, themeOK := findTheme(themes, name)
	themeWarn := ""
	if !themeOK {
		if name != "" {
			themeWarn = "unknown theme " + name
		}
		theme = defaultTheme()
	}
	idx := 0
	for i, t := range themes {
		if strings.EqualFold(t.Name, theme.Name) {
			idx = i
			break
		}
	}

	aspect := opts.CellAspect
	aspectSource := "manual"
	if aspect == 0 && !opts.NoAutoAspect && (cfg.AutoAspect == nil || *cfg.AutoAspect) {
		if detected, ok := detectCellAspect(); ok {
			aspect = detected
			aspectSource = "auto"
		}
	}
	if aspect == 0 {
		aspect = cfg.CellAspect
		aspectSource = "config"
	}
	if aspect < 1.2 || aspect > 2.6 {
		aspect = 1.75
		aspectSource = "default"
	}

	hoverEnabled := !opts.NoHover
	if cfg.Hover != nil {
		hoverEnabled = *cfg.Hover && !opts.NoHover
	}
	mouseOn := !opts.NoMouse
	if cfg.Mouse != nil && !opts.NoMouse {
		mouseOn = *cfg.Mouse
	}

	m := Model{
		keys:           DefaultKeyMap(),
		opts:           opts,
		width:          120,
		height:         40,
		focus:          FocusPuzzles,
		selectedPuzzle: -1,
		editor:         NewEditor(),
		stage:          opts.Stage,
		mouseOn:        mouseOn,
		hoverEnabled:   hoverEnabled,
		theme:          theme,
		styles:         theme.styles(),
		config:         cfg,
		themes:         themes,
		themeIndex:     idx,
		cellAspect:     aspect,
	}
	switch {
	case cfgErr != nil:
		m.errMsg = cfgErr.Error()
	case themeWarn != "":
		m.errMsg = themeWarn
	}
	if noColor {
		if t, ok := findTheme(themes, "mono"); ok {
			m.theme = t
			m.styles = t.styles()
		}
	}
	if m.stage < solver.Stage2 || m.stage > solver.Stage4 {
		m.stage = solver.Stage4
	}
	loaded := false
	if opts.Facelets != "" {
		if err := m.editor.LoadFacelets(opts.Facelets); err != nil {
			m.errMsg = err.Error()
		} else {
			loaded = true
		}
	}
	if opts.Scramble != "" {
		moves, err := cube.ParseNotation(opts.Scramble)
		if err != nil {
			m.errMsg = err.Error()
		} else if err := m.editor.LoadScramble(moves); err != nil {
			m.errMsg = err.Error()
		} else {
			loaded = true
		}
	}
	if opts.Demo && !loaded {
		_ = m.editor.LoadFacelets("ybgrbrrbbrgbooygorwyyrwyggboboorgwbryobyygyrwwwowgwowg")
		loaded = true
	}
	if loaded {
		m.selectedPuzzle = 0
		m.focus = FocusCube
	}
	if aspectSource != "default" && m.status == "" && m.errMsg == "" {
		m.status = fmt.Sprintf("aspect %.2f (%s)", aspect, aspectSource)
	}
	m.rebuildHits()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rebuildHits()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg.Mouse())
	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg.Mouse())
	case tea.MouseWheelMsg:
		return m.handleWheel(msg.Mouse())
	case solveDoneMsg:
		return m.handleSolveDone(msg)
	case tickMsg:
		return m.handleTick(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "ctrl+c" {
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	}
	if m.helpVisible {
		if s == "?" || s == "esc" || s == "q" {
			m.helpVisible = false
		}
		return m, nil
	}
	if m.themeMenuVisible {
		m.handleThemeMenuKey(msg)
		return m, nil
	}
	if key.Matches(msg, m.keys.Quit) {
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Theme) {
		m.openThemeMenu()
		return m, nil
	}
	if key.Matches(msg, m.keys.Aspect) {
		m.cycleAspect()
		return m, nil
	}
	if key.Matches(msg, m.keys.Mouse) {
		m.mouseOn = !m.mouseOn
		m.status = "mouse " + onOff(m.mouseOn)
		m.rebuildHits()
		return m, nil
	}
	if key.Matches(msg, m.keys.Help) {
		m.helpVisible = true
		return m, nil
	}
	if key.Matches(msg, m.keys.NextPane) {
		m.cycleFocus(1)
		m.rebuildHits()
		return m, nil
	}
	if key.Matches(msg, m.keys.PrevPane) {
		m.cycleFocus(-1)
		m.rebuildHits()
		return m, nil
	}

	if m.selectedPuzzle >= 0 {
		if s == "v" {
			m.togglePlaybackView()
			return m, nil
		}
		if s == "e" {
			m.showPlayback = false
			m.solution.Playing = false
			m.focus = FocusCube
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keys.Stage2):
			m.stage = solver.Stage2
		case key.Matches(msg, m.keys.Stage3):
			m.stage = solver.Stage3
		case key.Matches(msg, m.keys.Stage4):
			m.stage = solver.Stage4
		case key.Matches(msg, m.keys.Solve):
			m.startSolve()
		case key.Matches(msg, m.keys.Scramble):
			m.loadRandomScramble()
		}
		if s == "c" && m.focus != FocusCube {
			m.clearCube()
		}
	}
	if s == "esc" {
		if m.solving && m.cancel != nil {
			m.cancel()
			m.status = "cancelling..."
			return m, nil
		}
		m.focus = FocusPuzzles
		m.rebuildHits()
		return m, nil
	}

	switch m.focus {
	case FocusPuzzles:
		m.handlePuzzleKey(msg)
	case FocusCube:
		if m.selectedPuzzle >= 0 {
			m.handleEditorKey(msg)
		}
	case FocusActions:
		m.handleActionKey(msg)
	case FocusOutput:
		m.handleOutputKey(msg)
	}
	m.rebuildHits()
	cmd := m.pendingCmd
	m.pendingCmd = nil
	return m, cmd
}

func (m *Model) handlePuzzleKey(msg tea.KeyPressMsg) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.puzzleCursor = clamp(m.puzzleCursor-1, 0, len(pickerOptions)-1)
	case key.Matches(msg, m.keys.Down):
		m.puzzleCursor = clamp(m.puzzleCursor+1, 0, len(pickerOptions)-1)
	case msg.String() == "enter":
		m.selectPuzzle(m.puzzleCursor)
	}
}

func (m *Model) handleEditorKey(msg tea.KeyPressMsg) {
	m.showPlayback = false
	m.solution.Playing = false
	s := msg.String()
	switch {
	case key.Matches(msg, m.keys.Up):
		m.editor.MoveCell(-1, 0)
	case key.Matches(msg, m.keys.Down):
		m.editor.MoveCell(1, 0)
	case key.Matches(msg, m.keys.Left):
		m.editor.MoveCell(0, -1)
	case key.Matches(msg, m.keys.Right):
		m.editor.MoveCell(0, 1)
	case key.Matches(msg, m.keys.FaceUp):
		m.editor.MoveFace(0)
	case key.Matches(msg, m.keys.FaceDown):
		m.editor.MoveFace(1)
	case key.Matches(msg, m.keys.FaceLeft):
		m.editor.MoveFace(2)
	case key.Matches(msg, m.keys.FaceRight):
		m.editor.MoveFace(3)
	case key.Matches(msg, m.keys.NextFace):
		m.editor.MoveFace(3)
	case key.Matches(msg, m.keys.PrevFace):
		m.editor.MoveFace(2)
	case key.Matches(msg, m.keys.Paint):
		m.editor.Paint(m.editor.Current)
	case key.Matches(msg, m.keys.Clear):
		m.editor.Clear()
	case s == "w":
		m.editor.Paint(White)
	case s == "y":
		m.editor.Paint(Yellow)
	case s == "r":
		m.editor.Paint(Red)
	case s == "o":
		m.editor.Paint(Orange)
	case s == "b":
		m.editor.Paint(Blue)
	case s == "g":
		m.editor.Paint(Green)
	}
}

func (m *Model) handleActionKey(msg tea.KeyPressMsg) {
	actions := m.layout().actions
	if len(actions) == 0 {
		return
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		m.actionCursor = clamp(m.actionCursor-1, 0, len(actions)-1)
	case key.Matches(msg, m.keys.Down):
		m.actionCursor = clamp(m.actionCursor+1, 0, len(actions)-1)
	case msg.String() == "enter":
		m.actionCursor = clamp(m.actionCursor, 0, len(actions)-1)
		if actions[m.actionCursor].enabled {
			m.activateAction(actions[m.actionCursor].kind)
		}
	}
}

func (m *Model) handleOutputKey(msg tea.KeyPressMsg) {
	switch {
	case key.Matches(msg, m.keys.Down):
		m.scrollOutput(1)
	case key.Matches(msg, m.keys.Up):
		m.scrollOutput(-1)
	case key.Matches(msg, m.keys.PageDown):
		m.scrollOutput(m.layout().out.perRow)
	case key.Matches(msg, m.keys.PageUp):
		m.scrollOutput(-m.layout().out.perRow)
	case key.Matches(msg, m.keys.PlayPause):
		m.togglePlayback()
	case key.Matches(msg, m.keys.StepBack):
		m.stepSolution(-1)
	case key.Matches(msg, m.keys.StepForward):
		m.stepSolution(1)
	case key.Matches(msg, m.keys.First):
		m.solution.Step = 0
		m.solution.Playing = false
		m.ensureStepVisible()
	case key.Matches(msg, m.keys.Last):
		m.solution.Step = len(m.solution.Moves)
		m.solution.Playing = false
		m.ensureStepVisible()
	}
}

func (m *Model) selectPuzzle(index int) {
	if index < 0 || index >= len(pickerOptions) {
		return
	}
	if !pickerOptions[index].enabled {
		m.status = pickerOptions[index].label + " is not supported yet"
		return
	}
	if m.selectedPuzzle != index {
		m.selectedPuzzle = index
	}
	m.focus = FocusCube
	m.errMsg = ""
	m.status = pickerOptions[index].label + " selected"
	m.rebuildHits()
}

func (m *Model) openThemeMenu() {
	if len(m.themes) == 0 {
		return
	}
	m.themeMenuVisible = true
	m.themeCursor = m.themeIndex
	m.themeScroll = 0
	m.ensureThemeVisible()
	m.rebuildHits()
}

func (m *Model) handleThemeMenuKey(msg tea.KeyPressMsg) {
	switch {
	case msg.String() == "esc" || msg.String() == "q" || key.Matches(msg, m.keys.Theme):
		m.themeMenuVisible = false
	case key.Matches(msg, m.keys.Up):
		m.themeCursor = clamp(m.themeCursor-1, 0, len(m.themes)-1)
		m.ensureThemeVisible()
	case key.Matches(msg, m.keys.Down):
		m.themeCursor = clamp(m.themeCursor+1, 0, len(m.themes)-1)
		m.ensureThemeVisible()
	case msg.String() == "enter":
		m.applyThemeIndex(m.themeCursor)
	}
	m.rebuildHits()
}

func (m *Model) ensureThemeVisible() {
	visible := m.themeMenuVisibleRows()
	if m.themeCursor < m.themeScroll {
		m.themeScroll = m.themeCursor
	}
	if m.themeCursor >= m.themeScroll+visible {
		m.themeScroll = m.themeCursor - visible + 1
	}
}

func (m *Model) applyThemeIndex(i int) {
	if i < 0 || i >= len(m.themes) {
		return
	}
	m.themeIndex = i
	m.theme = m.themes[i]
	m.styles = m.theme.styles()
	m.themeMenuVisible = false
	m.config.Theme = m.theme.Name
	if err := SaveConfig(m.config); err != nil {
		m.status = "theme: " + m.theme.Name + " (not saved: " + err.Error() + ")"
	} else {
		m.status = "theme: " + m.theme.Name + " (saved)"
	}
	m.rebuildHits()
}

func (m *Model) cycleFocus(dir int) {
	order := []Focus{FocusPuzzles, FocusOutput}
	if m.selectedPuzzle >= 0 {
		order = []Focus{FocusPuzzles, FocusCube, FocusActions, FocusOutput}
	}
	pos := -1
	for i, f := range order {
		if f == m.focus {
			pos = i
			break
		}
	}
	if pos < 0 {
		m.focus = order[0]
		return
	}
	m.focus = order[(pos+dir+len(order))%len(order)]
}

func (m *Model) cycleAspect() {
	values := []float64{1.5, 1.6, 1.7, 1.75, 1.8, 1.9, 2.0}
	next := 1.75
	for i, v := range values {
		if math.Abs(v-m.cellAspect) < 0.01 {
			next = values[(i+1)%len(values)]
			break
		}
	}
	m.cellAspect = next
	m.status = fmt.Sprintf("cell aspect: %.2f", next)
	m.rebuildHits()
}

func (m *Model) cycleTheme(dir int) {
	if len(m.themes) == 0 {
		return
	}
	m.themeIndex = (m.themeIndex + dir + len(m.themes)) % len(m.themes)
	m.theme = m.themes[m.themeIndex]
	m.styles = m.theme.styles()
	m.status = "theme: " + m.theme.Name
	m.rebuildHits()
}

func (m *Model) togglePlaybackView() {
	if len(m.solution.Moves) == 0 {
		return
	}
	m.showPlayback = !m.showPlayback
	if !m.showPlayback {
		m.solution.Playing = false
	}
}

func (m *Model) clearCube() {
	if m.selectedPuzzle < 0 {
		return
	}
	m.editor = NewEditor()
	m.solution = Solution{}
	m.errMsg = ""
	m.status = "cleared cube input"
}

func (m *Model) loadRandomScramble() {
	if m.selectedPuzzle < 0 {
		return
	}
	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0x9e3779b97f4a7c15))
	if err := m.editor.LoadScramble(RandomScramble(rng, 20)); err != nil {
		m.errMsg = err.Error()
	} else {
		m.errMsg = ""
		m.status = "loaded random scramble"
	}
}

func (m *Model) activateAction(kind actionKind) {
	switch kind {
	case actionStage2:
		m.stage = solver.Stage2
	case actionStage3:
		m.stage = solver.Stage3
	case actionStage4:
		m.stage = solver.Stage4
	case actionSolve:
		if m.solving {
			if m.cancel != nil {
				m.cancel()
			}
		} else {
			m.startSolve()
		}
	case actionScramble:
		m.loadRandomScramble()
	case actionClear:
		m.clearCube()
	case actionMouse:
		m.mouseOn = !m.mouseOn
		m.status = "mouse " + onOff(m.mouseOn)
	case actionTheme:
		m.openThemeMenu()
	case actionView:
		m.togglePlaybackView()
	case actionHelp:
		m.helpVisible = true
	case actionPlay:
		m.togglePlayback()
	case actionStepBack:
		m.stepSolution(-1)
	case actionStepForward:
		m.stepSolution(1)
	case actionFirst:
		m.solution.Step = 0
		m.solution.Playing = false
		m.ensureStepVisible()
	case actionLast:
		m.solution.Step = len(m.solution.Moves)
		m.solution.Playing = false
		m.ensureStepVisible()
	}
}

func (m *Model) startSolve() {
	if m.selectedPuzzle < 0 {
		return
	}
	c, err := m.editor.Validate()
	if err != nil {
		m.errMsg = err.Error()
		m.status = ""
		m.focus = FocusOutput
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	m.cancel = cancel
	m.solveGen++
	m.solving = true
	m.showPlayback = false
	m.errMsg = ""
	m.status = fmt.Sprintf("solving with %s...", m.stage)
	m.focus = FocusOutput
	m.outputScroll = 0
	gen := m.solveGen
	stage := m.stage
	start := time.Now()
	cmd := func() tea.Msg {
		moves, err := solver.Solve(ctx, c, stage, nil)
		return solveDoneMsg{gen: gen, stage: stage, moves: moves, start: c, err: err, elapsed: time.Since(start)}
	}
	m.pendingCmd = tea.Batch(cmd, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{gen: gen, kind: "spin"}
	}))
}

func (m Model) handleSolveDone(msg solveDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.solveGen {
		return m, nil
	}
	m.solving = false
	m.cancel = nil
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = ""
		m.focus = FocusOutput
		m.rebuildHits()
		return m, nil
	}
	states := make([]cube.Cube, len(msg.moves)+1)
	states[0] = msg.start
	for i, move := range msg.moves {
		states[i+1] = states[i].ApplyMove(move)
	}
	m.solution = Solution{Stage: msg.stage, Moves: msg.moves, States: states, Elapsed: msg.elapsed}
	m.showPlayback = true
	m.errMsg = ""
	m.status = fmt.Sprintf("%s solved in %d moves (%s)", msg.stage, len(msg.moves), msg.elapsed.Round(time.Millisecond))
	m.focus = FocusOutput
	m.outputScroll = 0
	m.ensureStepVisible()
	m.rebuildHits()
	return m, nil
}

func (m Model) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.solveGen {
		return m, nil
	}
	if m.solving {
		m.spinner++
		return m, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
			return tickMsg{gen: msg.gen, kind: "spin"}
		})
	}
	if m.solution.Playing {
		if m.solution.Step >= len(m.solution.Moves) {
			m.solution.Playing = false
			return m, nil
		}
		m.solution.Step++
		m.ensureStepVisible()
		return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
			return tickMsg{gen: msg.gen, kind: "play"}
		})
	}
	return m, nil
}

func (m *Model) togglePlayback() {
	if len(m.solution.Moves) == 0 {
		return
	}
	m.solution.Playing = !m.solution.Playing
	if m.solution.Playing {
		m.pendingCmd = tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
			return tickMsg{gen: m.solveGen, kind: "play"}
		})
	}
}

func (m *Model) stepSolution(delta int) {
	if len(m.solution.Moves) == 0 {
		return
	}
	m.solution.Step = clamp(m.solution.Step+delta, 0, len(m.solution.Moves))
	m.solution.Playing = false
	m.ensureStepVisible()
}

func (m *Model) scrollOutput(delta int) {
	l := m.layout()
	total := len(m.solution.Moves)
	if total == 0 {
		m.outputScroll = 0
		return
	}
	visibleRows := l.out.visible
	maxScroll := maxInt(0, (total-1)/l.out.perRow-visibleRows+1)
	m.outputScroll = clamp(m.outputScroll+delta, 0, maxScroll)
}

func (m *Model) ensureStepVisible() {
	if len(m.solution.Moves) == 0 {
		m.outputScroll = 0
		return
	}
	l := m.layout()
	idx := m.solution.Step - 1
	if idx < 0 {
		idx = 0
	}
	row := idx / l.out.perRow
	if row < m.outputScroll {
		m.outputScroll = row
	}
	if row >= m.outputScroll+l.out.visible {
		m.outputScroll = row - l.out.visible + 1
	}
	maxScroll := maxInt(0, (len(m.solution.Moves)-1)/l.out.perRow-l.out.visible+1)
	m.outputScroll = clamp(m.outputScroll, 0, maxScroll)
}

func (m Model) View() tea.View {
	width, height := m.width, m.height
	if width <= 0 || height <= 0 {
		width, height = 120, 40
	}
	if width < minWidth || height < minHeight {
		v := tea.NewView(lipgloss.NewStyle().Width(width).Height(height).Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("terminal too small\nneed at least %dx%d (have %dx%d)", minWidth, minHeight, width, height)))
		v.AltScreen = true
		return v
	}
	if m.helpVisible {
		v := tea.NewView(m.renderHelp(width, height))
		v.AltScreen = true
		return v
	}
	if m.themeMenuVisible {
		v := tea.NewView(m.renderThemeMenu(width, height))
		v.AltScreen = true
		if m.mouseOn {
			v.MouseMode = tea.MouseModeAllMotion
		}
		return v
	}

	l := m.layout()
	left := m.renderPuzzlePane(l)
	midTop := lipgloss.JoinHorizontal(lipgloss.Top, m.renderCubePane(l), m.renderPalettePane(l))
	middle := lipgloss.JoinVertical(lipgloss.Left, midTop, m.renderOutputPane(l))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, middle)
	if l.right.rect.w > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, middle, m.renderActionPane(l))
	}
	content := m.renderHeader(width) + "\n" + body + "\n" + m.renderFooter(width)

	v := tea.NewView(lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(content))
	v.AltScreen = true
	if m.mouseOn {
		if m.hoverEnabled {
			v.MouseMode = tea.MouseModeAllMotion
		} else {
			v.MouseMode = tea.MouseModeCellMotion
		}
	}
	return v
}

func (m Model) renderHeader(width int) string {
	left := m.styles.Title.Render("rbxcb-tui")
	var right string
	if m.selectedPuzzle < 0 {
		right = "no puzzle selected"
	} else {
		face := string(faceOrder[m.editor.Face])
		right = fmt.Sprintf("%s  ·  Face %s (%d/6)  ·  center %s  ·  %d/54  ·  %s  ·  mouse %s",
			pickerOptions[m.selectedPuzzle].label, face, m.editor.Face+1,
			m.editor.Faces[m.editor.Face][4].Name(), m.editor.Filled(), m.stage, onOff(m.mouseOn))
		right += "  ·  " + m.theme.Name
	}
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + m.styles.Subtle.Render(right)
}

func (m Model) renderFooter(width int) string {
	status := m.status
	if m.errMsg != "" {
		status = m.styles.Error.Render(m.errMsg)
	}
	if status == "" {
		status = m.styles.Subtle.Render("ready")
	}
	return status + "\n" + m.styles.Subtle.Render(m.footerHelp())
}

func (m Model) footerHelp() string {
	switch m.focus {
	case FocusPuzzles:
		return "j/k or ↑/↓ select puzzle   enter open   tab focus   ? help   q quit"
	case FocusCube:
		return "arrows cell   ctrl+arrows face   w/y/r/o/b/g paint   x clear   s solve   n scramble   tab focus"
	case FocusActions:
		return "j/k or ↑/↓ action   enter activate   tab focus   ? help   q quit"
	case FocusOutput:
		return "j/k scroll   space play/pause   h/l step   g/G first/last   tab focus   q quit"
	}
	return "tab focus   ? help   q quit"
}

func (m Model) renderPane(r rect, focused bool, lines []string) string {
	need := r.h - 2
	if need < 1 {
		need = 1
	}
	for len(lines) < need {
		lines = append(lines, "")
	}
	if len(lines) > need {
		lines = lines[:need]
	}
	style := m.styles.Pane(focused).Width(r.w).Height(r.h)
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) paneTitle(text string, focused bool) string {
	if m.styles.Mono && focused {
		return m.styles.Title.Render("▸ " + text)
	}
	return m.styles.Title.Render(text)
}

func (m Model) renderPuzzlePane(l appLayout) string {
	lines := make([]string, l.left.inner.h)
	lines[0] = m.paneTitle("PUZZLES", m.focus == FocusPuzzles)
	for _, row := range l.puzzles {
		box := row.rect
		rel := box.y - l.left.inner.y
		if rel < 0 || rel+2 >= len(lines) {
			continue
		}
		opt := pickerOptions[row.index]
		selected := m.selectedPuzzle == row.index
		hovered := m.hoverEnabled && m.hover.ID == fmt.Sprintf("puzzle:%d", row.index) && opt.enabled
		border := m.styles.Button
		text := m.styles.Move
		status := "○"
		if selected {
			status = "●"
		}
		if !opt.enabled {
			border = m.styles.ButtonMuted
			text = m.styles.ButtonMuted
			status = "×"
		}
		if hovered {
			border = m.styles.ButtonHover
			text = m.styles.ButtonHover
		}
		marker := " "
		switch {
		case selected:
			marker = "▸"
		case m.focus == FocusPuzzles && m.puzzleCursor == row.index:
			marker = ">"
		}
		label := fmt.Sprintf("%s %-7s %s", marker, opt.label, status)
		contentW := box.w - 4
		content := text.Render(padRight(label, contentW))
		lines[rel] = border.Render("╭" + strings.Repeat("─", box.w-2) + "╮")
		lines[rel+1] = border.Render("│") + " " + content + " " + border.Render("│")
		lines[rel+2] = border.Render("╰" + strings.Repeat("─", box.w-2) + "╯")
	}
	hintY := l.left.inner.h - 1
	if hintY > 12 {
		lines[hintY] = m.styles.Subtle.Render("enter to open")
	}
	return m.renderPane(l.left.rect, m.focus == FocusPuzzles, lines)
}

func (m Model) displayFaces() ([6][9]Color, bool) {
	if m.showPlayback && len(m.solution.States) > 0 {
		step := clamp(m.solution.Step, 0, len(m.solution.States)-1)
		palette := cube.StandardPalette()
		if centers, ok := m.editor.Centers(); ok {
			for i := 0; i < 6; i++ {
				palette[i] = centers[i].Letter()
			}
		}
		rendered, err := cube.Render(m.solution.States[step], palette)
		if err == nil {
			var faces [6][9]Color
			for i := 0; i < 54; i++ {
				c, ok := colorFromLetter(rendered[i])
				if !ok {
					return m.editor.Faces, false
				}
				faces[i/9][i%9] = c
			}
			return faces, true
		}
	}
	return m.editor.Faces, false
}

func orientedColor(faces [6][9]Color, front, up, display int) Color {
	if idx := orientationIndex[front][up][display]; idx >= 0 {
		return faces[front][idx]
	}
	return faces[front][display]
}

func (m Model) renderCubePane(l appLayout) string {
	lines := make([]string, l.cube.inner.h)
	if m.selectedPuzzle < 0 {
		msg := "Select a puzzle from the left pane"
		lines[l.cube.inner.h/2] = lipgloss.PlaceHorizontal(l.cube.inner.w, lipgloss.Center, m.styles.Subtle.Render(msg))
		return m.renderPane(l.cube.rect, m.focus == FocusCube, lines)
	}
	g := l.cubeGeom
	e := &m.editor
	faces, playback := m.displayFaces()
	put := func(absY int, s string) {
		rel := absY - g.inner.y
		if rel >= 0 && rel < len(lines) {
			lines[rel] = s
		}
	}
	centers, centersOK := e.Centers()
	centerName := orientedColor(faces, e.Face, e.Up, 4).Name()
	title := m.styles.Title.Render(pickerOptions[m.selectedPuzzle].label) +
		m.styles.Subtle.Render(fmt.Sprintf("  ·  Face %s (%d/6)  ·  center %s", string(faceOrder[e.Face]), e.Face+1, centerName))
	if m.styles.Mono && m.focus == FocusCube {
		title = m.styles.Title.Render("▸ ") + title
	}
	if playback {
		badge := fmt.Sprintf("  ·  PLAYBACK %d/%d", m.solution.Step, len(m.solution.Moves))
		if m.solution.Step >= len(m.solution.Moves) {
			badge = "  ·  SOLVED"
		}
		title += m.styles.Success.Render(badge)
	}
	title = ansi.Truncate(title, g.inner.w, "…")
	put(g.titleY, lipgloss.PlaceHorizontal(g.inner.w, lipgloss.Center, title))

	put(g.upY, spaces(g.centerX-g.inner.x)+m.arrowStyle("arrow:up").Render("↑"))

	for r := 0; r < 3; r++ {
		cells := make([][4]string, 3)
		for c := 0; c < 3; c++ {
			idx := r*3 + c
			hovered := m.hoverEnabled && m.hover.ID == fmt.Sprintf("cell:%d:%d", r, c)
			cells[c] = m.cellLines(orientedColor(faces, e.Face, e.Up, idx), !playback && idx == e.Cell, hovered, g.cellW)
		}
		for lineIdx := 0; lineIdx < g.cellH; lineIdx++ {
			parts := make([]string, 3)
			for c := 0; c < 3; c++ {
				parts[c] = cells[c][lineIdx]
			}
			row := strings.Join(parts, spaces(g.gapX))
			prefix := spaces(g.cubeX - g.inner.x)
			suffix := ""
			if r == 1 && lineIdx == 1 {
				prefix = spaces(g.leftArrowX-g.inner.x) + m.arrowStyle("arrow:left").Render("←") +
					spaces(g.cubeX-(g.leftArrowX+1))
				suffix = spaces(g.rightArrowX-(g.cubeX+g.cubeW)) + m.arrowStyle("arrow:right").Render("→")
			}
			put(g.cubeY+r*(g.cellH+g.rowGap)+lineIdx, prefix+row+suffix)
		}
	}
	put(g.downY, spaces(g.centerX-g.inner.x)+m.arrowStyle("arrow:down").Render("↓"))

	info := fmt.Sprintf("Current %s   Filled %d/54", e.Current.Name(), e.Filled())
	if playback {
		info = fmt.Sprintf("Playback %d/%d   input preserved", m.solution.Step, len(m.solution.Moves))
	}
	put(g.infoY, lipgloss.PlaceHorizontal(g.inner.w, lipgloss.Center, info))
	if centersOK {
		centersLine := fmt.Sprintf("Centers  U:%c L:%c F:%c R:%c B:%c D:%c",
			centers[0].Letter(), centers[1].Letter(), centers[2].Letter(),
			centers[3].Letter(), centers[4].Letter(), centers[5].Letter())
		put(g.centersY, lipgloss.PlaceHorizontal(g.inner.w, lipgloss.Center, centersLine))
	} else {
		put(g.centersY, lipgloss.PlaceHorizontal(g.inner.w, lipgloss.Center, m.styles.Subtle.Render("Centers: set six different colors")))
	}
	return m.renderPane(l.cube.rect, m.focus == FocusCube, lines)
}

func (m Model) arrowStyle(id string) lipgloss.Style {
	if m.hoverEnabled && m.hover.ID == id {
		return m.styles.ButtonHover
	}
	return m.styles.Selected
}

func (m Model) cellLines(color Color, selected, hovered bool, cellW int) [4]string {
	interior := cellW - 2
	if interior < 1 {
		interior = 1
	}
	fill := m.stickerFill(color, interior)
	if selected {
		border := m.styles.CellBorder
		if hovered {
			border = m.styles.CellHover
		}
		return [4]string{
			border.Render("┏" + strings.Repeat("━", interior) + "┓"),
			border.Render("┃") + fill + border.Render("┃"),
			border.Render("┃") + fill + border.Render("┃"),
			border.Render("┗" + strings.Repeat("━", interior) + "┛"),
		}
	}
	if hovered {
		border := m.styles.CellHover
		return [4]string{
			border.Render("┌" + strings.Repeat("─", interior) + "┐"),
			border.Render("│") + fill + border.Render("│"),
			border.Render("│") + fill + border.Render("│"),
			border.Render("└" + strings.Repeat("─", interior) + "┘"),
		}
	}
	full := m.stickerFill(color, cellW)
	return [4]string{full, full, full, full}
}

func (m Model) stickerFill(color Color, width int) string {
	if width < 1 {
		width = 1
	}
	if color == Empty {
		return m.styles.ButtonMuted.Render(strings.Repeat("░", width))
	}
	if m.styles.Mono {
		return strings.Repeat(monoPatterns[color], width)
	}
	return m.styles.Sticker(int(color)).Render(strings.Repeat("█", width))
}

func (m Model) renderPalettePane(l appLayout) string {
	if l.palette.rect.w <= 0 {
		return ""
	}
	lines := make([]string, l.palette.inner.h)
	lines[0] = m.paneTitle("COLOURS", false)
	pg := l.paletteGeom
	e := &m.editor
	put := func(absY int, s string) {
		rel := absY - l.palette.inner.y
		if rel >= 0 && rel < len(lines) {
			lines[rel] = s
		}
	}
	for row := 0; row < 3; row++ {
		var seg [2][5]string
		for col := 0; col < 2; col++ {
			i := row*2 + col
			border := m.styles.SwatchBorder
			labelStyle := m.styles.Subtle
			if Color(i) == e.Current {
				border = m.styles.Selected
				labelStyle = m.styles.Selected
			}
			if m.hoverEnabled && m.hover.ID == fmt.Sprintf("palette:%d", i) {
				border = m.styles.ButtonHover
				labelStyle = m.styles.ButtonHover
			}
			swatch := strings.Repeat(monoPatterns[i], pg.swatchW)
			if !m.styles.Mono {
				swatch = m.styles.Sticker(i).Render(strings.Repeat("█", pg.swatchW))
			}
			seg[col][0] = padRight(border.Render("╭"+strings.Repeat("─", pg.swatchW)+"╮"), pg.colW)
			seg[col][1] = border.Render("│") + swatch + border.Render("│") + spaces(maxInt(0, pg.colW-pg.chipW))
			seg[col][2] = seg[col][1]
			seg[col][3] = padRight(border.Render("╰"+strings.Repeat("─", pg.swatchW)+"╯"), pg.colW)
			seg[col][4] = padCenter(labelStyle.Render(colorNames[i]), pg.colW)
		}
		y := pg.firstY + row*pg.stride
		for k := 0; k < 5; k++ {
			put(y+k, seg[0][k]+spaces(pg.colGap)+seg[1][k])
		}
	}
	return m.renderPane(l.palette.rect, false, lines)
}

func (m Model) renderOutputPane(l appLayout) string {
	lines := make([]string, l.output.inner.h)
	o := l.out
	put := func(absY int, s string) {
		rel := absY - o.inner.y
		if rel >= 0 && rel < len(lines) {
			lines[rel] = s
		}
	}
	put(o.titleY, m.paneTitle("SOLUTION", m.focus == FocusOutput)+"  "+m.styles.Subtle.Render(m.outputSummary()))
	switch {
	case m.solving:
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		put(o.statusY, frames[m.spinner%len(frames)]+" "+m.status)
		put(o.statusY+1, m.styles.Subtle.Render("esc cancel"))
	case m.errMsg != "":
		put(o.statusY, m.styles.Error.Render(m.errMsg))
	case len(m.solution.Moves) == 0:
		put(o.statusY, "No solution yet.")
		hint := "Paint the cube and press s."
		if m.selectedPuzzle < 0 {
			hint = "Select a puzzle from the left pane."
		}
		put(o.statusY+1, m.styles.Subtle.Render(hint))
	default:
		put(o.statusY, m.outputStatusLine())
		for row := 0; row < o.visible; row++ {
			var b strings.Builder
			for col := 0; col < o.perRow; col++ {
				idx := m.outputScroll + row*o.perRow + col
				if idx >= len(m.solution.Moves) {
					break
				}
				label := fmt.Sprintf("%2d %-3s", idx+1, m.solution.Moves[idx].String())
				switch {
				case idx+1 == m.solution.Step:
					label = m.styles.CurrentMove.Render(label)
				case m.hoverEnabled && m.hover.ID == fmt.Sprintf("move:%d", idx+1):
					label = m.styles.Selected.Render(label)
				default:
					label = m.styles.Move.Render(label)
				}
				b.WriteString(label)
				if col < o.perRow-1 {
					b.WriteString(" ")
				}
			}
			put(o.movesY+row, b.String())
		}
	}
	return m.renderPane(l.output.rect, m.focus == FocusOutput, lines)
}

func (m Model) outputSummary() string {
	if m.solving {
		return m.stage.String()
	}
	if len(m.solution.Moves) == 0 {
		return ""
	}
	return fmt.Sprintf("%s · %d moves · %s", m.solution.Stage, len(m.solution.Moves), m.solution.Elapsed.Round(time.Millisecond))
}

func (m Model) outputStatusLine() string {
	state := "paused"
	if m.solution.Playing {
		state = "playing"
	}
	if m.solution.Step == 0 {
		return fmt.Sprintf("%s · start · step 0/%d", state, len(m.solution.Moves))
	}
	return fmt.Sprintf("%s · %s · step %d/%d", state, m.solution.Moves[m.solution.Step-1].String(), m.solution.Step, len(m.solution.Moves))
}

func (m Model) renderActionPane(l appLayout) string {
	if m.selectedPuzzle < 0 || l.right.rect.w <= 0 {
		return ""
	}
	lines := make([]string, l.right.inner.h)
	iy := l.right.inner.y

	for _, sec := range l.sections {
		rel := sec.rect.y - iy
		if rel < 0 || rel+sec.rect.h > len(lines) {
			continue
		}
		top := []rune("╭" + strings.Repeat("─", sec.rect.w-2) + "╮")
		label := []rune(" " + sec.title + " ")
		copy(top[2:], label)
		lines[rel] = m.styles.Subtle.Render(string(top))
		for cl := 1; cl < sec.rect.h-1; cl++ {
			y := sec.rect.y + cl
			type pillRow struct {
				row actionRow
				idx int
			}
			var rows []pillRow
			for i, row := range l.actions {
				if row.rect.y == y {
					rows = append(rows, pillRow{row, i})
				}
			}
			if len(rows) == 0 {
				lines[rel+cl] = m.styles.Subtle.Render("│" + spaces(sec.rect.w-2) + "│")
				continue
			}
			var b strings.Builder
			cursor := 0
			baseX := sec.rect.x + 1
			for _, pr := range rows {
				gap := pr.row.rect.x - baseX - cursor
				if gap > 0 {
					b.WriteString(spaces(gap))
				}
				b.WriteString(m.renderActionPill(pr.row, pr.idx))
				cursor += gap + pr.row.rect.w
			}
			if rem := sec.rect.w - 2 - cursor; rem > 0 {
				b.WriteString(spaces(rem))
			}
			lines[rel+cl] = m.styles.Subtle.Render("│") + b.String() + m.styles.Subtle.Render("│")
		}
		lines[rel+sec.rect.h-1] = m.styles.Subtle.Render("╰" + strings.Repeat("─", sec.rect.w-2) + "╯")
	}
	return m.renderPane(l.right.rect, m.focus == FocusActions, lines)
}

func (m Model) renderActionPill(row actionRow, index int) string {
	label := row.label
	if row.kind == actionSolve {
		switch {
		case m.solving:
			label = "[esc] Cancel"
		case len(m.solution.Moves) > 0:
			label = "[s] Re-solve"
		}
	}
	if row.kind == actionView {
		if m.showPlayback {
			label = "[v] Input"
		} else {
			label = "[v] Playback"
		}
	}
	hovered := m.hoverEnabled && m.hover.ID == "action:"+actionID(row.kind)
	focused := m.focus == FocusActions && m.actionCursor == index
	selectedStage := (row.kind == actionStage2 && m.stage == solver.Stage2) ||
		(row.kind == actionStage3 && m.stage == solver.Stage3) ||
		(row.kind == actionStage4 && m.stage == solver.Stage4)
	style := m.styles.ButtonRow
	switch {
	case !row.enabled:
		style = m.styles.ButtonRowMuted
	case hovered || focused:
		style = m.styles.ButtonRowHover
	case selectedStage:
		style = m.styles.ButtonRowSelected
	}
	width := row.rect.w
	if width < 2 {
		width = 2
	}
	marker := " "
	if focused && row.enabled {
		marker = ">"
	}
	text := marker + padRight(label, width-2) + " "
	return style.Render(text)
}

func actionID(k actionKind) string {
	return fmt.Sprintf("%d", int(k))
}

func padCenter(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return spaces(left) + s + spaces(width-w-left)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + spaces(width-w)
}

func (m Model) renderThemeMenu(width, height int) string {
	l := m.themeMenuLayout()
	lines := make([]string, l.rect.h-2)
	for i := range lines {
		lines[i] = spaces(l.rect.w - 4)
	}
	lines[0] = m.styles.Title.Render("Themes")
	end := m.themeScroll + l.visible
	if end > len(m.themes) {
		end = len(m.themes)
	}
	row := 0
	for i := m.themeScroll; i < end; i++ {
		if row >= l.visible {
			break
		}
		marker := "  "
		if i == m.themeIndex {
			marker = "● "
		}
		if i == m.themeCursor {
			if marker == "  " {
				marker = "> "
			} else {
				marker = "▸ "
			}
		}
		text := marker + m.themes[i].Name
		if i == m.themeIndex {
			text += "  *"
		}
		text = ansi.Truncate(text, l.rect.w-4, "…")
		hovered := m.hoverEnabled && m.hover.ID == fmt.Sprintf("theme:%d", i)
		switch {
		case hovered || i == m.themeCursor:
			text = m.styles.ButtonRowHover.Render(padRight(text, l.rect.w-4))
		case i == m.themeIndex:
			text = m.styles.ButtonRowSelected.Render(padRight(text, l.rect.w-4))
		default:
			text = padRight(text, l.rect.w-4)
		}
		lines[row+1] = text
		row++
	}
	lines[len(lines)-1] = m.styles.Subtle.Render("j/k move   enter   esc close")
	card := m.styles.PaneFocused.Width(l.rect.w).Height(l.rect.h).Render(strings.Join(lines, "\n"))
	out := make([]string, height)
	for i := range out {
		out[i] = spaces(width)
	}
	for i, line := range strings.Split(card, "\n") {
		y := l.rect.y + i
		if y < 0 || y >= height {
			continue
		}
		if x := l.rect.x; x >= 0 && x < width {
			w := lipgloss.Width(line)
			if w > width-x {
				line = ansi.Truncate(line, width-x, "")
				w = lipgloss.Width(line)
			}
			out[y] = spaces(x) + line + spaces(width-x-w)
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) renderHelp(width, height int) string {
	lines := []string{
		m.styles.Title.Render("rbxcb-tui help"),
		"",
		"Panes:",
		"  tab / shift+tab        cycle puzzles → cube → actions → output",
		"",
		"Puzzles pane:",
		"  j/k or ↑/↓             move selection",
		"  enter                  open the selected puzzle",
		"  mouse click            press a puzzle button",
		"",
		"Cube pane:",
		"  arrows or h/j/k/l      move the sticker cursor",
		"  ctrl+arrows or H/J/K/L change face",
		"  tab / shift+tab        cycle faces",
		"  w y r o b g            paint white/yellow/red/orange/blue/green",
		"  space                  repaint current sticker",
		"  x                      clear current sticker",
		"",
		"Actions pane:",
		"  j/k or ↑/↓, enter      select and activate an action",
		"  s / n / c / m          solve, scramble, clear, mouse",
		"  2 / 3 / 4              choose solver stage",
		"",
		"Output pane:",
		"  j/k or ↑/↓             scroll moves",
		"  space                  play/pause",
		"  h / l                  step backward/forward",
		"  g / G                  first/last move",
		"",
		"Facelet order: U L F R B D, each face row-major",
		"",
		m.styles.Subtle.Render("press esc or ? to close"),
	}
	box := m.styles.PaneBlurred.Width(min(width-4, 86)).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
