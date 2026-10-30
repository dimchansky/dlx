// Package solver solves Sudoku boards by reducing them to exact cover problems
// for package dlx.
//
// A board is a string of 81 bytes, row by row. A cell is a digit from '1' to '9',
// or one of '.', '_' and '0' for a blank. Any other board is a programmer error
// and makes Solve and SolveFirst panic: their results cannot carry an error, and
// reporting "no solution" would hide it. Boards from parser.ParseReader are valid.
// Solutions use the same format and contain only digits.
package solver

import (
	"fmt"

	"github.com/dimchansky/dlx"
)

const (
	boardSize = 9
	cellCount = boardSize * boardSize
)

// SolveFirst returns the first solution that Solve finds for board s and true,
// or an empty string and false if the board has no solutions. The choice of
// first solution is deterministic but otherwise arbitrary. It panics if s is
// not a valid board.
func SolveFirst(s string) (res string, ok bool) {
	Solve(s, SolutionAccepterFunc(func(solution string) bool {
		ok = true
		res = solution
		return true
	}))
	return
}

// SolutionAccepter accepts solutions of a Sudoku board in 81 character string format.
// Each solution is passed to AcceptSolution. Solve stops immediately when AcceptSolution returns true.
type SolutionAccepter interface {
	AcceptSolution(solution string) bool
}

// The SolutionAccepterFunc type is an adapter to allow the use of
// ordinary functions as SolutionAccepter. If f is a function with the appropriate signature,
// SolutionAccepterFunc(f) is a SolutionAccepter that calls f.
type SolutionAccepterFunc func(string) bool

// AcceptSolution calls f(solution) and returns its result.
func (f SolutionAccepterFunc) AcceptSolution(solution string) bool {
	return f(solution)
}

// Solve finds the solutions of board s and passes each one to accepter, until
// accepter returns true or no solutions remain. A board with conflicting givens
// has no solutions, so accepter is not called. Solve panics if s is not a valid
// board or if accepter is nil.
func Solve(s string, accepter SolutionAccepter) {
	if accepter == nil {
		panic("solver: nil SolutionAccepter")
	}
	validateBoard(s)

	m := encodeConstraints(s)

	m.Solve(dlx.SolutionAccepterFunc(func(cs [][]int) bool {
		return accepter.AcceptSolution(decodeExactCoverSolution(cs))
	}))
}

func validateBoard(s string) {
	if len(s) != cellCount {
		panic(fmt.Sprintf("solver: board has %d bytes, want %d", len(s), cellCount))
	}
	for position := 0; position < cellCount; position++ {
		c := s[position]
		isDigit := c >= '1' && c <= '9'
		isBlank := c == '.' || c == '_' || c == '0'
		if !isDigit && !isBlank {
			panic(fmt.Sprintf("solver: invalid cell %q at position %d", c, position))
		}
	}
}

func encodeConstraints(s string) dlx.Matrix {
	m := dlx.NewMatrix(324)

	for row, position := 0, 0; row < boardSize; row++ {
		for column := 0; column < boardSize; column, position = column+1, position+1 {
			region := row/3*3 + column/3
			digit := int(s[position] - '1') // zero based digit
			if digit >= 0 && digit < boardSize {
				m.AddRow(position, 81+row*9+digit, 162+column*9+digit, 243+region*9+digit)
			} else {
				for digit = 0; digit < boardSize; digit++ {
					m.AddRow(position, 81+row*9+digit, 162+column*9+digit, 243+region*9+digit)
				}
			}
		}
	}

	return m
}

// decodeExactCoverSolution relies on dlx keeping row values in AddRow order:
// the first holds the position and the second the row and digit.
func decodeExactCoverSolution(cs [][]int) string {
	b := make([]byte, len(cs))
	for _, row := range cs {
		position := row[0]
		value := row[1] % boardSize
		b[position] = byte(value) + '1'
	}
	return string(b)
}
