package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Theme describes all editable colours used by the TUI.
type Theme struct {
	Name          string    `json:"-"`
	Accent        string    `json:"accent"`
	AccentHover   string    `json:"accent_hover"`
	Border        string    `json:"border"`
	Muted         string    `json:"muted"`
	Text          string    `json:"text"`
	Error         string    `json:"error"`
	Success       string    `json:"success"`
	Stickers      [6]string `json:"stickers"`
	ButtonBG      string    `json:"button_bg"`
	ButtonBGHover string    `json:"button_bg_hover"`
	ButtonFG      string    `json:"button_fg"`
	ButtonFGHover string    `json:"button_fg_hover"`
	Mono          bool      `json:"mono"`
}

// Config is the optional XDG JSON config.
type Config struct {
	Theme      string                 `json:"theme,omitempty"`
	Hover      *bool                  `json:"hover,omitempty"`
	Mouse      *bool                  `json:"mouse,omitempty"`
	CellAspect float64                `json:"cell_aspect,omitempty"`
	AutoAspect *bool                  `json:"auto_aspect,omitempty"`
	Themes     map[string]ThemeConfig `json:"themes,omitempty"`
}

// ThemeConfig is a user-defined theme in config.json.
type ThemeConfig struct {
	Accent        string   `json:"accent"`
	AccentHover   string   `json:"accent_hover"`
	Border        string   `json:"border"`
	Muted         string   `json:"muted"`
	Text          string   `json:"text"`
	Error         string   `json:"error"`
	Success       string   `json:"success"`
	Stickers      []string `json:"stickers"`
	ButtonBG      string   `json:"button_bg"`
	ButtonBGHover string   `json:"button_bg_hover"`
	ButtonFG      string   `json:"button_fg"`
	ButtonFGHover string   `json:"button_fg_hover"`
	Mono          bool     `json:"mono"`
}

func legacyDefaultTheme() Theme {
	return Theme{
		Name:          "default",
		Accent:        "#38bdf8",
		AccentHover:   "#0ea5c6",
		Border:        "#374151",
		Muted:         "#9ca3af",
		Text:          "#e5e7eb",
		Error:         "#f87171",
		Success:       "#4ade80",
		Stickers:      [6]string{"#f2f2f2", "#f2d024", "#d93636", "#ef8b2c", "#2f6fe4", "#2fa866"},
		ButtonBG:      "#1f2937",
		ButtonBGHover: "#111827",
		ButtonFG:      "#e5e7eb",
		ButtonFGHover: "#f9fafb",
	}
}

func gruvboxTheme() Theme {
	return Theme{
		Name:          "gruvbox",
		Accent:        "#fabd2f",
		AccentHover:   "#d79921",
		Border:        "#665c54",
		Muted:         "#a89984",
		Text:          "#ebdbb2",
		Error:         "#fb4934",
		Success:       "#b8bb26",
		Stickers:      [6]string{"#fbf1c7", "#fabd2f", "#cc241d", "#d65d0e", "#458588", "#98971a"},
		ButtonBG:      "#3c3836",
		ButtonBGHover: "#282828",
		ButtonFG:      "#ebdbb2",
		ButtonFGHover: "#fbf1c7",
	}
}

func defaultTheme() Theme { return gruvboxTheme() }

