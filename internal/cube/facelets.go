package cube

import (
	"fmt"
	"strings"
)

// Facelets is the 54-sticker representation in U, L, F, R, B, D order, each
// face row-major. Values are color letters chosen by the caller.
type Facelets [54]byte

var faceColors = [6]byte{'w', 'o', 'g', 'r', 'b', 'y'}

// StandardPalette maps U, L, F, R, B, D to white, orange, green, red, blue,
// yellow.
func StandardPalette() [6]byte { return faceColors }

// SolvedFacelets returns the solved cube using face letters.
func SolvedFacelets() Facelets {
	var f Facelets
	for face := 0; face < 6; face++ {
		for cell := 0; cell < 9; cell++ {
			f[face*9+cell] = faceOrder[face]
		}
	}
	return f
}

// String renders the 54-character facelet string.
func (f Facelets) String() string { return string(f[:]) }

// SolvedColorString returns the solved facelet string using standard colors.
func SolvedColorString() string {
	var b strings.Builder
	b.Grow(54)
	for face := 0; face < 6; face++ {
		for cell := 0; cell < 9; cell++ {
			b.WriteByte(faceColors[face])
		}
	}
	return b.String()
}

// ColorForFace maps a face letter to its standard display color letter.
func ColorForFace(face byte) byte {
	for i, f := range faceOrder {
		if f == face {
			return faceColors[i]
		}
	}
	return '?'
}

func faceFromBasic(orientation int8) byte {
	for i, v := range basicOrientationInfo {
		if v == orientation {
			return faceOrder[i]
		}
	}
	panic("invalid basic orientation")
}

// Render converts a cubie state into facelets using the supplied palette. The
// solver's corner orientation encoding does not distinguish the two
// non-reference corner stickers, so any consistent assignment is sufficient;
// parsing the result always recovers the same cubie state.
func Render(c Cube, palette [6]byte) (Facelets, error) {
	if err := Validate(c); err != nil {
		return Facelets{}, err
	}
	var f Facelets
	for face := 0; face < 6; face++ {
		for cell := 0; cell < 9; cell++ {
			f[face*9+cell] = palette[face]
		}
	}

	for piece := 0; piece < 12; piece++ {
		pos := int(c.EdgePos[piece])
		ori := c.EdgeOri[piece]
		highSide, lowSide := edgeSides(piece)
		destFace := faceFromBasic(ori)
		a, b := edgeList[pos][0], edgeList[pos][1]
		if faceOfIndex(a) == destFace {
			f[a] = palette[faceIndexOf(highSide)]
			f[b] = palette[faceIndexOf(lowSide)]
		} else {
			f[b] = palette[faceIndexOf(highSide)]
			f[a] = palette[faceIndexOf(lowSide)]
		}
	}

	for piece := 0; piece < 8; piece++ {
		pos := int(c.CornerPos[piece])
		ori := c.CornerOri[piece]
		highSide, lowSides := cornerSides(piece)
		destFace := faceFromBasic(ori)
		idx := cornerList[pos]
		highPos := -1
		for k := 0; k < 3; k++ {
			if faceOfIndex(idx[k]) == destFace {
				highPos = k
				break
			}
		}
		if highPos < 0 {
			return Facelets{}, fmt.Errorf("invalid corner orientation for piece %d", piece)
		}
		f[idx[highPos]] = palette[faceIndexOf(highSide)]
		remaining := make([]int, 0, 2)
		for k := 0; k < 3; k++ {
			if k != highPos {
				remaining = append(remaining, k)
			}
		}
		f[idx[remaining[0]]] = palette[faceIndexOf(lowSides[0])]
		f[idx[remaining[1]]] = palette[faceIndexOf(lowSides[1])]
	}
	return f, nil
}

func edgeSides(piece int) (high, low byte) {
	a, b := faceOfIndex(edgeList[piece][0]), faceOfIndex(edgeList[piece][1])
	if priority[faceIndexOf(a)] >= priority[faceIndexOf(b)] {
		return a, b
	}
	return b, a
}

