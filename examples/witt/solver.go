package main

import (
	"math/bits"
	"sort"

	"github.com/dimchansky/dlx"
)

// spec names a Steiner system S(t,k,v): groups of k points, called blocks,
// chosen from v points so that every set of t points lies in exactly one block.
type spec struct {
	t, k, v int
}

// solve passes each Steiner system to accept until accept returns true. A
// system is a sorted list of blocks, and each block is a sorted list of points.
func solve(s spec, accept func(blocks [][]int) bool) {
	matrix, subsets := buildMatrix(s)
	matrix.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(subsets, exactCover))
	}))
}

// countSolutions returns the number of Steiner systems. It skips decoding, which
// matters when there are millions of them.
func countSolutions(s spec) uint64 {
	matrix, _ := buildMatrix(s)
	var total uint64
	matrix.Solve(dlx.SolutionAccepterFunc(func([][]int) bool {
		total++
		return false
	}))
	return total
}

// buildMatrix makes one column for each set of t points and one row for each
// possible block, with a 1 in the column of every t-set inside that block. An
// exact cover then says that each t-set lies in exactly one picked block, which
// is the definition of a Steiner system.
//
// It also returns the t-set of each column, as a bit mask of points. The search
// reports columns, and these masks are what turn them back into blocks.
func buildMatrix(s spec) (dlx.Matrix, []uint64) {
	points := make([]int, s.v)
	for i := range points {
		points[i] = i
	}

	columnOf := make(map[uint64]int)
	var subsets []uint64
	forEachSubset(points, s.t, func(subset []int) {
		columnOf[maskOf(subset)] = len(subsets)
		subsets = append(subsets, maskOf(subset))
	})

	matrix := dlx.NewMatrix(len(subsets))
	row := make([]int, 0, binomial(s.k, s.t))
	forEachSubset(points, s.k, func(block []int) {
		row = row[:0]
		forEachSubset(block, s.t, func(subset []int) {
			row = append(row, columnOf[maskOf(subset)])
		})
		matrix.AddRow(row...)
	})
	return matrix, subsets
}

// decode joins the t-sets of each picked row, which gives back its block.
func decode(subsets []uint64, exactCover [][]int) [][]int {
	blocks := make([][]int, len(exactCover))
	for i, row := range exactCover {
		var block uint64
		for _, column := range row {
			block |= subsets[column]
		}
		blocks[i] = pointsOf(block)
	}
	sort.Slice(blocks, func(i, j int) bool { return isBefore(blocks[i], blocks[j]) })
	return blocks
}

// forEachSubset calls visit with every subset of items of the given size, in
// dictionary order. The slice is reused between calls.
func forEachSubset(items []int, size int, visit func(subset []int)) {
	subset := make([]int, size)
	var choose func(position, next int)
	choose = func(position, next int) {
		if position == size {
			visit(subset)
			return
		}
		for i := next; i <= len(items)-size+position; i++ {
			subset[position] = items[i]
			choose(position+1, i+1)
		}
	}
	choose(0, 0)
}

func maskOf(points []int) uint64 {
	var mask uint64
	for _, point := range points {
		mask |= 1 << point
	}
	return mask
}

func pointsOf(mask uint64) []int {
	points := make([]int, 0, bits.OnesCount64(mask))
	for ; mask != 0; mask &= mask - 1 {
		points = append(points, bits.TrailingZeros64(mask))
	}
	return points
}

// isBefore reports whether block a comes before block b in dictionary order.
// Both have the same length.
func isBefore(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// binomial returns the number of ways to choose k items out of n. The largest
// value it meets here, for n up to maxPoints, fits in 64 bits at every step.
func binomial(n, k int) uint64 {
	result := uint64(1)
	for i := 0; i < k; i++ {
		result = result * uint64(n-i) / uint64(i+1)
	}
	return result
}
