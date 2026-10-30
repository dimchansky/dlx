package main

import (
	"fmt"
	"strings"
)

// molecule is a carbon skeleton: atoms numbered from 0, the bonds that join
// them, and the hexagonal rings. Each bond lists its lower atom first.
type molecule struct {
	atoms    int
	bonds    [][2]int
	hexagons [][]int
}

// moleculeNames lists the names newMolecule accepts.
var moleculeNames = []string{"c60", "benzene", "naphthalene", "benzyl"}

// newMolecule returns the named molecule.
func newMolecule(name string) (molecule, error) {
	switch name {
	case "c60":
		return newC60(), nil
	case "benzene":
		return newBenzene(), nil
	case "naphthalene":
		return newNaphthalene(), nil
	case "benzyl":
		return newBenzyl(), nil
	}
	return molecule{}, fmt.Errorf("unknown molecule %q: want one of %s", name, strings.Join(moleculeNames, ", "))
}

// newBond returns the bond between two atoms, lower atom first.
func newBond(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

// newBenzene is a ring of six atoms.
func newBenzene() molecule {
	mol := molecule{atoms: 6, hexagons: [][]int{{0, 1, 2, 3, 4, 5}}}
	for a := 0; a < 6; a++ {
		mol.bonds = append(mol.bonds, newBond(a, (a+1)%6))
	}
	return mol
}

// newBenzyl is benzene with a seventh atom bonded to atom 0. The odd atom has
// no partner, so this molecule has no Kekule structure.
func newBenzyl() molecule {
	mol := newBenzene()
	mol.atoms = 7
	mol.bonds = append(mol.bonds, newBond(0, 6))
	return mol
}

// newNaphthalene is a ring of ten atoms with one extra bond across the middle,
// between atoms 4 and 9, which makes two hexagons side by side.
func newNaphthalene() molecule {
	mol := molecule{atoms: 10, hexagons: [][]int{{9, 0, 1, 2, 3, 4}, {4, 5, 6, 7, 8, 9}}}
	for a := 0; a < 10; a++ {
		mol.bonds = append(mol.bonds, newBond(a, (a+1)%10))
	}
	mol.bonds = append(mol.bonds, newBond(4, 9))
	return mol
}

// pentagonNeighbours describes how the 12 pentagons of a football sit next to
// each other: row p lists the five pentagons around pentagon p, in order around
// it. Two pentagons never touch, so "next to" means that one bond joins them.
var pentagonNeighbours = [12][5]int{
	{1, 2, 3, 4, 5},
	{0, 2, 6, 10, 5},
	{0, 3, 7, 6, 1},
	{0, 4, 8, 7, 2},
	{0, 5, 9, 8, 3},
	{0, 1, 10, 9, 4},
	{1, 2, 7, 11, 10},
	{2, 3, 8, 11, 6},
	{3, 4, 9, 11, 7},
	{4, 5, 10, 11, 8},
	{5, 1, 6, 11, 9},
	{6, 7, 8, 9, 10},
}

// newC60 builds the football shape from pentagonNeighbours. Every atom belongs
// to exactly one pentagon, so pentagon p holds atoms 5p to 5p+4, with atom 5p+i
// pointing at the pentagon named in pentagonNeighbours[p][i].
//
//   - Neighbouring atoms of a pentagon are bonded: 60 bonds.
//   - Two atoms pointing at each other from two pentagons are bonded: 30 bonds.
//   - Three pentagons that all sit next to each other surround a hexagon: 20
//     hexagons.
func newC60() molecule {
	mol := molecule{atoms: 60}
	for p, around := range pentagonNeighbours {
		for i, q := range around {
			next := around[(i+1)%5]
			mol.bonds = append(mol.bonds, newBond(5*p+i, 5*p+(i+1)%5))
			if p < q {
				mol.bonds = append(mol.bonds, newBond(atomPointingAt(p, q), atomPointingAt(q, p)))
			}
			if p < q && p < next { // Each hexagon once, from its lowest pentagon.
				mol.hexagons = append(mol.hexagons, []int{
					atomPointingAt(p, q), atomPointingAt(q, p),
					atomPointingAt(q, next), atomPointingAt(next, q),
					atomPointingAt(next, p), atomPointingAt(p, next),
				})
			}
		}
	}
	return mol
}

// atomPointingAt returns the atom of pentagon p that points at pentagon q,
// which must be next to p.
func atomPointingAt(p, q int) int {
	for i, neighbour := range pentagonNeighbours[p] {
		if neighbour == q {
			return 5*p + i
		}
	}
	panic(fmt.Sprintf("fullerene: pentagons %d and %d are not neighbours", p, q))
}
