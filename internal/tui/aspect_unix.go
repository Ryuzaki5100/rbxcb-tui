//go:build linux || darwin

package tui

import (
	"os"

	"golang.org/x/sys/unix"
)

// detectCellAspect reads the terminal's pixel dimensions and derives the
// character cell height/width ratio.
func detectCellAspect() (float64, bool) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, false
	}
	return aspectFromPixels(ws.Xpixel, ws.Ypixel, ws.Col, ws.Row)
}

func aspectFromPixels(xpixel, ypixel, cols, rows uint16) (float64, bool) {
	if xpixel == 0 || ypixel == 0 || cols == 0 || rows == 0 {
		return 0, false
	}
	aspect := (float64(ypixel) / float64(rows)) / (float64(xpixel) / float64(cols))
	if aspect < 1.2 || aspect > 2.6 {
		return 0, false
	}
	return aspect, true
}
