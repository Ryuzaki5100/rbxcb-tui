package tui

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
)

// Color is one of the six standard sticker colors.
type Color uint8

const (
	White Color = iota
	Yellow
	Red
	Orange
	Blue
	Green
	Empty Color = 255
)

var colorNames = [6]string{"White", "Yellow", "Red", "Orange", "Blue", "Green"}
var colorLetters = [6]byte{'w', 'y', 'r', 'o', 'b', 'g'}
var monoPatterns = [6]string{"█", "▓", "▒", "░", "▚", "▞"}

func colorFromLetter(b byte) (Color, bool) {
	for i, c := range colorLetters {
		if c == b {
			return Color(i), true
		}
	}
	return Empty, false
}

func (c Color) Letter() byte {
	if c == Empty {
		return '.'
	}
	return colorLetters[c]
}

func (c Color) Name() string {
	if c == Empty {
		return "empty"
	}
	return colorNames[c]
}

// Editor holds the user's 54-cell color input. It is never mutated by solving.
type Editor struct {
	Faces   [6][9]Color
	Face    int
	Up      int
	Cell    int
	Current Color
}

func NewEditor() Editor {
	e := Editor{Current: White, Face: 0, Up: 4}
	for f := 0; f < 6; f++ {
		for c := 0; c < 9; c++ {
			e.Faces[f][c] = Empty
		}
	}
	return e
}

func (e *Editor) SetFace(i int) {
	if i < 0 {
		i = 5
	}
	e.Face = i % 6
	if orientationIndex[e.Face][e.Up][0] < 0 {
		e.Up = defaultUp(e.Face)
	}
}

func (e *Editor) canonCell(display int) int {
	if display < 0 {
		display = 0
	}
	if display > 8 {
		display = 8
	}
	if idx := orientationIndex[e.Face][e.Up][display]; idx >= 0 {
		return idx
	}
	return display
}

func (e *Editor) DisplayColor(display int) Color {
	return e.Faces[e.Face][e.canonCell(display)]
}

func (e *Editor) PaintAt(display int, color Color) {
	e.Faces[e.Face][e.canonCell(display)] = color
	e.Current = color
}

func (e *Editor) ClearAt(display int) {
	e.Faces[e.Face][e.canonCell(display)] = Empty
}

func (e *Editor) MoveCell(dr, dc int) {
	r, c := e.Cell/3, e.Cell%3
	r = clamp(r+dr, 0, 2)
	c = clamp(c+dc, 0, 2)
	e.Cell = r*3 + c
}

func (e *Editor) Paint(color Color) {
	e.PaintAt(e.Cell, color)
	if e.Cell < 8 {
		e.Cell++
	}
}

func (e *Editor) Clear() {
	e.ClearAt(e.Cell)
}

func (e *Editor) Filled() int {
	n := 0
	for f := 0; f < 6; f++ {
		for c := 0; c < 9; c++ {
			if e.Faces[f][c] != Empty {
				n++
			}
		}
	}
	return n
}

func (e *Editor) Counts() [6]int {
	var counts [6]int
	for f := 0; f < 6; f++ {
		for c := 0; c < 9; c++ {
			if e.Faces[f][c] != Empty {
				counts[e.Faces[f][c]]++
			}
		}
	}
	return counts
}

func (e *Editor) Centers() ([6]Color, bool) {
	var centers [6]Color
	seen := map[Color]bool{}
	for f := 0; f < 6; f++ {
		centers[f] = e.Faces[f][4]
		if centers[f] == Empty || seen[centers[f]] {
			return centers, false
		}
		seen[centers[f]] = true
	}
	return centers, true
}

// FaceletsString serializes the editor in the U,L,F,R,B,D row-major order
// expected by the solver and facelet parser.
func (e *Editor) FaceletsString() (string, error) {
	if e.Filled() != 54 {
		return "", fmt.Errorf("fill all 54 cells (%d/54 done)", e.Filled())
	}
	if _, ok := e.Centers(); !ok {
		return "", fmt.Errorf("the six face centers must all have different colors")
	}
	var b strings.Builder
	b.Grow(54)
	for f := 0; f < 6; f++ {
		for c := 0; c < 9; c++ {
			b.WriteByte(e.Faces[f][c].Letter())
		}
	}
	return b.String(), nil
}

func (e *Editor) Validate() (cube.Cube, error) {
	s, err := e.FaceletsString()
	if err != nil {
		return cube.Cube{}, err
	}
	c, err := cube.ParseFacelets(s)
	if err != nil {
		return cube.Cube{}, err
	}
	return c, nil
}

func (e *Editor) LoadFacelets(s string) error {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 54 {
		return fmt.Errorf("facelets must contain 54 characters")
	}
	next := Editor{
		Current: e.Current,
		Face:    e.Face,
		Up:      e.Up,
		Cell:    e.Cell,
	}
	for f := 0; f < 6; f++ {
		for c := 0; c < 9; c++ {
			color, ok := colorFromLetter(s[f*9+c])
			if !ok {
				return fmt.Errorf("unknown color %q", string(s[f*9+c]))
			}
			next.Faces[f][c] = color
		}
	}
	if orientationIndex[next.Face][next.Up][0] < 0 {
		next.Up = defaultUp(next.Face)
	}
	*e = next
	return nil
}

// LoadScramble replaces the editor input with a rendered scramble using the
// standard color scheme.
func (e *Editor) LoadScramble(moves []cube.Move) error {
	state := cube.Solved().ApplyMoves(moves)
	rendered, err := cube.Render(state, cube.StandardPalette())
	if err != nil {
		return err
	}
	return e.LoadFacelets(rendered.String())
}

// RandomScramble generates a legal random scramble.
func RandomScramble(rng *rand.Rand, length int) []cube.Move {
	faces := []byte{'U', 'D', 'L', 'R', 'F', 'B'}
	moves := make([]cube.Move, 0, length)
	last := byte(0)
	for len(moves) < length {
		face := faces[rng.IntN(len(faces))]
		if face == last {
			continue
		}
		last = face
		moves = append(moves, cube.Move{Face: face, Amount: 1 + rng.IntN(3)})
	}
	return moves
}

var faceOrder = [6]byte{'U', 'L', 'F', 'R', 'B', 'D'}

func (e *Editor) MoveFace(direction int) {
	switch direction {
	case 0: // pitch up
		old := e.Face
		e.Face = e.Up
		e.Up = oppositeFace[old]
	case 1: // pitch down
		old := e.Face
		e.Face = oppositeFace[e.Up]
		e.Up = old
	case 2: // yaw left
		right := rightFace[e.Face][e.Up]
		if right >= 0 {
			e.Face = oppositeFace[right]
		}
	case 3: // yaw right
		right := rightFace[e.Face][e.Up]
		if right >= 0 {
			e.Face = right
		}
	}
	if orientationIndex[e.Face][e.Up][0] < 0 {
		e.Up = defaultUp(e.Face)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
