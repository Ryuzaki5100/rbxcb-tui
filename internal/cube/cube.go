package cube

import "fmt"

// Cube is the cubie-level representation of a 3x3x3 Rubik's cube.
type Cube struct {
	EdgePos   [12]uint8
	EdgeOri   [12]int8
	CornerPos [8]uint8
	CornerOri [8]int8
}

var basicOrientationInfo = [6]int8{3, -1, -2, 1, 2, -3}

func faceOfIndex(index int) byte { return faceOrder[index/9] }

// Solved returns a solved cube.
func Solved() Cube {
	var c Cube
	for i := range c.EdgePos {
		c.EdgePos[i] = uint8(i)
		c.EdgeOri[i] = solvedEdgeOrientation[i]
	}
	for i := range c.CornerPos {
		c.CornerPos[i] = uint8(i)
		c.CornerOri[i] = solvedCornerOrientation[i]
	}
	return c
}

// IsSolved reports whether the cube is in the solved state.
func (c Cube) IsSolved() bool { return c == Solved() }

// ApplyTurn applies a single clockwise quarter turn.
func (c Cube) ApplyTurn(face byte) Cube {
	idx, ok := moveIndex[face]
	if !ok {
		panic(fmt.Sprintf("invalid turn %q", face))
	}
	var next Cube
	for j := 0; j < 12; j++ {
		pos := int(c.EdgePos[j])
		ori := int(c.EdgeOri[j]) + 3
		next.EdgeOri[j] = nextEdgeOrientation[idx][pos][ori]
		next.EdgePos[j] = uint8(nextEdgePos[idx][pos])
	}
	for j := 0; j < 8; j++ {
		pos := int(c.CornerPos[j])
		ori := int(c.CornerOri[j]) + 3
		next.CornerOri[j] = nextCornerOrientation[idx][pos][ori]
		next.CornerPos[j] = uint8(nextCornerPos[idx][pos])
	}
	return next
}

// ApplyMove applies a canonical move.
func (c Cube) ApplyMove(m Move) Cube {
	for i := 0; i < m.Amount; i++ {
		c = c.ApplyTurn(m.Face)
	}
	return c
}

// ApplyMoves applies a sequence of moves.
func (c Cube) ApplyMoves(moves []Move) Cube {
	for _, m := range moves {
		c = c.ApplyMove(m)
	}
	return c
}

// String returns a compact debug representation.
func (c Cube) String() string {
	return fmt.Sprintf("Cube{EdgePos:%v EdgeOri:%v CornerPos:%v CornerOri:%v}",
		c.EdgePos, c.EdgeOri, c.CornerPos, c.CornerOri)
}
