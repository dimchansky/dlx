# Sudoku solver

A command-line Sudoku solver, written as an example of using package [dlx](https://github.com/dimchansky/dlx). It reads a board from standard input and prints its solutions to standard output. The [solver package](solver/solver.go) shows how a board is turned into an exact cover problem.

## Installation

    go install github.com/dimchansky/dlx/examples/sudoku@latest

## Input

A board is nine lines of nine cells, with whitespace between the cells. A cell is a digit from 1 to 9, or `_` or `0` for a blank:

```
_ 8 _ _ _ 9 7 4 3
_ 5 _ _ _ 8 _ 1 _
_ 1 _ _ _ _ _ _ _
8 _ _ _ _ 5 _ _ _
_ _ _ 8 _ 4 _ _ _
_ _ _ 3 _ _ _ _ 6
_ _ _ _ _ _ _ 7 _
_ 3 _ 5 _ _ _ 8 _
9 7 2 4 _ _ _ 5 _
```

- `.` is not accepted as a blank by the command. The `solver` package accepts it, but the command's parser does not.
- The board starts on the first line, so a blank line before it or inside it is an error.
- Only blank lines may follow the ninth line. Anything else after it is an error.
- Input is read to the end, so if you type a board at a terminal, finish with Ctrl-D.
- A board with conflicting digits is not an input error. It has no solutions.

## Usage

    sudoku [-limit n] < board.txt

`-limit n` sets how many solutions to print, 1 by default. Zero prints all of them, which may never finish on a board with few digits: an empty board has too many solutions to print.

`-help` prints the usage text to standard error.

Solutions go to standard output. Errors go to standard error, prefixed `sudoku:`, except for the flag package's own message about an invalid flag.

| Exit code | Meaning                                                        |
|-----------|----------------------------------------------------------------|
| 0         | At least one solution was printed, or `-help` was given        |
| 1         | The board has no solutions                                     |
| 2         | Invalid usage, invalid input, or a read or write failure       |

If standard output is a closed pipe, as with `sudoku < sudoku.txt | true`, the command is killed by SIGPIPE instead of exiting with code 2. Shells report that as 141.

## Examples

These commands use [sudoku.txt](sudoku.txt), the board shown above, which has 8 solutions. Run them from this directory.

Find the first solution:

    sudoku < sudoku.txt

Output:

    Solution 1:
    2 8 6 1 5 9 7 4 3
    4 5 7 6 3 8 2 1 9
    3 1 9 7 4 2 5 6 8
    8 2 1 9 6 5 4 3 7
    6 9 3 8 7 4 1 2 5
    7 4 5 3 2 1 8 9 6
    5 6 8 2 1 3 9 7 4
    1 3 4 5 9 7 6 8 2
    9 7 2 4 8 6 3 5 1

Find two solutions:

    sudoku -limit 2 < sudoku.txt

Find all solutions:

    sudoku -limit 0 < sudoku.txt
