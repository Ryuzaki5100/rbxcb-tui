package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/solver"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/tui"
)

var version = "dev"

func main() {
	var (
		demo     = flag.Bool("demo", false, "load the sample cube on startup")
		noMouse  = flag.Bool("no-mouse", false, "start with mouse capture disabled")
		noHover  = flag.Bool("no-hover", false, "disable hover highlighting")
		noColor  = flag.Bool("no-color", false, "force the monochrome theme")
		theme    = flag.String("theme", "", "theme name (see --list-themes)")
		aspect   = flag.Float64("cell-aspect", 0, "terminal character aspect ratio (auto by default)")
		noAuto   = flag.Bool("no-auto-aspect", false, "disable automatic cell aspect detection")
		noAnim   = flag.Bool("no-animation", false, "disable face rotation animation")
		list     = flag.Bool("list-themes", false, "list available themes and exit")
		facelets = flag.String("facelets", "", "load a 54-character cube state")
		scramble = flag.String("scramble", "", "apply a scramble such as \"R U R' U'\"")
		stage    = flag.Int("stage", 4, "default solver stage: 2, 3, or 4")
		showVer  = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("rbxcb-tui", version)
		return
	}
	if *list {
		cfg, _ := tui.LoadConfig()
		for _, t := range tui.ThemesWithCustom(cfg) {
			fmt.Println(t.Name)
		}
		return
	}
	if *stage < 2 || *stage > 4 {
		fmt.Fprintln(os.Stderr, "stage must be 2, 3, or 4")
		os.Exit(2)
	}

	model := tui.New(tui.Options{
		Demo:         *demo,
		NoMouse:      *noMouse,
		NoHover:      *noHover,
		NoColor:      *noColor,
		Theme:        *theme,
		CellAspect:   *aspect,
		NoAutoAspect: *noAuto,
		NoAnimation:  *noAnim,
		Facelets:     *facelets,
		Scramble:     *scramble,
		Stage:        solver.Stage(*stage),
	})

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "rbxcb-tui:", err)
		os.Exit(1)
	}
}
