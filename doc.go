/*
Package dlx implements Knuth's algorithm DLX (algorithm X implemented in terms
of dancing links), which solves exact cover problems.

An exact cover problem is a matrix of 0s and 1s. A solution is a subset of its
rows that contains exactly one 1 in each column. Many puzzles and combinatorial
problems reduce to this form: see the sudoku solver in the examples/sudoku
directory.

Knuth's paper explains the algorithm: https://arxiv.org/pdf/cs/0011047v1.pdf

# Usage

Consider the exact cover problem specified by the matrix:

	1	0	0	1	0	0	1
	1	0	0	1	0	0	0
	0	0	0	1	1	0	1
	0	0	1	0	1	1	0
	0	1	1	0	0	1	1
	0	1	0	0	0	0	1

The following code finds all solutions to the exact cover problem:

	m := dlx.NewMatrix(7)
	m.AddRow(0, 3, 6)
	m.AddRow(0, 3)
	m.AddRow(3, 4, 6)
	m.AddRow(2, 4, 5)
	m.AddRow(1, 2, 5, 6)
	m.AddRow(1, 6)

	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		fmt.Println("Solution found:")
		for _, row := range exactCover {
			fmt.Println(row)
		}
		return false
	}))

The code output:

	Solution found:
	[0 3]
	[2 4 5]
	[1 6]

# Contract

Return true from the accepter to stop the search early. A Matrix stays valid
after Solve returns, however the search ended, so it can be solved again and
more rows can be added to it.

Invalid arguments are programmer errors and panic with a message prefixed
"dlx:". This applies to a negative column count, a column index that is out of
range or repeated within a row, and a nil accepter. A rejected row leaves the
matrix unchanged.

A Matrix is not safe for concurrent use, and an accepter must not call AddRow
or Solve on the matrix being solved. See the Matrix and SolutionAccepter
documentation for the details.
*/
package dlx
