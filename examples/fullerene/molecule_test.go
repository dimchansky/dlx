package main

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

func TestNewMolecule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		wantAtoms    int
		wantBonds    int
		wantHexagons int
	}{
		{name: "c60", wantAtoms: 60, wantBonds: 90, wantHexagons: 20},
		{name: "benzene", wantAtoms: 6, wantBonds: 6, wantHexagons: 1},
		{name: "naphthalene", wantAtoms: 10, wantBonds: 11, wantHexagons: 2},
		{name: "benzyl", wantAtoms: 7, wantBonds: 7, wantHexagons: 1},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mol, err := newMolecule(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if mol.atoms != tt.wantAtoms || len(mol.bonds) != tt.wantBonds || len(mol.hexagons) != tt.wantHexagons {
				t.Fatalf("got %d atoms, %d bonds, %d hexagons, want %d, %d, %d",
					mol.atoms, len(mol.bonds), len(mol.hexagons), tt.wantAtoms, tt.wantBonds, tt.wantHexagons)
			}

			adjacent := bondSet(mol)
			if len(adjacent) != 2*tt.wantBonds {
				t.Error("a bond is listed twice")
			}
			for _, bond := range mol.bonds {
				if bond[0] >= bond[1] {
					t.Errorf("bond %v does not list its lower atom first", bond)
				}
			}
			for _, ring := range mol.hexagons {
				for i, atom := range ring {
					if !adjacent[[2]int{atom, ring[(i+1)%6]}] {
						t.Errorf("hexagon %v has no bond %d-%d", ring, atom, ring[(i+1)%6])
					}
				}
			}
		})
	}
}

func TestNewMoleculeUnknown(t *testing.T) {
	t.Parallel()

	if _, err := newMolecule("graphite"); err == nil {
		t.Fatal("newMolecule accepted an unknown name")
	}
}

func bondSet(mol molecule) map[[2]int]bool {
	set := make(map[[2]int]bool, 2*len(mol.bonds))
	for _, bond := range mol.bonds {
		set[[2]int{bond[0], bond[1]}] = true
		set[[2]int{bond[1], bond[0]}] = true
	}
	return set
}

func TestAtomPointingAtPanicsForStrangers(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("atomPointingAt accepted two pentagons that are not neighbours")
		}
	}()
	atomPointingAt(0, 11)
}

// TestC60Shape checks the football against facts that do not depend on how it
// was built: three bonds at every atom, Euler's formula for a closed surface, and
// each atom on one pentagon and two hexagons.
func TestC60Shape(t *testing.T) {
	t.Parallel()

	mol := newC60()
	degree := make([]int, mol.atoms)
	for _, bond := range mol.bonds {
		degree[bond[0]]++
		degree[bond[1]]++
	}
	hexagonsAt := make([]int, mol.atoms)
	for _, ring := range mol.hexagons {
		for _, atom := range ring {
			hexagonsAt[atom]++
		}
	}
	for atom := 0; atom < mol.atoms; atom++ {
		if degree[atom] != 3 || hexagonsAt[atom] != 2 {
			t.Errorf("atom %d has %d bonds and %d hexagons, want 3 and 2", atom, degree[atom], hexagonsAt[atom])
		}
	}

	const pentagons = 12
	if got := mol.atoms - len(mol.bonds) + pentagons + len(mol.hexagons); got != 2 {
		t.Errorf("atoms - bonds + faces = %d, want 2", got)
	}
}

// graph is the separate model of C60 that the tests check everything against. It
// shares nothing with newC60: its atoms are points in space, a bond is a pair of
// points at the right distance, and rings are found by searching along bonds.
type graph struct {
	points [][3]float64
	adj    [][]int
}

// newCoordinateGraph builds C60 from the published coordinates of a truncated
// icosahedron: all cyclic shifts, with all signs, of three points. Its bonds have
// length 2.
func newCoordinateGraph(t *testing.T) graph {
	t.Helper()

	phi := (1 + math.Sqrt(5)) / 2
	seeds := [][3]float64{{0, 1, 3 * phi}, {1, 2 + phi, 2 * phi}, {phi, 2, 2*phi + 1}}
	var g graph
	for _, seed := range seeds {
		for shift := 0; shift < 3; shift++ {
			for signs := 0; signs < 8; signs++ {
				point := [3]float64{seed[shift%3], seed[(shift+1)%3], seed[(shift+2)%3]}
				skip := false
				for axis := range point {
					if signs>>axis&1 == 1 {
						skip = skip || point[axis] == 0 // -0 would repeat the point.
						point[axis] = -point[axis]
					}
				}
				if !skip {
					g.points = append(g.points, point)
				}
			}
		}
	}

	// Numbering from one end of the ball to the other keeps the free atoms of a
	// matching search close together, which is what makes its memo work.
	sort.Slice(g.points, func(i, j int) bool { return g.points[i][2] < g.points[j][2] })

	g.adj = make([][]int, len(g.points))
	bonds := 0
	for a := range g.points {
		for b := a + 1; b < len(g.points); b++ {
			if math.Abs(distance(g.points[a], g.points[b])-2) < 1e-9 {
				g.adj[a] = append(g.adj[a], b)
				g.adj[b] = append(g.adj[b], a)
				bonds++
			}
		}
	}
	if len(g.points) != 60 || bonds != 90 {
		t.Fatalf("coordinates gave %d atoms and %d bonds, want 60 and 90", len(g.points), bonds)
	}
	return g
}

func distance(a, b [3]float64) float64 {
	sum := 0.0
	for axis := range a {
		sum += (a[axis] - b[axis]) * (a[axis] - b[axis])
	}
	return math.Sqrt(sum)
}