func cornerSides(piece int) (high byte, low []byte) {
	faces := []byte{
		faceOfIndex(cornerList[piece][0]),
		faceOfIndex(cornerList[piece][1]),
		faceOfIndex(cornerList[piece][2]),
	}
	hi := 0
	for i := 1; i < 3; i++ {
		if priority[faceIndexOf(faces[i])] > priority[faceIndexOf(faces[hi])] {
			hi = i
		}
	}
	high = faces[hi]
	for i, face := range faces {
		if i != hi {
			low = append(low, face)
		}
	}
	return high, low
}

// IsSolved reports whether every face is uniform.
func (f Facelets) IsSolved() bool {
	for face := 0; face < 6; face++ {
		want := f[face*9+4]
		for cell := 0; cell < 9; cell++ {
			if f[face*9+cell] != want {
				return false
			}
		}
	}
	return true
}

// ParseFacelets parses the 54-character color input used by the Java backend.
func ParseFacelets(input string) (Cube, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	if len(input) != 54 {
		return Cube{}, fmt.Errorf("facelets must contain exactly 54 characters, got %d", len(input))
	}

	counts := map[byte]int{}
	for i := 0; i < len(input); i++ {
		c := input[i]
		if c < 'a' || c > 'z' {
			return Cube{}, fmt.Errorf("facelets may only contain letters, found %q", string(c))
		}
		counts[c]++
	}
	if len(counts) != 6 {
		return Cube{}, fmt.Errorf("facelets must use exactly six distinct colors, got %d", len(counts))
	}
	for color, count := range counts {
		if count != 9 {
			return Cube{}, fmt.Errorf("each color must appear exactly 9 times; %q appears %d times", string(color), count)
		}
	}

	var colorToFace [256]byte
	for i, face := range faceOrder {
		c := input[i*9+4]
		if colorToFace[c] != 0 {
			return Cube{}, fmt.Errorf("the six face centers must all have different colors")
		}
		colorToFace[c] = face
	}

	var c Cube
	seenEdges := [12]bool{}
	for pos, idx := range edgeList {
		side1 := colorToFace[input[idx[0]]]
		side2 := colorToFace[input[idx[1]]]
		if side1 == 0 || side2 == 0 {
			return Cube{}, fmt.Errorf("edge at position %d uses an invalid center color", pos)
		}
		bin := binEncoding[faceIndexOf(side1)] ^ binEncoding[faceIndexOf(side2)]
		piece := edgeNumberForPos[bin]
		if piece < 0 || seenEdges[piece] {
			return Cube{}, fmt.Errorf("invalid or duplicate edge piece at position %d", pos)
		}
		seenEdges[piece] = true
		c.EdgePos[piece] = uint8(pos)
		ref := idx[0]
		if priority[faceIndexOf(side2)] > priority[faceIndexOf(side1)] {
			ref = idx[1]
		}
		c.EdgeOri[piece] = basicOrientationInfo[ref/9]
	}

	seenCorners := [8]bool{}
	for pos, idx := range cornerList {
		side1 := colorToFace[input[idx[0]]]
		side2 := colorToFace[input[idx[1]]]
		side3 := colorToFace[input[idx[2]]]
		if side1 == 0 || side2 == 0 || side3 == 0 {
			return Cube{}, fmt.Errorf("corner at position %d uses an invalid center color", pos)
		}
		bin := binEncoding[faceIndexOf(side1)] ^ binEncoding[faceIndexOf(side2)] ^ binEncoding[faceIndexOf(side3)]
		piece := cornerNumberForPos[bin]
		if piece < 0 || seenCorners[piece] {
			return Cube{}, fmt.Errorf("invalid or duplicate corner piece at position %d", pos)
		}
		seenCorners[piece] = true
		c.CornerPos[piece] = uint8(pos)
		ref := idx[0]
		refFace := side1
		if priority[faceIndexOf(side2)] > priority[faceIndexOf(refFace)] {
			ref = idx[1]
			refFace = side2
		}
		if priority[faceIndexOf(side3)] > priority[faceIndexOf(refFace)] {
			ref = idx[2]
			refFace = side3
		}
		c.CornerOri[piece] = basicOrientationInfo[ref/9]
	}

	if err := Validate(c); err != nil {
		return Cube{}, err
	}
	return c, nil
}

func faceIndexOf(face byte) int {
	for i, f := range faceOrder {
		if f == face {
			return i
		}
	}
	panic("invalid face")
}
