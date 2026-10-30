package main

import (
	"strconv"
	"strings"

	"github.com/dimchansky/dlx"
)

// change is one way to get from a row to the next. It swaps the bells in the
// listed pairs of places, counted from 0.
type change struct {
	name  byte
	swaps [][2]int
}

// allChanges lists every change that moves no bell by more than one place, on
// up to 4 bells. No other change exists: a pair of neighbouring places is 1-2,
// 2-3 or 3-4, and the only two pairs that do not overlap are 1-2 and 3-4.
var allChanges = []change{
	{name: 'a', swaps: [][2]int{{0, 1}}},
	{name: 'b', swaps: [][2]int{{1, 2}}},
	{name: 'c', swaps: [][2]int{{2, 3}}},
	{name: 'd', swaps: [][2]int{{0, 1}, {2, 3}}},
}

// singleSwaps returns the names of the changes that swap one pair of bells.
func singleSwaps(bells int) string {
	return "abc"[:bells-1]
}

// fits reports whether every place the change touches exists on this many bells.
func (c change) fits(bells int) bool {
	for _, swap := range c.swaps {
		if swap[1] >= bells {
			return false
		}
	}
	return true
}

// apply returns the row that follows row under the change.
func (c change) apply(row string) string {
	next := []byte(row)
	for _, swap := range c.swaps {
		next[swap[0]], next[swap[1]] = next[swap[1]], next[swap[0]]
	}
	return string(next)
}

// allRows returns every order of the bells, with rounds, the row 1234..., first.
func allRows(bells int) []string {
	var rows []string
	var extend func(row string)
	extend = func(row string) {
		if len(row) == bells {
			rows = append(rows, row)
			return
		}
		for bell := 1; bell <= bells; bell++ {
			name := strconv.Itoa(bell)
			if !strings.Contains(row, name) {
				extend(row + name)
			}
		}
	}
	extend("")
	return rows
}

// buildMatrix lays out its columns in two groups:
//
//	0 .. n-1      out of row i: the step that leaves row i
//	n .. 2n-1     into row i: the step that arrives at row i
//
// Each allowed step from a row to a row is a matrix row with two 1s. An exact
// cover therefore gives every row one way out and one way in, which is a set of
// closed loops that visit each row once. The matrix cannot say that there must
// be only one loop, so solve leaves that to the caller.
func buildMatrix(rows []string, allowed []change) dlx.Matrix {
	n := len(rows)
	index := make(map[string]int, n)
	for i, row := range rows {
		index[row] = i
	}

	m := dlx.NewMatrix(2 * n)
	for from, row := range rows {
		for _, c := range allowed {
			m.AddRow(from, n+index[c.apply(row)])
		}
	}
	return m
}

// solve passes each cycle cover to accept until accept returns true. A cover is
// given as next, where next[i] is the index in rows of the row that follows
// rows[i]. Each column list comes back in AddRow order, out column first.
func solve(rows []string, allowed []change, accept func(next []int) bool) {
	n := len(rows)
	buildMatrix(rows, allowed).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		next := make([]int, n)
		for _, step := range exactCover {
			next[step[0]] = step[1] - n
		}
		return accept(next)
	}))
}

// extentOf follows a cover from rounds and returns the rows in ringing order,
// with rounds again at the end. It returns nil if the loop through rounds closes
// before it has visited every row, which is the filter the matrix cannot apply.
func extentOf(rows []string, next []int) []string {
	extent := []string{rows[0]}
	for i := next[0]; i != 0; i = next[i] {
		extent = append(extent, rows[i])
	}
	if len(extent) != len(rows) {
		return nil
	}
	return append(extent, rows[0])
}
