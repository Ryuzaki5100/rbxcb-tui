package solver

import (
	"context"
	"testing"
	"time"

	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
)

const sampleFacelets = "ybgrbrrbbrgbooygorwyyrwyggboboorgwbryobyygyrwwwowgwowg"

func TestSolvedCubeNeedsNoMoves(t *testing.T) {
	c, err := cube.ParseFacelets(cube.SolvedColorString())
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []Stage{Stage2, Stage3, Stage4} {
		solution, err := Solve(context.Background(), c, stage, nil)
		if err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		if len(solution) != 0 {
			t.Fatalf("%s: expected no moves, got %v", stage, solution)
		}
	}
}

func TestSampleFaceletsAllStages(t *testing.T) {
	c, err := cube.ParseFacelets(sampleFacelets)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []Stage{Stage2, Stage3, Stage4} {
		stage := stage
		t.Run(stage.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			solution, err := Solve(ctx, c, stage, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(solution) == 0 {
				t.Fatal("empty solution")
			}
			after := c.ApplyMoves(solution)
			if !after.IsSolved() {
				t.Fatalf("not solved after %v", solution)
			}
		})
	}
}

func TestKnownScramblesAllStages(t *testing.T) {
	scrambles := []string{
		"R U R' U'",
		"R U F' L D2 B R2",
	}
	for _, scramble := range scrambles {
		moves, err := cube.ParseNotation(scramble)
		if err != nil {
			t.Fatal(err)
		}
		start := cube.Solved().ApplyMoves(moves)
		for _, stage := range []Stage{Stage2, Stage3, Stage4} {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			solution, err := Solve(ctx, start, stage, nil)
			cancel()
			if err != nil {
				t.Fatalf("%s %s: %v", scramble, stage, err)
			}
			if !start.ApplyMoves(solution).IsSolved() {
				t.Fatalf("%s %s: solution %v did not solve", scramble, stage, solution)
			}
		}
	}
}

func TestCancel(t *testing.T) {
	moves, _ := cube.ParseNotation("U F2 U' F2 D2 B2 L2 U' B2 U' B2 U2 L R B D2 F2 L R'")
	start := cube.Solved().ApplyMoves(moves)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Solve(ctx, start, Stage2, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}
