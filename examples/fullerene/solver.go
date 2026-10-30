package main

import (
	"math/bits"
	"sort"

	"github.com/dimchansky/dlx"
)

// cover is one way of giving every atom exactly one role: it sits in one of the
// double bonds, or in one of the sextets (hexagons with six shared electrons).
type cover struct {
	sextets     uint32 // Bit h is set if hexagon h is a sextet.
	doubleBonds [][2]int
}

// sextetList returns the numbers of the hexagons in the set, in rising order.
func sextetList(set uint32) []int {
	list := []int{}
	for h := 0; h < 32; h++ {
		if set>>h&1 == 1 {
			list = append(list, h)
		}
	}
	return list
}

// solve passes each cover of mol to accept until accept returns true. Without
// withSextets it uses bonds alone, so each cover is a Kekule structure.
func solve(mol molecule, withSextets bool, accept func(cover) bool) {
	hexagonByRow := make(map[[6]int]int, len(mol.hexagons))
	for h, ring := range mol.hexagons {
		hexagonByRow[hexagonKey(ring)] = h
	}
	buildMatrix(mol, withSextets).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(exactCover, hexagonByRow))
	}))
}

// buildMatrix has one column for each atom. Each bond is a row covering its two
// atoms. With withSextets, each hexagon is also a row, covering its six atoms.
// Every atom is then in exactly one double bond or sextet.
func buildMatrix(mol molecule, withSextets bool) dlx.Matrix {
	m := dlx.NewMatrix(mol.atoms)
	for _, bond := range mol.bonds {
		m.AddRow(bond[0], bond[1])
	}
	if withSextets {
		for _, ring := range mol.hexagons {
			m.AddRow(ring...)
		}
	}
	return m
}

// hexagonKey turns the columns of a hexagon row into a map key. The order is
// safe to rely on, because dlx keeps each row's columns in AddRow order.
func hexagonKey(ring []int) [6]int {
	var key [6]int
	copy(key[:], ring)
	return key
}

// decode tells the rows apart by length: two columns are a bond, six a hexagon.
func decode(exactCover [][]int, hexagonByRow map[[6]int]int) cover {
	var c cover
	for _, row := range exactCover {
		if len(row) == 2 {
			c.doubleBonds = append(c.doubleBonds, [2]int{row[0], row[1]})
			continue
		}
		c.sextets |= 1 << hexagonByRow[hexagonKey(row)]
	}
	sort.Slice(c.doubleBonds, func(i, j int) bool {
		return c.doubleBonds[i][0] < c.doubleBonds[j][0]
	})
	return c
}

// clarLine summarises the covers that have one size of sextet set.
type clarLine struct {
	sextets    int // How many sextets each cover in this line has.
	covers     int // Covers with that many sextets.
	sets       int // Different sets of hexagons among those covers.
	structures int // Sets that cannot take one more sextet: Clar structures.
}

// clarReport is the result of going through every cover of a molecule.
type clarReport struct {
	lines   []clarLine // In rising order of sextets, sizes with no cover left out.
	largest []cover    // Every cover with the most sextets.
}

// analyseClar goes through every cover of mol, grouping them by set of sextets.
// A set that one more hexagon can join is not a Clar structure. Checking one
// more is enough: if a set can grow by two hexagons, it can grow by one of them,
// because the other hexagon's six atoms can pair up around their ring instead.
func analyseClar(mol molecule) clarReport {
	coversBySet := map[uint32]int{}
	var largest []cover
	mostSextets := 0
	solve(mol, true, func(c cover) bool {
		coversBySet[c.sextets]++
		size := bits.OnesCount32(c.sextets)
		if size > mostSextets {
			mostSextets, largest = size, nil
		}
		if size == mostSextets {
			largest = append(largest, c)
		}
		return false
	})

	bySize := make([]clarLine, len(mol.hexagons)+1)
	for set, covers := range coversBySet {
		line := &bySize[bits.OnesCount32(set)]
		line.covers += covers
		line.sets++
		if !canGrow(set, len(mol.hexagons), coversBySet) {
			line.structures++
		}
	}

	report := clarReport{largest: largest}
	for size, line := range bySize {
		if line.covers > 0 {
			line.sextets = size
			report.lines = append(report.lines, line)
		}
	}
	return report
}

// canGrow reports whether one more of the hexagons can join the set.
func canGrow(set uint32, hexagons int, coversBySet map[uint32]int) bool {
	for h := 0; h < hexagons; h++ {
		if coversBySet[set|1<<h] > 0 && set>>h&1 == 0 {
			return true
		}
	}
	return false
}