// rings returns every cycle of the given length, each once, as atoms in order.
func (g graph) rings(length int) [][]int {
	var found [][]int
	seen := map[string]bool{}
	path := make([]int, 0, length)
	onPath := make([]bool, len(g.adj))
	var extend func(start, atom int)
	extend = func(start, atom int) {
		if len(path) == length {
			for _, next := range g.adj[atom] {
				if next == start {
					key := append([]int(nil), path...)
					sort.Ints(key)
					if !seen[fmt.Sprint(key)] {
						seen[fmt.Sprint(key)] = true
						found = append(found, append([]int(nil), path...))
					}
				}
			}
			return
		}
		for _, next := range g.adj[atom] {
			if next > start && !onPath[next] {
				onPath[next] = true
				path = append(path, next)
				extend(start, next)
				path = path[:len(path)-1]
				onPath[next] = false
			}
		}
	}
	for start := range g.adj {
		onPath[start] = true
		path = append(path[:0], start)
		extend(start, start)
		onPath[start] = false
	}
	return found
}

func TestCoordinateGraphRings(t *testing.T) {
	t.Parallel()

	g := newCoordinateGraph(t)
	if got := len(g.rings(5)); got != 12 {
		t.Errorf("found %d pentagons, want 12", got)
	}
	if got := len(g.rings(6)); got != 20 {
		t.Errorf("found %d hexagons, want 20", got)
	}
}

// sameShapes calls visit with every renumbering of mol's atoms that turns its
// bonds into the bonds of other, stopping when visit returns true. Because the
// next atom is always bonded to one already placed, there are only three places
// to try for it, so the search stays small.
func sameShapes(mol molecule, other [][]int, visit func(mapping []int) bool) {
	adj := make([][]int, mol.atoms)
	for _, bond := range mol.bonds {
		adj[bond[0]] = append(adj[bond[0]], bond[1])
		adj[bond[1]] = append(adj[bond[1]], bond[0])
	}
	order := []int{0}
	placed := make([]bool, mol.atoms)
	placed[0] = true
	for i := 0; i < len(order); i++ {
		for _, next := range adj[order[i]] {
			if !placed[next] {
				placed[next] = true
				order = append(order, next)
			}
		}
	}

	isBond := make([][]bool, len(other))
	for a := range other {
		isBond[a] = make([]bool, len(other))
		for _, b := range other[a] {
			isBond[a][b] = true
		}
	}

	mapping := make([]int, mol.atoms)
	taken := make([]bool, len(other))
	for atom := range mapping {
		mapping[atom] = -1
	}
	fits := func(atom, image int) bool {
		for _, neighbour := range adj[atom] {
			if mapping[neighbour] >= 0 && !isBond[image][mapping[neighbour]] {
				return false
			}
		}
		return !taken[image]
	}
	var place func(i int) bool
	place = func(i int) bool {
		if i == len(order) {
			return visit(mapping)
		}
		atom := order[i]
		for image := range other {
			if fits(atom, image) {
				mapping[atom], taken[image] = image, true
				stopped := place(i + 1)
				mapping[atom], taken[image] = -1, false
				if stopped {
					return true
				}
			}
		}
		return false
	}
	place(0)
}

// TestC60MatchesCoordinates proves that the molecule used by the solver is the
// football itself: renumbering its atoms gives the coordinate graph exactly, and
// its hexagons land exactly on the rings found in that graph.
func TestC60MatchesCoordinates(t *testing.T) {
	t.Parallel()

	mol := newC60()
	g := newCoordinateGraph(t)
	wantRings := map[string]bool{}
	for _, ring := range g.rings(6) {
		wantRings[atomSetKey(ring)] = true
	}

	matched := false
	sameShapes(mol, g.adj, func(mapping []int) bool {
		for _, ring := range mol.hexagons {
			image := make([]int, len(ring))
			for i, atom := range ring {
				image[i] = mapping[atom]
			}
			if !wantRings[atomSetKey(image)] {
				t.Errorf("hexagon %v lands on %v, which is not a ring", ring, image)
			}
		}
		matched = true
		return true
	})
	if !matched {
		t.Fatal("the molecule and the coordinate graph have different shapes")
	}
}

func atomSetKey(atoms []int) string {
	sorted := append([]int(nil), atoms...)
	sort.Ints(sorted)
	return fmt.Sprint(sorted)
}

// TestC60Symmetries counts the renumberings of the football onto itself. The 120
// are the 60 rotations of an icosahedron and their mirror images.
func TestC60Symmetries(t *testing.T) {
	t.Parallel()

	if got := len(symmetries(newC60())); got != 120 {
		t.Fatalf("got %d symmetries, want 120", got)
	}
}

func symmetries(mol molecule) [][]int {
	adj := make([][]int, mol.atoms)
	for _, bond := range mol.bonds {
		adj[bond[0]] = append(adj[bond[0]], bond[1])
		adj[bond[1]] = append(adj[bond[1]], bond[0])
	}
	var found [][]int
	sameShapes(mol, adj, func(mapping []int) bool {
		found = append(found, append([]int(nil), mapping...))
		return false
	})
	return found
}

func TestPentagonNeighboursAreConsistent(t *testing.T) {
	t.Parallel()

	for p, around := range pentagonNeighbours {
		for i, q := range around {
			if atom := atomPointingAt(q, p); atom/5 != q {
				t.Errorf("atomPointingAt(%d, %d) = %d, not on pentagon %d", q, p, atom, q)
			}
			next := around[(i+1)%5]
			if !contains(pentagonNeighbours[q][:], next) {
				t.Errorf("pentagons %d and %d follow each other around %d but are not neighbours", q, next, p)
			}
		}
	}
}

func contains(list []int, value int) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
