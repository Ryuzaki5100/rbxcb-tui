package cube

import (
	"fmt"
	"strings"
)

// Move is a canonical move: 1, 2, or 3 clockwise quarter turns of a face.
type Move struct {
	Face   byte
	Amount int
}

func (m Move) String() string {
	switch m.Amount {
	case 1:
		return string(m.Face)
	case 2:
		return string(m.Face) + "2"
	case 3:
		return string(m.Face) + "'"
	default:
		return ""
	}
}

// ParseNotation parses standard notation such as "R U R' U2 F2'".
func ParseNotation(notation string) ([]Move, error) {
	var moves []Move
	i := 0
	for i < len(notation) {
		ch := notation[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ',' {
			i++
			continue
		}
		face := ch
		if face >= 'a' && face <= 'z' {
			face -= 'a' - 'A'
		}
		if _, ok := moveIndex[face]; !ok {
			return nil, fmt.Errorf("invalid move %q in %q", string(ch), notation)
		}
		i++
		amount := 1
		if i < len(notation) && notation[i] >= '0' && notation[i] <= '9' {
			digit := notation[i]
			i++
			if digit < '1' || digit > '3' {
				return nil, fmt.Errorf("invalid move amount %q in %q", string(digit), notation)
			}
			amount = int(digit - '0')
		}
		if i < len(notation) && notation[i] == '\'' {
			i++
			if amount == 2 {
				amount = 2
			} else {
				amount = 4 - amount
			}
		}
		moves = append(moves, Move{Face: face, Amount: amount})
	}
	return moves, nil
}

// Expand turns a sequence into individual clockwise quarter turns.
func Expand(moves []Move) []byte {
	var turns []byte
	for _, m := range moves {
		for i := 0; i < m.Amount; i++ {
			turns = append(turns, m.Face)
		}
	}
	return turns
}

// Inverse returns the inverse of a sequence.
func Inverse(moves []Move) []Move {
	result := make([]Move, 0, len(moves))
	for i := len(moves) - 1; i >= 0; i-- {
		m := moves[i]
		amount := 4 - m.Amount
		if amount == 4 {
			amount = 0
		}
		if amount != 0 {
			result = append(result, Move{Face: m.Face, Amount: amount})
		}
	}
	return result
}

// Simplify merges consecutive turns of the same face, dropping identities.
func Simplify(moves []Move) []Move {
	var result []Move
	for _, m := range moves {
		if len(result) > 0 && result[len(result)-1].Face == m.Face {
			amount := (result[len(result)-1].Amount + m.Amount) % 4
			if amount == 0 {
				result = result[:len(result)-1]
			} else {
				result[len(result)-1].Amount = amount
			}
			continue
		}
		if m.Amount%4 != 0 {
			result = append(result, Move{Face: m.Face, Amount: m.Amount % 4})
		}
	}
	return result
}

// FormatMoves renders moves as space-separated standard notation.
func FormatMoves(moves []Move) string {
	parts := make([]string, len(moves))
	for i, m := range moves {
		parts[i] = m.String()
	}
	return strings.Join(parts, " ")
}
