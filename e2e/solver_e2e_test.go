package e2e

import (
	"context"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
	"github.com/Ryuzaki5100/rbxcb-tui/internal/solver"
)

func scrambleCount() int {
	if testing.Short() {
		return 1
	}
	if raw := os.Getenv("E2E_SCRAMBLES"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

func randomScramble(rng *rand.Rand, length int) []cube.Move {
	faces := []byte{'U', 'D', 'L', 'R', 'F', 'B'}
	moves := make([]cube.Move, 0, length)
	last := byte(0)
	for len(moves) < length {
		face := faces[rng.IntN(len(faces))]
		if face == last {
			continue
		}
		last = face
		amount := 1 + rng.IntN(3)
		moves = append(moves, cube.Move{Face: face, Amount: amount})
	}
	return moves
}

// E2E solves randomly generated scrambles through the same facelet round-trip
// the TUI uses, for every staged solver.
func E2E(t *testing.T, stage solver.Stage, seeds []uint64, lengths []int, count int) {
	t.Helper()
	for _, seed := range seeds {
		rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
		for i := 0; i < count; i++ {
			length := lengths[i%len(lengths)]
			scramble := randomScramble(rng, length)
			start := cube.Solved().ApplyMoves(scramble)
			rendered, err := cube.Render(start, cube.StandardPalette())
			if err != nil {
				t.Fatalf("seed=%d scramble=%s render: %v", seed, cube.FormatMoves(scramble), err)
			}
			parsed, err := cube.ParseFacelets(rendered.String())
			if err != nil {
				t.Fatalf("seed=%d scramble=%s parse: %v", seed, cube.FormatMoves(scramble), err)
			}
			if parsed != start {
				t.Fatalf("seed=%d scramble=%s facelet round-trip mismatch", seed, cube.FormatMoves(scramble))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			solution, err := solver.Solve(ctx, parsed, stage, nil)
			cancel()
			if err != nil {
				t.Fatalf("stage=%s seed=%d scramble=%s solve: %v", stage, seed, cube.FormatMoves(scramble), err)
			}
			if len(solution) == 0 {
				t.Fatalf("stage=%s seed=%d scramble=%s returned empty solution", stage, seed, cube.FormatMoves(scramble))
			}
			final := parsed.ApplyMoves(solution)
			if !final.IsSolved() {
				t.Fatalf("stage=%s seed=%d scramble=%s solution=%v left cube unsolved", stage, seed, cube.FormatMoves(scramble), solution)
			}
			finalFacelets, err := cube.Render(final, cube.StandardPalette())
			if err != nil || !finalFacelets.IsSolved() {
				t.Fatalf("stage=%s seed=%d final facelets not solved: %v", stage, seed, err)
			}
		}
	}
}

func TestE2ERandomScrambles(t *testing.T) {
	seeds := []uint64{1, 2, 3}
	lengths := []int{1, 5, 10, 20, 25}
	for _, stage := range []solver.Stage{solver.Stage2, solver.Stage3, solver.Stage4} {
		stage := stage
		t.Run(stage.String(), func(t *testing.T) {
			E2E(t, stage, seeds, lengths, scrambleCount())
		})
	}
}
