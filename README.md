# dlx [![Build Status](https://travis-ci.org/dimchansky/dlx.svg?branch=master)](https://travis-ci.org/dimchansky/dlx) [![Godoc](https://godoc.org/github.com/dimchansky/dlx?status.png)](https://godoc.org/github.com/dimchansky/dlx)

`dlx` is a Go implementation of [Knuth's algorithm DLX](https://en.wikipedia.org/wiki/Knuth%27s_Algorithm_X) (algorithm X implemented in terms of [dancing links](https://en.wikipedia.org/wiki/Dancing_Links))

A fantastic explanation of dancing links algorithm can be found [here](http://arxiv.org/abs/cs/0011047v1) ([PDF](http://arxiv.org/pdf/cs/0011047v1.pdf)).

## Installation

    go get github.com/dimchansky/dlx

## Examples

There is [sudoku solver](https://github.com/dimchansky/dlx/tree/master/examples/sudoku) included.
The [solver code](https://github.com/dimchansky/dlx/blob/master/examples/sudoku/solver/solver.go) provides an example of using `dlx` library to solve Sudoku.

Consider the exact cover problem specified by the matrix:

|   0   |   1   |   2   |   3   |   4   |   5   |   6   | 
| :---: | :---: | :---: | :---: | :---: | :---: | :---: | 
|   1   |   0   |   0   |   1   |   0   |   0   |   1   | 
|   1   |   0   |   0   |   1   |   0   |   0   |   0   | 
|   0   |   0   |   0   |   1   |   1   |   0   |   1   | 
|   0   |   0   |   1   |   0   |   1   |   1   |   0   | 
|   0   |   1   |   1   |   0   |   0   |   1   |   1   | 
|   0   |   1   |   0   |   0   |   0   |   0   |   1   | 

The following code finds all solutions to the exact cover problem:

```go
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
```

The code output:

    Solution found:
    [0 3]
    [2 4 5]
    [1 6]
