package cube

import "fmt"

// Validate checks the structural invariants needed before solving.
func Validate(c Cube) error {
	seenEdges := [12]bool{}
	for _, pos := range c.EdgePos {
		if int(pos) >= 12 || seenEdges[pos] {
			return fmt.Errorf("edge positions are not a valid permutation")
		}
		seenEdges[pos] = true
	}
	seenCorners := [8]bool{}
	for _, pos := range c.CornerPos {
		if int(pos) >= 8 || seenCorners[pos] {
			return fmt.Errorf("corner positions are not a valid permutation")
		}
		seenCorners[pos] = true
	}
	for _, ori := range c.EdgeOri {
		if ori < -3 || ori > 3 || ori == 0 {
			return fmt.Errorf("invalid edge orientation %d", ori)
		}
	}
	for _, ori := range c.CornerOri {
		if ori < -3 || ori > 3 || ori == 0 {
			return fmt.Errorf("invalid corner orientation %d", ori)
		}
	}
	if inversionParity(c.EdgePos[:]) != inversionParity(c.CornerPos[:]) {
		return fmt.Errorf("impossible permutation parity")
	}
	return nil
}

func inversionParity(perm []uint8) int {
	inversions := 0
	for i := 0; i < len(perm); i++ {
		for j := i + 1; j < len(perm); j++ {
			if perm[i] > perm[j] {
				inversions++
			}
		}
	}
	return inversions % 2
}
