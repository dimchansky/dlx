# dlx [![Build Status](https://travis-ci.org/dimchansky/dlx.svg?branch=master)](https://travis-ci.org/dimchansky/dlx)

`dlx` is a Go implementation of [Knuth's algorithm DLX](https://en.wikipedia.org/wiki/Knuth%27s_Algorithm_X) (algorithm X implemented in terms of [dancing links](https://en.wikipedia.org/wiki/Dancing_Links))

A fantastic explanation of dancing links algorithm can be found [here](http://arxiv.org/abs/cs/0011047v1) ([PDF](http://arxiv.org/pdf/cs/0011047v1.pdf)).

## Installation

    go get github.com/dimchansky/dlx

## Examples

There is [sudoku solver](https://github.com/dimchansky/dlx/tree/master/examples/sudoku) included.
The [solver code](https://github.com/dimchansky/dlx/blob/master/examples/sudoku/solver/solver.go) provides an example of using `dlx` library to solve Sudoku.