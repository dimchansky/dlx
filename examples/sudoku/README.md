# Sudoku solver

A sudoku solver written in Go as an example of usage of package [dlx](https://github.com/dimchansky/dlx).
It expects a partially solved Sudoku board as input, that will be read from standard input. 
Then, it will try to solve the board and print it solved on standard output.

The input format is simple. The board must be given as cells from top to bottom
and left to right, with empty cells represented by underscores and other cells
represented by its value. Example:

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

## Usage

```
go get -u github.com/dimchansky/dlx/examples/sudoku
sudoku -help
```

### Examples

Find first solution:

```
cat sudoku.txt | ./sudoku
```

Find two solutions:

```
cat sudoku.txt | ./sudoku -limit 2
```

Find all solutions:

```
cat sudoku.txt | ./sudoku -limit 0
```
