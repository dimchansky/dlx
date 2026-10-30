# N queens

A command-line solver for the n-queens problem, written as an example of using package [dlx](https://github.com/dimchansky/dlx): place n queens on an n by n chessboard so that no two share a row, column or diagonal.

## How it maps to exact cover

This example shows what to do with an "at most once" requirement. Each board row and column holds exactly one queen, but a diagonal holds at most one, and most diagonals stay empty. dlx requires every column to be covered exactly once, so the problem doesn't fit as it stands.

The fix is a slack row, as described in the [dlx README](../../README.md#what-it-doesnt-do): for each optional column, add a row with that column alone. The search picks it when no queen covers that diagonal.

| Matrix part | Meaning                                               | Count for n=8 |
|-------------|-------------------------------------------------------|---------------|
| Column      | A board row, which needs exactly one queen            | 8             |
| Column      | A board column, which needs exactly one queen         | 8             |
| Column      | A diagonal, which takes at most one queen             | 15            |
| Column      | An antidiagonal, which takes at most one queen        | 15            |
| Row         | A queen on one square, covering its four lines        | 64            |
| Row         | A slack row for one diagonal or antidiagonal          | 30            |

Two facts make this safe, and both must hold whenever you use the recipe:

- Slack rows add no duplicate solutions. An empty diagonal can be covered in only one way, by its own slack row, so each placement of queens is exactly one exact cover.
- Slack rows can be told apart from real rows. A queen's row has four columns and a slack row has one, so [solver.go](solver.go) drops rows of length one when it decodes a solution.

Slack rows have a cost. dlx branches on the column with the fewest remaining rows, and it treats optional diagonals like any other column. On the machine used to write this, the first solution for n=24 took under a second, while n=32 had not finished after two minutes. The total for n=13 took about a second.

Rotations and reflections of a placement count as separate solutions.

| n         | 1 | 2 | 3 | 4 | 5  | 6 | 7  | 8  | 9   | 10  | 11   | 12    |
|-----------|---|---|---|---|----|---|----|----|-----|-----|------|-------|
| Solutions | 1 | 0 | 0 | 2 | 10 | 4 | 40 | 92 | 352 | 724 | 2680 | 14200 |

The tests check these totals and compare every solution with an independent backtracking search.

## Installation

    go install github.com/dimchansky/dlx/examples/queens@latest

## Usage

    queens [-n size] [-limit n] [-count]

- `-n size` sets the board size, 8 by default, from 1 to 64.
- `-limit n` sets how many solutions to print, 1 by default. Zero prints all of them.
- `-count` enumerates every solution and prints the total instead, so `-limit` has no effect with it.

`-help` prints the usage text to standard error.

Solutions and totals go to standard output. Errors go to standard error, prefixed `queens:`, except for the flag package's own message about an invalid flag.

| Exit code | Meaning                                                 |
|-----------|---------------------------------------------------------|
| 0         | At least one solution exists, or `-help` was given      |
| 1         | The board has no solutions                              |
| 2         | Invalid usage, or a write failure                       |

## Examples

Find the first solution for a standard chessboard:

    queens

Output:

    Solution 1:
    Q . . . . . . .
    . . . . Q . . .
    . . . . . . . Q
    . . . . . Q . .
    . . Q . . . . .
    . . . . . . Q .
    . Q . . . . . .
    . . . Q . . . .

Count the solutions for n=12:

    queens -n 12 -count

Output:

    Solutions: 14200

A board of size 2 or 3 has no solutions, so this exits with code 1:

    queens -n 3

Output, on standard error:

    queens: no solution found
