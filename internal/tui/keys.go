package tui

import "charm.land/bubbles/v2/key"

// KeyMap is the single source of truth for all bindings.
type KeyMap struct {
	Up, Down, Left, Right            key.Binding
	PageUp, PageDown                 key.Binding
	FaceUp, FaceDown                 key.Binding
	FaceLeft, FaceRight              key.Binding
	NextFace, PrevFace               key.Binding
	NextPane, PrevPane               key.Binding
	Paint, Clear                     key.Binding
	Solve, Cancel                    key.Binding
	Stage2, Stage3, Stage4           key.Binding
	PlayPause, StepBack              key.Binding
	StepForward, First               key.Binding
	Last, Scramble                   key.Binding
	Theme, Aspect, Mouse, Help, Back key.Binding
	Quit                             key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "move left")),
		Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "move right")),
		PageUp:      key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u", "page up")),
		PageDown:    key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d", "page down")),
		FaceUp:      key.NewBinding(key.WithKeys("ctrl+up", "K"), key.WithHelp("ctrl+↑/K", "face up")),
		FaceDown:    key.NewBinding(key.WithKeys("ctrl+down", "J"), key.WithHelp("ctrl+↓/J", "face down")),
		FaceLeft:    key.NewBinding(key.WithKeys("ctrl+left", "H"), key.WithHelp("ctrl+←/H", "face left")),
		FaceRight:   key.NewBinding(key.WithKeys("ctrl+right", "L"), key.WithHelp("ctrl+→/L", "face right")),
		NextFace:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next face")),
		PrevFace:    key.NewBinding(key.WithKeys("["), key.WithHelp("[", "previous face")),
		NextPane:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		PrevPane:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous pane")),
		Paint:       key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "paint")),
		Clear:       key.NewBinding(key.WithKeys("x", "backspace", "delete"), key.WithHelp("x", "clear")),
		Solve:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "solve")),
		Cancel:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Stage2:      key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "2-stage")),
		Stage3:      key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "3-stage")),
		Stage4:      key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "4-stage")),
		PlayPause:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "play/pause")),
		StepBack:    key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "step back")),
		StepForward: key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "step forward")),
		First:       key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "first move")),
		Last:        key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "last move")),
		Scramble:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "random scramble")),
		Theme:       key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "cycle theme")),
		Aspect:      key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "cell aspect")),
		Mouse:       key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "toggle mouse")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