// BuiltinThemes returns the themes shipped with the app.
func BuiltinThemes() []Theme {
	return []Theme{
		gruvboxTheme(),
		legacyDefaultTheme(),
		{
			Name:          "nord",
			Accent:        "#88c0d0",
			AccentHover:   "#5e81ac",
			Border:        "#4c566a",
			Muted:         "#81a1c1",
			Text:          "#eceff4",
			Error:         "#bf616a",
			Success:       "#a3be8c",
			Stickers:      [6]string{"#eceff4", "#ebcb8b", "#bf616a", "#d08770", "#5e81ac", "#a3be8c"},
			ButtonBG:      "#3b4252",
			ButtonBGHover: "#2e3440",
			ButtonFG:      "#eceff4",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "dracula",
			Accent:        "#8be9fd",
			AccentHover:   "#5bc7dd",
			Border:        "#6272a4",
			Muted:         "#a4b1d8",
			Text:          "#f8f8f2",
			Error:         "#ff5555",
			Success:       "#50fa7b",
			Stickers:      [6]string{"#f8f8f2", "#f1fa8c", "#ff5555", "#ffb86c", "#6272a4", "#50fa7b"},
			ButtonBG:      "#44475a",
			ButtonBGHover: "#282a36",
			ButtonFG:      "#f8f8f2",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "catppuccin",
			Accent:        "#89b4fa",
			AccentHover:   "#5b8bd0",
			Border:        "#585b70",
			Muted:         "#a6adc8",
			Text:          "#cdd6f4",
			Error:         "#f38ba8",
			Success:       "#a6e3a1",
			Stickers:      [6]string{"#cdd6f4", "#f9e2af", "#f38ba8", "#fab387", "#89b4fa", "#a6e3a1"},
			ButtonBG:      "#313244",
			ButtonBGHover: "#1e1e2e",
			ButtonFG:      "#cdd6f4",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "tokyo-night",
			Accent:        "#7aa2f7",
			AccentHover:   "#5a7fd6",
			Border:        "#414868",
			Muted:         "#9aa5ce",
			Text:          "#c0caf5",
			Error:         "#f7768e",
			Success:       "#9ece6a",
			Stickers:      [6]string{"#c0caf5", "#e0af68", "#f7768e", "#ff9e64", "#7aa2f7", "#9ece6a"},
			ButtonBG:      "#24283b",
			ButtonBGHover: "#1a1b26",
			ButtonFG:      "#c0caf5",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "onedark",
			Accent:        "#61afef",
			AccentHover:   "#4b8ec4",
			Border:        "#3e4451",
			Muted:         "#abb2bf",
			Text:          "#abb2bf",
			Error:         "#e06c75",
			Success:       "#98c379",
			Stickers:      [6]string{"#abb2bf", "#e5c07b", "#e06c75", "#d19a66", "#61afef", "#98c379"},
			ButtonBG:      "#2c313a",
			ButtonBGHover: "#21252b",
			ButtonFG:      "#abb2bf",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "rose-pine",
			Accent:        "#c4a7e7",
			AccentHover:   "#a58ac9",
			Border:        "#403d52",
			Muted:         "#908caa",
			Text:          "#e0def4",
			Error:         "#eb6f92",
			Success:       "#9ccfd8",
			Stickers:      [6]string{"#e0def4", "#f6c177", "#eb6f92", "#ebbcba", "#c4a7e7", "#9ccfd8"},
			ButtonBG:      "#26233a",
			ButtonBGHover: "#1f1d2e",
			ButtonFG:      "#e0def4",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "kanagawa",
			Accent:        "#7e9cd8",
			AccentHover:   "#5f7fb8",
			Border:        "#54546d",
			Muted:         "#a6a69c",
			Text:          "#dcd7ba",
			Error:         "#e46876",
			Success:       "#98bb6c",
			Stickers:      [6]string{"#dcd7ba", "#e6c384", "#e46876", "#ffa066", "#7e9cd8", "#98bb6c"},
			ButtonBG:      "#2a2a37",
			ButtonBGHover: "#1f1f28",
			ButtonFG:      "#dcd7ba",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "solarized-dark",
			Accent:        "#268bd2",
			AccentHover:   "#1f6fa8",
			Border:        "#586e75",
			Muted:         "#93a1a1",
			Text:          "#eee8d5",
			Error:         "#dc322f",
			Success:       "#859900",
			Stickers:      [6]string{"#eee8d5", "#b58900", "#dc322f", "#cb4b16", "#268bd2", "#859900"},
			ButtonBG:      "#073642",
			ButtonBGHover: "#002b36",
			ButtonFG:      "#eee8d5",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "github-dark",
			Accent:        "#58a6ff",
			AccentHover:   "#3b82d0",
			Border:        "#30363d",
			Muted:         "#8b949e",
			Text:          "#c9d1d9",
			Error:         "#f85149",
			Success:       "#3fb950",
			Stickers:      [6]string{"#c9d1d9", "#d29922", "#f85149", "#db6d28", "#58a6ff", "#3fb950"},
			ButtonBG:      "#21262d",
			ButtonBGHover: "#161b22",
			ButtonFG:      "#c9d1d9",
			ButtonFGHover: "#ffffff",
		},
		{
			Name:          "mono",
			Accent:        "#e5e7eb",
			AccentHover:   "#9ca3af",
			Border:        "#6b7280",
			Muted:         "#9ca3af",
			Text:          "#f3f4f6",
			Error:         "#d1d5db",
			Success:       "#e5e7eb",
			Stickers:      [6]string{"#f3f4f6", "#d1d5db", "#9ca3af", "#6b7280", "#e5e7eb", "#cbd5e1"},
			ButtonBG:      "#374151",
			ButtonBGHover: "#1f2937",
			ButtonFG:      "#f3f4f6",
			ButtonFGHover: "#ffffff",
			Mono:          true,
		},
	}
}

