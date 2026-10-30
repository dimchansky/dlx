# dlx [![Go Reference](https://pkg.go.dev/badge/github.com/dimchansky/dlx.svg)](https://pkg.go.dev/github.com/dimchansky/dlx) [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![CI](https://github.com/dimchansky/dlx/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/dimchansky/dlx/actions/workflows/ci.yml) [![Coverage Status](https://coveralls.io/repos/github/dimchansky/dlx/badge.svg?branch=master)](https://coveralls.io/github/dimchansky/dlx?branch=master)

dlx solves exact cover problems in Go, using Knuth's algorithm DLX.

An exact cover problem is a table of 0s and 1s. The task is to pick some of its rows so that every column has a 1 in exactly one picked row: no column is missed and no column is covered twice. Think of the columns as requirements and the rows as choices, each meeting some of the requirements.

DLX is a backtracking search. It picks a column, tries each row that covers it, removes everything that row conflicts with, and continues until all columns are covered or it hits a dead end. The removals are undone cheaply through doubly linked lists, the "dancing links" of the name. [Knuth's paper](https://arxiv.org/abs/cs/0011047) explains it in full.

## What it can be used for

You turn your problem into a table, and dlx reports the sets of rows that solve it. The problem fits directly if every requirement must be met exactly once:

- Sudoku: each cell, and each digit in each row, column and box, is a column; each possible digit placement is a row. See the [sudoku solver](examples/sudoku).
- Tiling and pentomino puzzles where every cell is filled and, for pentominoes, every piece is used: each cell and piece is a column; each placement of a piece is a row.
- Assignment problems where every task and every slot is used exactly once: each task and slot is a column; each allowed pairing is a row.

## What it doesn't do

- There are no secondary columns: every column must be covered exactly once, so "at most once" can't be stated directly. You can get that effect by adding a row with that column alone, and dropping such rows from the solutions you receive. A problem where many requirements are optional, such as scheduling with free time slots, may suit another tool better.
- It doesn't optimize. Rows have no costs or weights, so it finds solutions but doesn't rank them.
- It can't be cancelled or timed out. Exact cover is NP-complete, so a search can take very long, and your code only runs when a solution is found.
- It doesn't identify rows. A solution lists each picked row's column indices, not its position, so your columns must let you tell the rows apart. The sudoku solver decodes each placement from its column indices.

## Installation

    go get github.com/dimchansky/dlx

Requires Go 1.19 or later. No dependencies outside the standard library.

## Example

This matrix has seven columns and six rows:

|  0  |  1  |  2  |  3  |  4  |  5  |  6  |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
|  1  |  0  |  0  |  1  |  0  |  0  |  1  |
|  1  |  0  |  0  |  1  |  0  |  0  |  0  |
|  0  |  0  |  0  |  1  |  1  |  0  |  1  |
|  0  |  0  |  1  |  0  |  1  |  1  |  0  |
|  0  |  1  |  1  |  0  |  0  |  1  |  1  |
|  0  |  1  |  0  |  0  |  0  |  0  |  1  |

Each `AddRow` call lists the columns that hold a 1 in that row:

```go
package main

import (
	"fmt"

	"github.com/dimchansky/dlx"
)

func main() {
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
		return false // keep searching
	}))
}
```

Output:

    Solution found:
    [0 3]
    [2 4 5]
    [1 6]

This is the only solution: the second, fourth and sixth rows.

### Stopping early

Return true from the accepter to stop the search. This matrix has two solutions, and the search ends after the first:

```go
m := dlx.NewMatrix(2)
m.AddRow(0)
m.AddRow(1)
m.AddRow(0, 1)

m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
	fmt.Println(exactCover) // [[0] [1]]
	return true
}))
```

## Behavior

- Solution order is deterministic: matrices built by the same `AddRow` calls report the same solutions in the same order. The search branches on the leftmost column with the fewest remaining rows and tries that column's rows in `AddRow` order.
- Within a solution, rows appear in the order the search selected them, not in `AddRow` order. Each row lists its columns in the order they were passed to `AddRow`.
- The accepter owns the slices it receives: they are freshly allocated for each solution, so it can keep and modify them. `AddRow` doesn't keep the slice passed to it.
- A `Matrix` stays valid after `Solve` returns, however the search ended: you can solve it again or add more rows.
- If the accepter panics, the matrix is left in an undefined state and must be discarded.
- A row with no columns is ignored. A row added twice counts as two rows and gives separate solutions.
- A matrix with no columns has one solution, which is empty. A matrix with a column that no row covers has none.
- Invalid arguments panic with a message prefixed `dlx:`: a negative column count, a column index that is out of range or repeated within a row, and a nil `SolutionAccepter`. A rejected row leaves the matrix unchanged. A nil `SolutionAccepterFunc` is not detected and panics with a runtime error when the first solution is reported.
- A `Matrix` is not safe for concurrent use, including concurrent calls to `Solve`.
- The accepter must not call `AddRow` or `Solve` on the matrix being solved.

## License

[Apache License 2.0](LICENSE)
