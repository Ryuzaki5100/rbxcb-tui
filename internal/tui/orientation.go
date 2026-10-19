package tui

// Cube view orientation tables. Coordinates use x right, y up, z front.
type vec3 struct{ x, y, z int }

var faceNormals = [6]vec3{
	{0, 1, 0},  // U
	{-1, 0, 0}, // L
	{0, 0, 1},  // F
	{1, 0, 0},  // R
	{0, 0, -1}, // B
	{0, -1, 0}, // D
}

var oppositeFace = [6]int{5, 3, 4, 1, 2, 0}

// orientationIndex[front][up][displayCell] = canonical cell on the front face.
var orientationIndex [6][6][9]int

// rightFace[front][up] = the face to the viewer's right.
var rightFace [6][6]int

func init() {
	for f := 0; f < 6; f++ {
		for u := 0; u < 6; u++ {
			rightFace[f][u] = -1
			for i := 0; i < 9; i++ {
				orientationIndex[f][u][i] = -1
			}
		}
	}
	cellCoords := buildFaceCellCoords()
	for f := 0; f < 6; f++ {
		for u := 0; u < 6; u++ {
			if dot(faceNormals[f], faceNormals[u]) != 0 {
				continue
			}
			right := cross(faceNormals[u], faceNormals[f])
			rf := normalFace(right)
			if rf < 0 {
				continue
			}
			rightFace[f][u] = rf
			index := coordIndex(cellCoords[f])
			for r := 0; r < 3; r++ {
				for c := 0; c < 3; c++ {
					p := vec3{
						x: faceNormals[f].x + faceNormals[u].x*(1-r) + right.x*(c-1),
						y: faceNormals[f].y + faceNormals[u].y*(1-r) + right.y*(c-1),
						z: faceNormals[f].z + faceNormals[u].z*(1-r) + right.z*(c-1),
					}
					if idx, ok := index[p]; ok {
						orientationIndex[f][u][r*3+c] = idx
					}
				}
			}
		}
	}
}

func buildFaceCellCoords() [6][9]vec3 {
	var coords [6][9]vec3
	for f := 0; f < 6; f++ {
		for r := 0; r < 3; r++ {
			for c := 0; c < 3; c++ {
				switch faceOrder[f] {
				case 'U':
					coords[f][r*3+c] = vec3{c - 1, 1, r - 1}
				case 'D':
					coords[f][r*3+c] = vec3{c - 1, -1, 1 - r}
				case 'F':
					coords[f][r*3+c] = vec3{c - 1, 1 - r, 1}
				case 'B':
					coords[f][r*3+c] = vec3{1 - c, 1 - r, -1}
				case 'R':
					coords[f][r*3+c] = vec3{1, 1 - r, 1 - c}
				case 'L':
					coords[f][r*3+c] = vec3{-1, 1 - r, c - 1}
				}
			}
		}
	}
	return coords
}

func coordIndex(coords [9]vec3) map[vec3]int {
	m := make(map[vec3]int, 9)
	for i, p := range coords {
		m[p] = i
	}
	return m
}

func dot(a, b vec3) int { return a.x*b.x + a.y*b.y + a.z*b.z }

func cross(a, b vec3) vec3 {
	return vec3{
		x: a.y*b.z - a.z*b.y,
		y: a.z*b.x - a.x*b.z,
		z: a.x*b.y - a.y*b.x,
	}
}

func normalFace(v vec3) int {
	for i, n := range faceNormals {
		if n == v {
			return i
		}
	}
	return -1
}

func defaultUp(front int) int {
	for _, up := range []int{0, 2, 4, 1, 3, 5} {
		if orientationIndex[front][up][0] >= 0 {
			return up
		}
	}
	return 0
}
