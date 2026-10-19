package solver

import (
	"context"
	"fmt"

	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
)

// Stage identifies a staged solving algorithm.
type Stage int

const (
	Stage2 Stage = 2
	Stage3 Stage = 3
	Stage4 Stage = 4
)

func (s Stage) String() string {
	switch s {
	case Stage2:
		return "2-stage"
	case Stage3:
		return "3-stage"
	case Stage4:
		return "4-stage"
	default:
		return fmt.Sprintf("stage %d", int(s))
	}
}

// ThreeStage1 is the edge/corner orientation mask used by the 2- and 3-stage
// solvers.
type ThreeStage1 struct{}

func (ThreeStage1) Mask(c cube.Cube) cube.MaskKey {
	edgeMask := make([][2]int, 12)
	for i := 0; i < 12; i++ {
		pos := int(c.EdgePos[i])
		if i >= 4 && i < 8 {
			edgeMask[pos][0] = 1
		}
		edgeMask[pos][1] = int(c.EdgeOri[i])
	}
	values := make([]int, 0, 32)
	for _, e := range edgeMask {
		values = append(values, e[0], e[1])
	}
	for i := 0; i < 8; i++ {
		values = append(values, 0)
	}
	for i := 0; i < 8; i++ {
		values[24+int(c.CornerPos[i])] = int(c.CornerOri[i])
	}
	return cube.MaskKey{Len: len(values), V: fill(values)}
}

func fill(values []int) [48]int8 {
	var out [48]int8
	for i, v := range values {
		out[i] = int8(v)
	}
	return out
}

var moveSets = map[Stage][][]cube.Move{}

func init() {
	all := parseMoves("U U2 U' D D2 D' L L2 L' R R2 R' F F2 F' B B2 B'")
	two := [][]cube.Move{all, parseMoves("U U2 U' D D2 D' R2 F2 L2 B2")}
	three := [][]cube.Move{all, parseMoves("U U2 U' R2 F2 B2 L2 D D2 D'"), parseMoves("U2 R2 F2 B2 L2 D2")}
	four := [][]cube.Move{all, parseMoves("U U2 U' R R2 R' F2 B2 L L2 L' D D2 D'"), parseMoves("U U2 U' R2 F2 B2 L2 D D2 D'"), parseMoves("U2 R2 F2 B2 L2 D2")}
	moveSets[Stage2] = two
	moveSets[Stage3] = three
	moveSets[Stage4] = four
}

func parseMoves(s string) []cube.Move {
	moves, err := cube.ParseNotation(s)
	if err != nil {
		panic(err)
	}
	return moves
}

func masksFor(stage Stage) []cube.Masker {
	switch stage {
	case Stage2:
		return []cube.Masker{ThreeStage1{}, cube.Stage4{}}
	case Stage3:
		return []cube.Masker{ThreeStage1{}, cube.Stage3{}, cube.Stage4{}}
	default:
		return []cube.Masker{cube.Stage1{}, cube.Stage2{}, cube.Stage3{}, cube.Stage4{}}
	}
}

// Solve runs the selected staged algorithm.
func Solve(ctx context.Context, c cube.Cube, stage Stage, progress ProgressFunc) ([]cube.Move, error) {
	if err := cube.Validate(c); err != nil {
		return nil, err
	}
	set, ok := moveSets[stage]
	if !ok {
		return nil, fmt.Errorf("unsupported stage %d", stage)
	}
	masks := masksFor(stage)
	var all []cube.Move
	for i, mask := range masks {
		local := func(p Progress) {
			p.Stage = i + 1
			p.Description = stage.String()
			if progress != nil {
				progress(p)
			}
		}
		solution, err := SolveStage(ctx, c, set[i], mask, local)
		if err != nil {
			return nil, fmt.Errorf("%s stage %d: %w", stage, i+1, err)
		}
		all = append(all, solution...)
		c = c.ApplyMoves(solution)
	}
	return cube.Simplify(all), nil
}

// SolveOLL solves the CFOP OLL stage.
func SolveOLL(ctx context.Context, c cube.Cube, progress ProgressFunc) ([]cube.Move, error) {
	if err := cube.Validate(c); err != nil {
		return nil, err
	}
	return SolveStage(ctx, c, parseMoves("R R2 R' U U2 U' F F2 F' B B2 B' L L2 L'"), cube.OLLMask{}, progress)
}

// SolvePLL solves the CFOP PLL stage.
func SolvePLL(ctx context.Context, c cube.Cube, progress ProgressFunc) ([]cube.Move, error) {
	if err := cube.Validate(c); err != nil {
		return nil, err
	}
	return SolveStage(ctx, c, parseMoves("U U2 U' R R2 R' F F2 F' B B2 B' L L2 L' D D2 D'"), cube.PLLMask{}, progress)
}
