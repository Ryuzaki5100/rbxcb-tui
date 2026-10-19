package solver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Ryuzaki5100/rbxcb-tui/internal/cube"
)

var (
	ErrNoSolution = errors.New("no solution found")
	ErrCancelled  = errors.New("solve cancelled")
	ErrLimit      = errors.New("search limit reached")
)

// Progress describes the state of a running stage search.
type Progress struct {
	Stage       int
	Depth       int
	Forward     int
	Backward    int
	Visited     int
	Elapsed     time.Duration
	Description string
}

// ProgressFunc receives search progress. It must be safe for concurrent use if
// the caller runs solves concurrently.
type ProgressFunc func(Progress)

type node struct {
	cube   cube.Cube
	key    cube.MaskKey
	parent int
	move   cube.Move
	depth  int
}

type limits struct {
	maxDepth int
	maxNodes int
}

// SolveStage runs a bidirectional breadth-first search from c to the solved
// cube, comparing only the state exposed by mask.
func SolveStage(ctx context.Context, c cube.Cube, restrictions []cube.Move, mask cube.Masker, progress ProgressFunc) ([]cube.Move, error) {
	target := cube.Solved()
	startKey := mask.Mask(c)
	targetKey := mask.Mask(target)
	if startKey == targetKey {
		return nil, nil
	}

	started := time.Now()
	fNodes := []node{{cube: c, key: startKey, parent: -1}}
	bNodes := []node{{cube: target, key: targetKey, parent: -1}}
	fMap := map[cube.MaskKey]int{startKey: 0}
	bMap := map[cube.MaskKey]int{targetKey: 0}
	fFrontier := []int{0}
	bFrontier := []int{0}
	lim := limits{maxDepth: 30, maxNodes: 12_000_000}
	visited := 2
	expansions := 0

	for len(fFrontier) > 0 && len(bFrontier) > 0 {
		select {
		case <-ctx.Done():
			return nil, ErrCancelled
		default:
		}

		// Expand the smaller frontier to keep memory use balanced.
		expandForward := len(fFrontier) <= len(bFrontier)
		var nextFrontier []int

		if expandForward {
			for _, idx := range fFrontier {
				n := fNodes[idx]
				if n.depth >= lim.maxDepth {
					continue
				}
				for _, m := range restrictions {
					nextCube := n.cube.ApplyMove(m)
					key := mask.Mask(nextCube)
					if bIdx, ok := bMap[key]; ok {
						fPath := reconstruct(fNodes, idx, m)
						bPath := reconstruct(bNodes, bIdx, cube.Move{})
						solution := append(fPath, cube.Inverse(bPath)...)
						return cube.Simplify(solution), nil
					}
					if _, seen := fMap[key]; seen {
						continue
					}
					fMap[key] = len(fNodes)
					fNodes = append(fNodes, node{cube: nextCube, key: key, parent: idx, move: m, depth: n.depth + 1})
					nextFrontier = append(nextFrontier, len(fNodes)-1)
					visited++
					expansions++
					if visited > lim.maxNodes {
						return nil, ErrLimit
					}
				}
				if expansions%2048 == 0 {
					select {
					case <-ctx.Done():
						return nil, ErrCancelled
					default:
					}
					if progress != nil {
						progress(Progress{Depth: n.depth, Forward: len(fMap), Backward: len(bMap), Visited: visited, Elapsed: time.Since(started)})
					}
				}
			}
			fFrontier = nextFrontier
		} else {
			for _, idx := range bFrontier {
				n := bNodes[idx]
				if n.depth >= lim.maxDepth {
					continue
				}
				for _, m := range restrictions {
					nextCube := n.cube.ApplyMove(m)
					key := mask.Mask(nextCube)
					if fIdx, ok := fMap[key]; ok {
						fPath := reconstruct(fNodes, fIdx, cube.Move{})
						bPath := reconstruct(bNodes, idx, m)
						solution := append(fPath, cube.Inverse(bPath)...)
						return cube.Simplify(solution), nil
					}
					if _, seen := bMap[key]; seen {
						continue
					}
					bMap[key] = len(bNodes)
					bNodes = append(bNodes, node{cube: nextCube, key: key, parent: idx, move: m, depth: n.depth + 1})
					nextFrontier = append(nextFrontier, len(bNodes)-1)
					visited++
					expansions++
					if visited > lim.maxNodes {
						return nil, ErrLimit
					}
				}
			}
			bFrontier = nextFrontier
		}

		if progress != nil {
			progress(Progress{Forward: len(fMap), Backward: len(bMap), Visited: visited, Elapsed: time.Since(started)})
		}
		if len(fFrontier) == 0 || len(bFrontier) == 0 {
			break
		}
	}
	if ctx.Err() != nil {
		return nil, ErrCancelled
	}
	return nil, fmt.Errorf("%w: explored %d states", ErrNoSolution, visited)
}

func reconstruct(nodes []node, idx int, extra cube.Move) []cube.Move {
	var reversed []cube.Move
	for idx >= 0 {
		n := nodes[idx]
		if idx == 0 {
			break
		}
		reversed = append(reversed, n.move)
		idx = n.parent
	}
	result := make([]cube.Move, 0, len(reversed)+1)
	for i := len(reversed) - 1; i >= 0; i-- {
		result = append(result, reversed[i])
	}
	if extra.Amount != 0 {
		result = append(result, extra)
	}
	return result
}
