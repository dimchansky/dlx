# dlx [![Go Reference](https://pkg.go.dev/badge/github.com/dimchansky/dlx.svg)](https://pkg.go.dev/github.com/dimchansky/dlx) [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![CI](https://github.com/dimchansky/dlx/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/dimchansky/dlx/actions/workflows/ci.yml) [![Coverage Status](https://coveralls.io/repos/github/dimchansky/dlx/badge.svg?branch=master)](https://coveralls.io/github/dimchansky/dlx?branch=master)

dlx solves exact cover problems in Go, using Knuth's algorithm DLX.

An exact cover problem is a table of 0s and 1s. The task is to pick some of its rows so that every column has a 1 in exactly one picked row: no column is missed and no column is covered twice. Think of the columns as requirements and the rows as choices, each meeting some of the requirements.

DLX is a backtracking search. It picks a column, tries each row that covers it, removes everything that row conflicts with, and continues until all columns are covered or it hits a dead end. The removals are undone cheaply through doubly linked lists, the "dancing links" of the name. DLX is Algorithm X implemented this way, and Knuth's paper ["Dancing Links"](https://arxiv.org/abs/cs/0011047) explains it in full.

## What it can be used for

You turn your problem into a table, and dlx reports the sets of rows that solve it. The problem fits directly if every requirement must be met exactly once:

- Sudoku: each cell, and each digit in each row, column and box, is a column; each possible digit placement is a row. See the [sudoku solver](examples/sudoku).
- Tiling and pentomino puzzles where every cell is filled and, for pentominoes, every piece is used: each cell and piece is a column; each placement of a piece is a row.
- Assignment problems where every task and every slot is used exactly once: each task and slot is a column; each allowed pairing is a row.

## Example programs

Each one is a small command with a README that explains how the problem maps to exact cover. If exact cover is new to you, start with the [gentle introduction](examples/README.md). It also explains [slack rows](examples/README.md#at-most-once-slack-rows), the way to say "at most once".

### Puzzles

| Example | Problem | What it teaches |
|---------|---------|-----------------|
| [sudoku](examples/sudoku) | Solve a Sudoku board. | The standard introduction to exact cover: one move meets four requirements at once. |
| [pentomino](examples/pentomino) | Tile a rectangle with the 12 pentominoes. | Two kinds of column in one matrix, and counting a tiling's flips and half turns as one. |
| [queens](examples/queens) | Place n queens so that no two attack each other. | "At most once" with slack rows, and what slack rows cost the search. |
| [langford](examples/langford) | Line up two copies of each number, with k numbers between the copies of k. | Removing mirror-image solutions from the matrix itself. |
| [cardtrick](examples/cardtrick) | Find a strategy for a trick where the assistant picks which card to hide and how to lay out the rest, so the magician can name it. | Slack rows for spare messages, counting that rules out a deck with no search, and a recipe that reaches decks the search can't. |

### Science and maths

| Example | Problem | What it teaches |
|---------|---------|-----------------|
| [kochenspecker](examples/kochenspecker) | Show that Peres's 33 rays can't be painted green or red so that no two perpendicular rays are both green and every three perpendicular rays include one green. | Proof by zero solutions: the search finds no answer, and that empty result is the Kochen-Specker theorem. |
| [fullerene](examples/fullerene) | Find the double bonds of the C60 molecule. | A scientific count: the 12500 structures found by search match a published total and a calculation that needs no search. |
| [witt](examples/witt) | Find Steiner systems, such as 132 committees of six drawn from twelve people, where any five people share exactly one. | A matrix that is the definition itself, and totals that measure symmetry, cross-checked by a division. |
| [hat](examples/hat) | Cover a disk with copies of the hat, the shape that covers the plane only without repeating. | Slack rows for the kites outside the disk, and a zero result: turned copies alone can't cover this disk of radius 6. |
| [hamming](examples/hamming) | Find perfect codes: sets of words of 0s and 1s where every word is at most one flip from exactly one chosen word. Use one to win a hat game 7 times out of 8. | One row per neighbourhood, with the search's 240 codes rebuilt by Hamming's recipe. |
| [coveringsystems](examples/coveringsystems) | Split all whole numbers into groups like "every 3rd number", so that each number is in exactly one group. | A finite matrix for an infinite question, totals checked against a known sequence, and zero solutions backing a theorem. |

### Music

| Example | Problem | What it teaches |
|---------|---------|-----------------|
| [twelvetone](examples/twelvetone) | Order the twelve notes so that every interval appears exactly once. | The equality trick, which makes two rows agree on a value. The totals match the published counts. |
| [vuza](examples/vuza) | Find where copies of a rhythm must enter so that every beat is struck once, and neither rhythm nor entries repeat. | Filtering in the callback that receives each solution: dlx finds every list of entries, and those that repeat are dropped. The results match published counts. |
| [bells](examples/bells) | Ring all 24 orders of four bells exactly once under the rules of change ringing. | Filtering in the callback that receives each solution, for a rule no column states: the steps must form a single loop. With all four kinds of swap allowed, 10792 of 2655552 covers pass, which is the published total. |

### Computing and scheduling

| Example | Problem | What it teaches |
|---------|---------|-----------------|
| [crew](examples/crew) | Choose crew pairings so that each flight is flown exactly once, at the lowest cost. | Ranking by cost outside dlx, and handling rows with equal columns. |
| [instrselect](examples/instrselect) | Choose machine instructions for an expression by covering its tree with tiles. | Ranking by cost outside dlx, why the tiles join up with no extra rule, and when dynamic programming is the better tool. |
| [perfecthash](examples/perfecthash) | Find a rule that gives each of Go's 25 keywords its own box in a table of exactly 25 boxes, known as a minimal perfect hash. | Two kinds of column, and why a minimal table needs no slack rows: every slot must be used. |

## What it doesn't do

- There are no secondary columns: every column must be covered exactly once, so "at most once" can't be stated directly. You can get that effect by adding a row with that column alone, called a slack row, and dropping such rows from the solutions you receive. This assumes no real row consists of that column alone, since the two could not be told apart. A problem where many requirements are optional, such as scheduling with free time slots, may suit another tool better.
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
