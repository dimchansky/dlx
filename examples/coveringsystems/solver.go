package main

import (
	"sort"

	"github.com/dimchansky/dlx"
)

// class is the set of integers that leave residue when divided by modulus.
type class struct {
	residue, modulus int
}

// solve passes each exact covering system to accept until accept returns true.
// Every modulus in a system divides n, and a system lists its classes by modulus
// and then by residue. With distinct, only systems of at least two classes with
// all moduli different are reported.
func solve(n int, distinct bool, accept func(system []class) bool) {
	buildMatrix(n, distinct).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(n, exactCover))
	}))
}

// buildMatrix has a column for each of the numbers 0 to n-1. When every modulus
// divides n, adding n to an integer doesn't change which classes it belongs to,
// so these n numbers stand for all the integers. Each row is one class and
// covers the numbers in that class.
//
// With distinct, the class with modulus 1 goes, and each remaining modulus gets
// a column, which each of its classes also covers. So no two picked classes can
// share a modulus. A modulus may also go unused, so it gets a slack row that
// covers its column alone.
func buildMatrix(n int, distinct bool) dlx.Matrix {
	moduli := divisors(n)
	nColumns := n
	if distinct {
		moduli = moduli[1:]
		nColumns += len(moduli)
	}

	m := dlx.NewMatrix(nColumns)
	for i, modulus := range moduli {
		for residue := 0; residue < modulus; residue++ {
			row := make([]int, 0, n/modulus+1)
			for number := residue; number < n; number += modulus {
				row = append(row, number)
			}
			if distinct {
				row = append(row, n+i)
			}
			m.AddRow(row...)
		}
		if distinct {
			m.AddRow(n + i)
		}
	}
	return m
}

// divisors returns the divisors of n in increasing order.
func divisors(n int) []int {
	var result []int
	for d := 1; d <= n; d++ {
		if n%d == 0 {
			result = append(result, d)
		}
	}
	return result
}

// decode relies on dlx keeping each row's columns in AddRow order: the numbers
// of a class come first, smallest first, so the first column is the residue. A
// slack row is the only row that starts at or after column n. A class with
// modulus m holds n/m of the numbers, which gives m back.
func decode(n int, exactCover [][]int) []class {
	system := make([]class, 0, len(exactCover))
	for _, row := range exactCover {
		if row[0] >= n {
			continue
		}
		size := 0
		for _, column := range row {
			if column < n {
				size++
			}
		}
		system = append(system, class{residue: row[0], modulus: n / size})
	}
	sort.Slice(system, func(i, j int) bool {
		if system[i].modulus != system[j].modulus {
			return system[i].modulus < system[j].modulus
		}
		return system[i].residue < system[j].residue
	})
	return system
}
