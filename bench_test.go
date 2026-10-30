package dlx_test

import (
	"fmt"
	"testing"

	"github.com/dimchansky/dlx"
)

const (
	sudokuColumns = 4 * 9 * 9

	// hardBoard has 17 givens and a single solution.
	hardBoard = ".......2143.......6........2.15..........637...........68...4.....23........7...."
	// unsolvableBoard has no solutions, so solving it never builds an exact cover.
	unsolvableBoard = ".....5.8....6.1.43..........1.5........1.6...3.......553.....61........4........."
	emptyBoard      = "................................................................................."
)

// newSudokuMatrix builds the exact cover matrix of a board: 324 columns and up
// to 729 rows. The cell, row, column and box constraints use one column each.
func newSudokuMatrix(board string) dlx.Matrix {
	m := dlx.NewMatrix(sudokuColumns)
	for position := 0; position < len(board); position++ {
		row, column := position/9, position%9
		box := row/3*3 + column/3
		for digit := 0; digit < 9; digit++ {
			if given := board[position]; given != '.' && int(given-'1') != digit {
				continue
			}
			m.AddRow(position, 81+row*9+digit, 162+column*9+digit, 243+box*9+digit)
		}
	}
	return m
}

// newRooksMatrix builds the n-by-n rooks placement problem: one rook per rank
// and per file. It has 2n columns, n*n rows and n! solutions.
func newRooksMatrix(n int) dlx.Matrix {
	m := dlx.NewMatrix(2 * n)
	for rank := 0; rank < n; rank++ {
		for file := 0; file < n; file++ {
			m.AddRow(rank, n+file)
		}
	}
	return m
}

func factorial(n int) int {
	f := 1
	for i := 2; i <= n; i++ {
		f *= i
	}
	return f
}

// benchmarkSolve solves one matrix repeatedly and fails unless every run
// reports wantSolutions solutions.
func benchmarkSolve(b *testing.B, m dlx.Matrix, wantSolutions int) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		solutions := 0
		m.Solve(dlx.SolutionAccepterFunc(func([][]int) bool {
			solutions++
			return false
		}))
		if solutions != wantSolutions {
			b.Fatalf("got %d solutions, want %d", solutions, wantSolutions)
		}
	}
}

func BenchmarkAddRow_Sudoku(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newSudokuMatrix(emptyBoard)
	}
}

func BenchmarkSolve_Sudoku(b *testing.B) {
	benchmarkSolve(b, newSudokuMatrix(hardBoard), 1)
}

// BenchmarkSolve_SudokuNoSolution measures the search alone: no solution is
// reported, so no exact cover is built.
func BenchmarkSolve_SudokuNoSolution(b *testing.B) {
	benchmarkSolve(b, newSudokuMatrix(unsolvableBoard), 0)
}

func BenchmarkSolve_Rooks(b *testing.B) {
	for _, n := range []int{6, 7, 8} {
		n := n
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			benchmarkSolve(b, newRooksMatrix(n), factorial(n))
		})
	}
}

// BenchmarkSolve_EarlyStop stops at the first solution of the empty board and
// reuses the matrix, so it also pays for restoring the matrix after the stop.
func BenchmarkSolve_EarlyStop(b *testing.B) {
	m := newSudokuMatrix(emptyBoard)
	accepter := dlx.SolutionAccepterFunc(func([][]int) bool { return true })
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Solve(accepter)
	}
}
