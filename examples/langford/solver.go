package main

import "github.com/dimchansky/dlx"

// solve passes each Langford pairing of the numbers 1 to n to accept until
// accept returns true. A pairing is the sequence of 2n numbers. Without
// withMirrors, only one of each pairing and its reversal is reported.
func solve(n int, withMirrors bool, accept func(sequence []int) bool) {
	buildMatrix(n, withMirrors).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(n, exactCover))
	}))
}

// buildMatrix has a column for each number, then a column for each of the 2n
// positions. Each row places one number: it covers that number's column and the
// columns of the two positions it takes, which have the number's value between
// them. So every number is placed exactly once and every position is filled
// exactly once.
//
// A sequence and its reversal are different exact covers, so every solution
// comes with a mirror image. Dropping rows is the cheapest way to remove them,
// because the search never visits the branches that those rows would open. The
// number 1 at positions p and p+2 lands at positions 2n-3-p and 2n-1-p when the
// sequence is reversed. The two first positions, p and 2n-3-p, add up to an odd
// number, so they always differ and exactly one of them is below n-1. Keeping
// only those rows keeps exactly one of each mirror pair.
func buildMatrix(n int, withMirrors bool) dlx.Matrix {
	m := dlx.NewMatrix(3 * n)
	for number := 1; number <= n; number++ {
		for first := 0; first+number+1 < 2*n; first++ {
			isMirrorRow := number == 1 && first >= n-1
			if isMirrorRow && !withMirrors {
				continue
			}
			m.AddRow(number-1, n+first, n+first+number+1)
		}
	}
	return m
}

// decode relies on dlx keeping each row's columns in AddRow order: the number
// comes first, then its two positions.
func decode(n int, exactCover [][]int) []int {
	sequence := make([]int, 2*n)
	for _, row := range exactCover {
		number := row[0] + 1
		sequence[row[1]-n] = number
		sequence[row[2]-n] = number
	}
	return sequence
}