func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "rbxcb-tui", "config.json")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "rbxcb-tui", "config.json")
}

// LoadConfig reads the optional XDG config. A missing file is not an error.
func LoadConfig() (Config, error) {
	var cfg Config
	path := configPath()
	if path == "" {
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

// SaveConfig writes the config atomically, creating the XDG directory if needed.
func SaveConfig(cfg Config) error {
	path := configPath()
	if path == "" {
		return fmt.Errorf("could not resolve config path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ThemesWithCustom merges built-in themes with user-defined themes.
func ThemesWithCustom(cfg Config) []Theme {
	themes := BuiltinThemes()
	names := make([]string, 0, len(cfg.Themes))
	for name := range cfg.Themes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		themes = append(themes, customTheme(name, cfg.Themes[name]))
	}
	return themes
}

func customTheme(name string, c ThemeConfig) Theme {
	t := defaultTheme()
	t.Name = name
	if c.Accent != "" {
		t.Accent = c.Accent
	}
	if c.AccentHover != "" {
		t.AccentHover = c.AccentHover
	}
	if c.Border != "" {
		t.Border = c.Border
	}
	if c.Muted != "" {
		t.Muted = c.Muted
	}
	if c.Text != "" {
		t.Text = c.Text
	}
	if c.Error != "" {
		t.Error = c.Error
	}
	if c.Success != "" {
		t.Success = c.Success
	}
	if c.ButtonBG != "" {
		t.ButtonBG = c.ButtonBG
	}
	if c.ButtonBGHover != "" {
		t.ButtonBGHover = c.ButtonBGHover
	}
	if c.ButtonFG != "" {
		t.ButtonFG = c.ButtonFG
	}
	if c.ButtonFGHover != "" {
		t.ButtonFGHover = c.ButtonFGHover
	}
	if len(c.Stickers) == 6 {
		copy(t.Stickers[:], c.Stickers)
	}
	t.Mono = c.Mono
	if t.AccentHover == "" {
		t.AccentHover = darkenHex(t.Accent, 0.8)
	}
	if t.ButtonBG == "" {
		t.ButtonBG = darkenHex(t.Border, 0.6)
	}
	if t.ButtonBGHover == "" {
		t.ButtonBGHover = darkenHex(t.ButtonBG, 0.8)
	}
	if t.ButtonFG == "" {
		t.ButtonFG = t.Text
	}
	if t.ButtonFGHover == "" {
		t.ButtonFGHover = contrastInk(t.ButtonBGHover)
	}
	return t
}

func findTheme(themes []Theme, name string) (Theme, bool) {
	for _, t := range themes {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return Theme{}, false
}

// ResolveTheme applies flag/env/config precedence and NO_COLOR.
func ResolveTheme(themes []Theme, flagValue string, noColor bool) (Theme, string) {
	if noColor {
		if t, ok := findTheme(themes, "mono"); ok {
			return t, "NO_COLOR"
		}
	}
	name := flagValue
	if name == "" {
		name = os.Getenv("RBXCB_THEME")
	}
	if name != "" {
		if t, ok := findTheme(themes, name); ok {
			return t, ""
		}
		return defaultTheme(), "unknown theme " + name
	}
	return defaultTheme(), ""
}

func contrastInk(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return "#ffffff"
	}
	lum := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
	if lum > 140 {
		return "#111111"
	}
	return "#ffffff"
}

func darkenHex(hex string, factor float64) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return hex
	}
	r = int(float64(r) * factor)
	g = int(float64(g) * factor)
	b = int(float64(b) * factor)
	return fmt.Sprintf("#%02x%02x%02x", clampByte(r), clampByte(g), clampByte(b))
}

func parseHex(hex string) (int, int, int, bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	r, err1 := strconv.ParseInt(hex[0:2], 16, 32)
	g, err2 := strconv.ParseInt(hex[2:4], 16, 32)
	b, err3 := strconv.ParseInt(hex[4:6], 16, 32)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(r), int(g), int(b), true
}

func clampByte(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
