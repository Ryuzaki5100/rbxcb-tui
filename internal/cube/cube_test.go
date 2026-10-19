package cube

import (
	"encoding/json"
	"os"
	"testing"
)

type fixtureState struct {
	EdgePos   []uint8 `json:"edgePos"`
	EdgeOri   []int8  `json:"edgeOri"`
	CornerPos []uint8 `json:"cornerPos"`
	CornerOri []int8  `json:"cornerOri"`
}

type fixtureScramble struct {
	Moves string       `json:"moves"`
	State fixtureState `json:"state"`
}

type fixtures struct {
	SolvedFacelets string                  `json:"solvedFacelets"`
	SampleFacelets string                  `json:"sampleFacelets"`
	Moves          map[string]fixtureState `json:"moves"`
	Scrambles      []fixtureScramble       `json:"scrambles"`
}

func loadFixtures(t *testing.T) fixtures {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/java_fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	var f fixtures
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	return f
}

func stateOf(c Cube) fixtureState {
	return fixtureState{
		EdgePos:   append([]uint8(nil), c.EdgePos[:]...),
		EdgeOri:   append([]int8(nil), c.EdgeOri[:]...),
		CornerPos: append([]uint8(nil), c.CornerPos[:]...),
		CornerOri: append([]int8(nil), c.CornerOri[:]...),
	}
}

func equalState(a, b fixtureState) bool {
	if len(a.EdgePos) != len(b.EdgePos) {
		return false
	}
	for i := range a.EdgePos {
		if a.EdgePos[i] != b.EdgePos[i] || a.EdgeOri[i] != b.EdgeOri[i] {
			return false
		}
	}
	for i := range a.CornerPos {
		if a.CornerPos[i] != b.CornerPos[i] || a.CornerOri[i] != b.CornerOri[i] {
			return false
		}
	}
	return true
}

func TestParseSolvedFacelets(t *testing.T) {
	c, err := ParseFacelets(SolvedColorString())
	if err != nil {
		t.Fatal(err)
	}
	if !c.IsSolved() {
		t.Fatalf("solved facelets parsed to %v", c)
	}
}

func TestMoveEngineMatchesJavaFixtures(t *testing.T) {
	f := loadFixtures(t)
	for notation, want := range f.Moves {
		moves, err := ParseNotation(notation)
		if err != nil {
			t.Fatalf("%s: %v", notation, err)
		}
		got := stateOf(Solved().ApplyMoves(moves))
		if !equalState(got, want) {
			t.Errorf("%s: got %+v want %+v", notation, got, want)
		}
	}
}

func TestScramblesMatchJavaFixtures(t *testing.T) {
	f := loadFixtures(t)
	for _, sc := range f.Scrambles {
		moves, err := ParseNotation(sc.Moves)
		if err != nil {
			t.Fatal(err)
		}
		got := stateOf(Solved().ApplyMoves(moves))
		if !equalState(got, sc.State) {
			t.Errorf("%s: got %+v want %+v", sc.Moves, got, sc.State)
		}
	}
}

func TestRenderRoundTrip(t *testing.T) {
	f := loadFixtures(t)
	states := []Cube{Solved()}
	for _, sc := range f.Scrambles {
		moves, err := ParseNotation(sc.Moves)
		if err != nil {
			t.Fatal(err)
		}
		states = append(states, Solved().ApplyMoves(moves))
	}
	for _, c := range states {
		rendered, err := Render(c, StandardPalette())
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseFacelets(rendered.String())
		if err != nil {
			t.Fatalf("parse rendered: %v", err)
		}
		if parsed != c {
			t.Errorf("round trip mismatch:\n got %v\nwant %v", parsed, c)
		}
	}
}

func TestNotationRoundTrip(t *testing.T) {
	input := "R U R' U2 F2' L3 B"
	moves, err := ParseNotation(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatMoves(moves); got != "R U R' U2 F2 L' B" {
		t.Fatalf("got %q", got)
	}
	if got := cubeApplyAndInverse(moves); !got.IsSolved() {
		t.Fatal("inverse did not solve")
	}
}

func cubeApplyAndInverse(moves []Move) Cube {
	return Solved().ApplyMoves(moves).ApplyMoves(Inverse(moves))
}

func TestParseFaceletsRejectsBadInput(t *testing.T) {
	cases := []string{
		"",
		"short",
		SolvedColorString() + "x",
		SolvedColorString()[:53] + "1",
		"bbbbbbbbb" + SolvedColorString()[9:],
	}
	for _, input := range cases {
		if _, err := ParseFacelets(input); err == nil {
			t.Errorf("expected error for %q", input)
		}
	}
}
