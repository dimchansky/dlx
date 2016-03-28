package solver

import "github.com/dimchansky/dlx"

// SolveFirst finds first solution of Sudoku in 81 character string format.
func SolveFirst(s string) (res string, ok bool) {
	Solve(s, SolutionAccepterFunc(func(solution string) bool {
		ok = true
		res = solution
		return true
	}))
	return
}

// SolutionAccepter accepts solutions for the given Sudoku in 81 character string format.
// Each solution is passed to AcceptSolution. Solver stops immediately when AcceptSolution returns true.
type SolutionAccepter interface {
	AcceptSolution(solution string) bool
}

// The SolutionAccepterFunc type is an adapter to allow the use of
// ordinary functions as SolutionAccepter. If f is a function with the appropriate signature,
// SolutionAccepterFunc(f) is a SolutionAccepter that calls f.
type SolutionAccepterFunc func(string) bool

// AcceptSolution calls f(exactCover) and returns its result.
func (f SolutionAccepterFunc) AcceptSolution(solution string) bool {
	return f(solution)
}

// Solve finds all solutions of Sudoku in 81 character string format.
// Each solution is passed to accepter. It stops immediately when accepter AcceptSolution returns true.
func Solve(s string, accepter SolutionAccepter) {
	m := encodeConstraints(s)

	m.Solve(dlx.SolutionAccepterFunc(func(cs [][]int) bool {
		return accepter.AcceptSolution(decodeExactCoverSolution(cs))
	}))
}

func encodeConstraints(s string) dlx.Matrix {
	m := dlx.NewMatrix(324)

	for row, position := 0, 0; row < 9; row++ {
		for column := 0; column < 9; column, position = column+1, position+1 {
			region := row/3*3 + column/3
			digit := int(s[position] - '1') // zero based digit
			if digit >= 0 && digit < 9 {
				m.AddRow(position, 81+row*9+digit, 162+column*9+digit, 243+region*9+digit)
			} else {
				for digit = 0; digit < 9; digit++ {
					m.AddRow(position, 81+row*9+digit, 162+column*9+digit, 243+region*9+digit)
				}
			}
		}
	}

	return m
}

func decodeExactCoverSolution(cs [][]int) string {
	b := make([]byte, len(cs))
	for _, row := range cs {
		position := row[0]
		value := row[1] % 9
		b[position] = byte(value) + '1'
	}
	return string(b)
}
