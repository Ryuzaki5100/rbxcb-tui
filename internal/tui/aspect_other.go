//go:build !linux && !darwin

package tui

func detectCellAspect() (float64, bool) { return 0, false }

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
