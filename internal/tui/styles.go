package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles is the precomputed style set for one theme.
type Styles struct {
	Title             lipgloss.Style
	Subtle            lipgloss.Style
	Error             lipgloss.Style
	Success           lipgloss.Style
	Selected          lipgloss.Style
	Move              lipgloss.Style
	CurrentMove       lipgloss.Style
	PaneFocused       lipgloss.Style
	PaneBlurred       lipgloss.Style
	Button            lipgloss.Style
	ButtonHover       lipgloss.Style
	ButtonMuted       lipgloss.Style
	Pill              lipgloss.Style
	PillHover         lipgloss.Style
	PillMuted         lipgloss.Style
	CellBorder        lipgloss.Style
	CellHover         lipgloss.Style
	ButtonRow         lipgloss.Style
	ButtonRowHover    lipgloss.Style
	ButtonRowSelected lipgloss.Style
	ButtonRowMuted    lipgloss.Style
	SwatchBorder      lipgloss.Style
	Stickers          [6]color.Color
	AccentHover       color.Color
	Mono              bool
}

func (t Theme) styles() Styles {
	if t.Mono {
		s := Styles{
			Title:             lipgloss.NewStyle().Bold(true),
			Subtle:            lipgloss.NewStyle(),
			Error:             lipgloss.NewStyle().Bold(true),
			Success:           lipgloss.NewStyle().Bold(true),
			Selected:          lipgloss.NewStyle().Bold(true).Underline(true),
			Move:              lipgloss.NewStyle(),
			CurrentMove:       lipgloss.NewStyle().Bold(true).Underline(true),
			Button:            lipgloss.NewStyle(),
			ButtonHover:       lipgloss.NewStyle().Bold(true).Underline(true),
			ButtonMuted:       lipgloss.NewStyle(),
			Pill:              lipgloss.NewStyle(),
			PillHover:         lipgloss.NewStyle().Bold(true).Underline(true),
			PillMuted:         lipgloss.NewStyle(),
			CellBorder:        lipgloss.NewStyle().Bold(true),
			CellHover:         lipgloss.NewStyle().Bold(true),
			ButtonRow:         lipgloss.NewStyle(),
			ButtonRowHover:    lipgloss.NewStyle().Bold(true).Reverse(true),
			ButtonRowSelected: lipgloss.NewStyle().Bold(true).Reverse(true),
			ButtonRowMuted:    lipgloss.NewStyle(),
			SwatchBorder:      lipgloss.NewStyle(),
			Mono:              true,
		}
		s.PaneFocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		s.PaneBlurred = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		return s
	}
	accent := lipgloss.Color(t.Accent)
	hover := lipgloss.Color(t.AccentHover)
	border := lipgloss.Color(t.Border)
	muted := lipgloss.Color(t.Muted)
	text := lipgloss.Color(t.Text)
	s := Styles{
		Title:       lipgloss.NewStyle().Bold(true).Foreground(accent),
		Subtle:      lipgloss.NewStyle().Foreground(muted),
		Error:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true),
		Success:     lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true),
		Selected:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		Move:        lipgloss.NewStyle().Foreground(text),
		CurrentMove: lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true),
		Button:      lipgloss.NewStyle().Foreground(accent),
		ButtonHover: lipgloss.NewStyle().Bold(true).Foreground(hover),
		ButtonMuted: lipgloss.NewStyle().Foreground(muted),
		Pill:        lipgloss.NewStyle().Foreground(accent),
		PillHover:   lipgloss.NewStyle().Bold(true).Foreground(hover),
		PillMuted:   lipgloss.NewStyle().Foreground(muted),
		CellBorder:  lipgloss.NewStyle().Foreground(accent),
		CellHover:   lipgloss.NewStyle().Bold(true).Foreground(hover),
		ButtonRow: lipgloss.NewStyle().
			Background(lipgloss.Color(t.ButtonBG)).
			Foreground(lipgloss.Color(t.ButtonFG)),
		ButtonRowHover: lipgloss.NewStyle().Bold(true).
			Background(lipgloss.Color(t.ButtonBGHover)).
			Foreground(lipgloss.Color(t.ButtonFGHover)),
		ButtonRowSelected: lipgloss.NewStyle().Bold(true).
			Background(accent).
			Foreground(lipgloss.Color(contrastInk(t.Accent))),
		ButtonRowMuted: lipgloss.NewStyle().
			Background(lipgloss.Color(darkenHex(t.ButtonBG, 0.65))).
			Foreground(muted),
		SwatchBorder: lipgloss.NewStyle().Foreground(border),
		AccentHover:  hover,
		Mono:         t.Mono,
	}
	for i, c := range t.Stickers {
		s.Stickers[i] = lipgloss.Color(c)
	}
	s.PaneFocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1)
	s.PaneBlurred = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1)
	return s
}

func (s Styles) Pane(focused bool) lipgloss.Style {
	if focused {
		return s.PaneFocused
	}
	return s.PaneBlurred
}

func (s Styles) Sticker(i int) lipgloss.Style {
	if s.Mono || s.Stickers[i] == nil {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(s.Stickers[i])
}
