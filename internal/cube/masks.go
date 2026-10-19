package cube

// MaskKey is a comparable encoded mask.
type MaskKey struct {
	Len int
	V   [48]int8
}

func makeKey(values []int) MaskKey {
	var k MaskKey
	k.Len = len(values)
	for i, v := range values {
		k.V[i] = int8(v)
	}
	return k
}

// Masker maps a cube to the state relevant for one solving stage.
type Masker interface {
	Mask(Cube) MaskKey
}

// Stage1 is the 4-stage edge-orientation mask.
type Stage1 struct{}

func (Stage1) Mask(c Cube) MaskKey {
	values := make([]int, 12)
	for i := 0; i < 12; i++ {
		pos := int(c.EdgePos[i])
		ori := int(c.EdgeOri[i])
		if pos < 4 || pos > 7 {
			if abs(ori) == 3 {
				values[pos] = 1
			}
		} else if abs(ori) == 2 {
			values[pos] = 1
		}
	}
	return makeKey(values)
}

// Stage2 is the 4-stage corner-orientation and edge-pairing mask.
type Stage2 struct{}

func (Stage2) Mask(c Cube) MaskKey {
	values := make([]int, 20)
	for i := 0; i < 8; i++ {
		values[int(c.CornerPos[i])] = int(c.CornerOri[i])
	}
	for i := 0; i < 12; i++ {
		pos := int(c.EdgePos[i])
		positionGroup := pos < 4 || pos > 7
		pieceGroup := i < 4 || i > 7
		if positionGroup != pieceGroup {
			values[8+pos] = 1
		}
	}
	return makeKey(values)
}

// Stage3 is the 4-stage permutation/parity mask.
type Stage3 struct{}

func (Stage3) Mask(c Cube) MaskKey {
	parityEdges := make([]int, 12)
	parityCorners := make([]int, 8)
	parityCornerPos := make([]int, 8)
	for i := 0; i < 12; i++ {
		pos := int(c.EdgePos[i])
		parityEdges[pos] = edgePossiblePlacesStage3[pos][i]
	}
	for i := 0; i < 8; i++ {
		pos := int(c.CornerPos[i])
		parityCorners[pos] = cornerPossiblePlacesStage3[pos][i]
		parityCornerPos[pos] = i / 4
	}
	values := make([]int, 0, 32)
	values = append(values, parityEdges[0:4]...)
	values = append(values, parityEdges[8:12]...)
	values = append(values, parityCorners...)
	values = append(values, parityCornerPos...)
	for _, pos := range c.CornerPos {
		values = append(values, int(pos))
	}
	return makeKey(values)
}

// Stage4 is the full cubie-position mask.
type Stage4 struct{}

func (Stage4) Mask(c Cube) MaskKey {
	values := make([]int, 0, 20)
	for _, pos := range c.EdgePos {
		values = append(values, int(pos))
	}
	for _, pos := range c.CornerPos {
		values = append(values, int(pos))
	}
	return makeKey(values)
}

// OLLMask is the CFOP orientation mask.
type OLLMask struct{}

func (OLLMask) Mask(c Cube) MaskKey {
	values := make([]int, 40)
	for i := 0; i < 12; i++ {
		pos := int(c.EdgePos[i])
		if i > 3 {
			values[pos] = i
		}
		values[20+pos] = int(c.EdgeOri[i])
	}
	for i := 0; i < 8; i++ {
		pos := int(c.CornerPos[i])
		if i > 3 {
			values[12+pos] = i
		}
		values[32+pos] = int(c.CornerOri[i])
	}
	return makeKey(values)
}

// PLLMask is the CFOP permutation mask.
type PLLMask struct{}

func (PLLMask) Mask(c Cube) MaskKey {
	values := make([]int, 0, 40)
	for i := 0; i < 12; i++ {
		values = append(values, int(c.EdgePos[i]), int(c.EdgeOri[i]))
	}
	for i := 0; i < 8; i++ {
		values = append(values, int(c.CornerPos[i]), int(c.CornerOri[i]))
	}
	return makeKey(values)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
