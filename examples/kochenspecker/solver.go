package main

import (
	"fmt"
	"sort"

	"github.com/dimchansky/dlx"
)

// noRay is the removed argument of newProblem when every ray stays.
const noRay = -1

// problem is the colouring rule of a set of rays. Rays are numbered from 0 here.
type problem struct {
	rayCount int     // how many rays the set has, the removed one included
	removed  int     // the ray left out, or noRay
	bases    [][]int // groups of dimension rays, each needing exactly one green
	pairs    [][]int // perpendicular rays in no shared basis: at most one green
}

// newProblem finds the groups of a set of rays with one ray left out, or with
// none left out if removed is noRay.
func newProblem(rays []ray, dimension, removed int) problem {
	perpendicular := make([][]bool, len(rays))
	for i := range rays {
		perpendicular[i] = make([]bool, len(rays))
		for j := range rays {
			perpendicular[i][j] = i != removed && j != removed && rays[i].perpendicular(rays[j])
		}
	}
	bases := findBases(perpendicular, dimension)
	return problem{
		rayCount: len(rays),
		removed:  removed,
		bases:    bases,
		pairs:    findPairs(perpendicular, bases),
	}
}

// findBases lists, in ascending order of ray number, every group of dimension
// rays that are all perpendicular to each other. No more than dimension rays
// can be, so each group is a complete set of directions.
func findBases(perpendicular [][]bool, dimension int) [][]int {
	var bases [][]int
	var extend func(basis []int, from int)
	extend = func(basis []int, from int) {
		if len(basis) == dimension {
			bases = append(bases, append([]int(nil), basis...))
			return
		}
		for r := from; r < len(perpendicular); r++ {
			if perpendicularToAll(perpendicular[r], basis) {
				extend(append(basis, r), r+1)
			}
		}
	}
	extend(nil, 0)
	return bases
}

func perpendicularToAll(perpendicular []bool, group []int) bool {
	for _, r := range group {
		if !perpendicular[r] {
			return false
		}
	}
	return true
}

// findPairs lists the perpendicular pairs that no basis holds together.
func findPairs(perpendicular [][]bool, bases [][]int) [][]int {
	together := make([][]bool, len(perpendicular))
	for i := range together {
		together[i] = make([]bool, len(perpendicular))
	}
	for _, basis := range bases {
		for _, i := range basis {
			for _, j := range basis {
				together[i][j] = true
			}
		}
	}

	var pairs [][]int
	for i := range perpendicular {
		for j := i + 1; j < len(perpendicular); j++ {
			if perpendicular[i][j] && !together[i][j] {
				pairs = append(pairs, []int{i, j})
			}
		}
	}
	return pairs
}

// columns returns the number of matrix columns: one per group.
func (p problem) columns() int {
	return len(p.bases) + len(p.pairs)
}

// matrixRows returns the number of matrix rows: one per ray, plus one slack row
// per pair.
func (p problem) matrixRows() int {
	rows := p.rayCount + len(p.pairs)
	if p.removed != noRay {
		rows--
	}
	return rows
}

// solve passes the green rays of each colouring to accept until accept returns
// true. Every other ray is red.
//
// The green rays are right only if each ray has a matrix row of its own. That
// holds for every published set with at most one ray removed, as the tests
// prove, but not once the pairs are dropped: rays that were told apart only by
// their pairs then have the same row. The total is still right, because dlx
// counts equal rows separately.
func (p problem) solve(accept func(green []int) bool) {
	matrix, rayOfRow := p.buildMatrix()
	matrix.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(rayOfRow, exactCover))
	}))
}

// buildMatrix lays out its columns in two groups:
//
//	0 .. len(bases)-1                 bases, each needing exactly one green ray
//	len(bases) .. columns()-1         pairs, each taking at most one green ray
//
// Each ray is a matrix row that covers every group the ray belongs to. Picking
// that row paints the ray green, and leaving it out paints the ray red. A basis
// column being covered once is the whole rule for that basis.
//
// A pair may hold no green ray, and dlx has no way to say "at most". So each
// pair also gets a slack row, with that column alone, which the search picks
// when both rays of the pair are red. An empty pair can be covered in only one
// way, so slack rows add no duplicate colourings.
//
// The returned map finds the ray of a row from the text of its column list.
// Slack rows are not in it.
func (p problem) buildMatrix() (dlx.Matrix, map[string]int) {
	matrix := dlx.NewMatrix(p.columns())
	groupsOfRay := make([][]int, p.rayCount)
	for column, group := range append(append([][]int(nil), p.bases...), p.pairs...) {
		for _, r := range group {
			groupsOfRay[r] = append(groupsOfRay[r], column)
		}
	}

	rayOfRow := make(map[string]int, p.rayCount)
	for r, groups := range groupsOfRay {
		matrix.AddRow(groups...)
		rayOfRow[fmt.Sprint(groups)] = r
	}
	for i := range p.pairs {
		matrix.AddRow(len(p.bases) + i)
	}
	return matrix, rayOfRow
}

// decode turns the picked rows into ray numbers, dropping slack rows.
func decode(rayOfRow map[string]int, exactCover [][]int) []int {
	green := make([]int, 0, len(exactCover))
	for _, row := range exactCover {
		if r, ok := rayOfRow[fmt.Sprint(row)]; ok {
			green = append(green, r)
		}
	}
	sort.Ints(green)
	return green
}
