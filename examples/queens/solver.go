package main

import "github.com/dimchansky/dlx"

// solve passes each solution of the n-queens problem to accept until accept
// returns true. A solution holds the column of the queen in each row.
func solve(n int, accept func(queens []int) bool) {
	buildMatrix(n).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(n, exactCover))
	}))
}

// buildMatrix lays out its columns in four groups:
//
//	0 .. n-1          board rows
//	n .. 2n-1         board columns
//	2n .. 4n-2        diagonals, numbered by row+column
//	4n-1 .. 6n-3      antidiagonals, numbered by row-column+n-1
//
// Each square of the board is a matrix row that covers its four lines.
//
// Every board row and column holds exactly one queen, so those columns fit exact
// cover as they are. A diagonal holds at most one queen, and dlx has no way to
// say "at most". So each diagonal also gets a slack row: a row with that column
// alone, which the search picks when no queen stands on that diagonal. A solution
// has exactly one way to cover each empty diagonal, so slack rows add no
// duplicate solutions. They cannot be mistaken for queens either, because a queen
// always covers four columns.
func buildMatrix(n int) dlx.Matrix {
	boardColumns := n
	diagonals := 2*n - 1
	firstDiagonal := 2 * n
	firstAntidiagonal := firstDiagonal + diagonals
	columnCount := firstAntidiagonal + diagonals

	m := dlx.NewMatrix(columnCount)
	for row := 0; row < n; row++ {
		for col := 0; col < n; col++ {
			m.AddRow(
				row,
				boardColumns+col,
				firstDiagonal+row+col,
				firstAntidiagonal+row-col+n-1,
			)
		}
	}
	for diagonal := firstDiagonal; diagonal < columnCount; diagonal++ {
		m.AddRow(diagonal)
	}
	return m
}

// decode drops the slack rows and relies on dlx keeping each row's columns in
// AddRow order: the board row comes first, then the board column.
func decode(n int, exactCover [][]int) []int {
	queens := make([]int, n)
	for _, row := range exactCover {
		if len(row) == 1 {
			continue // a slack row: this diagonal is empty
		}
		queens[row[0]] = row[1] - n
	}
	return queens
}
