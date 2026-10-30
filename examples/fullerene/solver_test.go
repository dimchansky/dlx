package main

import (
	"bytes"
	"math"
	"math/big"
	"math/bits"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Totals for C60. The first line of each slice is 0 sextets, so wantCovers[0] is
// the number of Kekule structures. The last three of wantStructures are the
// published Clar polynomial 5x^8 + 280x^7 + 10x^6.
var (
	wantC60Covers     = []int{12500, 41580, 67650, 66900, 42000, 16352, 3620, 360, 5}
	wantC60Sets       = []int{1, 20, 160, 660, 1510, 1912, 1240, 320, 5}
	wantC60Structures = []int{0, 0, 0, 0, 0, 0, 10, 280, 5}
)

func TestKekuleTotals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int
		slow bool
	}{
		{name: "benzene", want: 2},
		{name: "naphthalene", want: 3},
		{name: "benzyl", want: 0},
		{name: "c60", want: 12500, slow: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow with -race")
			}

			mol, err := newMolecule(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			adjacent := bondSet(mol)
			got := 0
			solve(mol, false, func(c cover) bool {
				got++
				checkCover(t, mol, adjacent, c)
				return false
			})
			if got != tt.want {
				t.Errorf("got %d structures, want %d", got, tt.want)
			}
		})
	}
}

// checkCover confirms that every atom has exactly one role, using only real bonds
// and real hexagons.
func checkCover(t *testing.T, mol molecule, adjacent map[[2]int]bool, c cover) {
	t.Helper()

	roles := make([]int, mol.atoms)
	for _, bond := range c.doubleBonds {
		if !adjacent[bond] {
			t.Errorf("double bond %v is not a bond", bond)
		}
		roles[bond[0]]++
		roles[bond[1]]++
	}
	for _, h := range sextetList(c.sextets) {
		for _, atom := range mol.hexagons[h] {
			roles[atom]++
		}
	}
	for atom, count := range roles {
		if count != 1 {
			t.Errorf("atom %d has %d roles", atom, count)
		}
	}
}

func TestSolveStopsWhenAccepted(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(newC60(), false, func(cover) bool {
		calls++
		return calls == 3
	})
	if calls != 3 {
		t.Fatalf("accept was called %d times, want 3", calls)
	}
}

func TestSextetList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		set  uint32
		want []int
	}{
		{name: "empty", set: 0, want: []int{}},
		{name: "first", set: 1, want: []int{0}},
		{name: "scattered", set: 1<<3 | 1<<19, want: []int{3, 19}},
		{name: "last bit", set: 1 << 31, want: []int{31}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sextetList(tt.set); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyseClar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		wantLines   []clarLine
		wantLargest int
		wantTotal   string
		slow        bool
	}{
		{
			name: "benzene",
			wantLines: []clarLine{
				{sextets: 0, covers: 2, sets: 1},
				{sextets: 1, covers: 1, sets: 1, structures: 1},
			},
			wantLargest: 1,
			wantTotal:   "  Total       3      2           1\n",
		},
		{
			name: "naphthalene",
			wantLines: []clarLine{
				{sextets: 0, covers: 3, sets: 1},
				{sextets: 1, covers: 2, sets: 2, structures: 2},
			},
			wantLargest: 2,
			wantTotal:   "  Total       5      3           2\n",
		},
		{name: "benzyl"},
		{
			name:        "c60",
			wantLines:   c60Lines(),
			wantLargest: 5,
			wantTotal:   "  Total  250967   5828         295\n",
			slow:        true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow with -race")
			}

			mol, err := newMolecule(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			got := analyseClar(mol)
			if !reflect.DeepEqual(got.lines, tt.wantLines) {
				t.Errorf("lines = %+v, want %+v", got.lines, tt.wantLines)
			}
			if len(got.largest) != tt.wantLargest {
				t.Errorf("got %d largest covers, want %d", len(got.largest), tt.wantLargest)
			}
			for _, c := range got.largest {
				checkCover(t, mol, bondSet(mol), c)
			}

			var out bytes.Buffer
			if len(got.lines) > 0 {
				if err := writeClarReport(&out, got); err != nil {
					t.Fatal(err)
				}
			}
			if !strings.Contains(out.String(), tt.wantTotal) {
				t.Errorf("output lacks %q:\n%s", tt.wantTotal, out.String())
			}
		})
	}
}

func c60Lines() []clarLine {
	var lines []clarLine
	for k := range wantC60Covers {
		lines = append(lines, clarLine{
			sextets: k, covers: wantC60Covers[k], sets: wantC60Sets[k], structures: wantC60Structures[k],
		})
	}
	return lines
}

// matchingCounter counts perfect matchings of the atoms a mask leaves free. It
// pairs off the lowest free atom in each possible way, and remembers the answer
// for each mask. It uses no exact cover and no data from the solver.
type matchingCounter struct {
	adj  [][]int
	memo map[uint64]int
}

func (m *matchingCounter) count(used uint64) int {
	if used == 1<<len(m.adj)-1 {
		return 1
	}
	if known, ok := m.memo[used]; ok {
		return known
	}
	atom := bits.TrailingZeros64(^used)
	total := 0
	for _, partner := range m.adj[atom] {
		if used>>partner&1 == 0 {
			total += m.count(used | 1<<atom | 1<<partner)
		}
	}
	m.memo[used] = total
	return total
}

// TestIndependentTable rebuilds the whole C60 table from the coordinate graph,
// by choosing hexagons directly and counting the matchings of what is left.
func TestIndependentTable(t *testing.T) {
	t.Parallel()

	g := newCoordinateGraph(t)
	hexagons := g.rings(6)
	masks := make([]uint64, len(hexagons))
	for h, ring := range hexagons {
		for _, atom := range ring {
			masks[h] |= 1 << atom
		}
	}

	counter := matchingCounter{adj: g.adj, memo: map[uint64]int{}}
	covers := make([]int, 9)
	sets := make([]int, 9)
	var choose func(next, size int, used uint64)
	choose = func(next, size int, used uint64) {
		if next == len(hexagons) {
			if matchings := counter.count(used); matchings > 0 {
				covers[size] += matchings
				sets[size]++
			}
			return
		}
		choose(next+1, size, used)
		if used&masks[next] == 0 {
			choose(next+1, size+1, used|masks[next])
		}
	}
	choose(0, 0, 0)

	if !reflect.DeepEqual(covers, wantC60Covers) {
		t.Errorf("covers = %v, want %v", covers, wantC60Covers)
	}
	if !reflect.DeepEqual(sets, wantC60Sets) {
		t.Errorf("sets = %v, want %v", sets, wantC60Sets)
	}
}

// touchMasks returns, for each hexagon, the set of hexagons sharing an atom.
func touchMasks(hexagons [][]int) []uint32 {
	touching := make([]uint32, len(hexagons))
	for a := range hexagons {
		for b := range hexagons {
			if a != b && atomSetsMeet(hexagons[a], hexagons[b]) {
				touching[a] |= 1 << b
			}
		}
	}
	return touching
}

func atomSetsMeet(a, b []int) bool {
	for _, atom := range a {
		if contains(b, atom) {
			return true
		}
	}
	return false
}

// nonTouchingSets calls visit with every set of hexagons no two of which touch.
func nonTouchingSets(touching []uint32, visit func(set, blocked uint32)) {
	var choose func(next int, set, blocked uint32)
	choose = func(next int, set, blocked uint32) {
		if next == len(touching) {
			visit(set, blocked)
			return
		}
		choose(next+1, set, blocked)
		if blocked>>next&1 == 0 {
			choose(next+1, set|1<<next, blocked|touching[next])
		}
	}
	choose(0, 0, 0)
}

// TestClarPolynomialFromTouching reaches the published polynomial without any
// matching at all: its coefficients are the sets of non-touching hexagons that
// have no room for one more. This fits the "maximal independent set polynomial"
// named in the abstract of Shiu, Lam and Zhang.
func TestClarPolynomialFromTouching(t *testing.T) {
	t.Parallel()

	touching := touchMasks(newCoordinateGraph(t).rings(6))
	for h, mask := range touching {
		if got := bits.OnesCount32(mask); got != 3 {
			t.Errorf("hexagon %d touches %d hexagons, want 3", h, got)
		}
	}

	sets := make([]int, 9)
	structures := make([]int, 9)
	nonTouchingSets(touching, func(set, blocked uint32) {
		size := bits.OnesCount32(set)
		sets[size]++
		if set|blocked == 1<<len(touching)-1 {
			structures[size]++
		}
	})
	if !reflect.DeepEqual(sets, wantC60Sets) {
		t.Errorf("sets = %v, want %v", sets, wantC60Sets)
	}
	if !reflect.DeepEqual(structures, wantC60Structures) {
		t.Errorf("structures = %v, want %v", structures, wantC60Structures)
	}
}

// TestDeterminant counts Kekule structures with no search of any kind. Direct the
// bonds so that going around every face meets an odd number of bonds pointing
// the way you walk. Kasteleyn proved that the matrix of those directions then has
// a determinant equal to the square of the number of perfect matchings.
func TestDeterminant(t *testing.T) {
	t.Parallel()

	g := newCoordinateGraph(t)
	faces := append(g.rings(5), g.rings(6)...)
	for f := range faces {
		g.turnOutwards(faces[f])
	}
	direction := directBonds(t, faces)

	matrix := make([][]*big.Rat, len(g.adj))
	for a := range matrix {
		matrix[a] = make([]*big.Rat, len(g.adj))
		for b := range matrix[a] {
			matrix[a][b] = big.NewRat(direction[[2]int{a, b}], 1)
		}
	}
	want := big.NewRat(12500*12500, 1)
	if got := determinant(matrix); got.Cmp(want) != 0 {
		t.Fatalf("determinant = %v, want %v", got, want)
	}
}

// turnOutwards reverses the ring if needed, so that all rings run the same way
// around when seen from outside the football.
func (g graph) turnOutwards(ring []int) {
	a, b, c := g.points[ring[0]], g.points[ring[1]], g.points[ring[2]]
	var u, v [3]float64
	for axis := range a {
		u[axis], v[axis] = b[axis]-a[axis], c[axis]-a[axis]
	}
	normal := [3]float64{u[1]*v[2] - u[2]*v[1], u[2]*v[0] - u[0]*v[2], u[0]*v[1] - u[1]*v[0]}
	if normal[0]*a[0]+normal[1]*a[1]+normal[2]*a[2] < 0 {
		for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
			ring[i], ring[j] = ring[j], ring[i]
		}
	}
}

// directBonds gives every bond a direction, stored as +1 and -1 entries, so that
// each face has an odd number of bonds pointing along its ring. It visits faces
// in the reverse of a breadth-first walk, so every face but the last still has a
// free bond, the one shared with the face that reached it. The last face has no
// choice left, and comes out right because the number of atoms is even.
func directBonds(t *testing.T, faces [][]int) map[[2]int]int64 {
	t.Helper()

	facesAtBond := map[[2]int][]int{}
	for f, ring := range faces {
		for i, atom := range ring {
			key := newBond(atom, ring[(i+1)%len(ring)])
			facesAtBond[key] = append(facesAtBond[key], f)
		}
	}
	order := []int{0}
	reached := make([]bool, len(faces))
	reached[0] = true
	for i := 0; i < len(order); i++ {
		ring := faces[order[i]]
		for j, atom := range ring {
			for _, f := range facesAtBond[newBond(atom, ring[(j+1)%len(ring)])] {
				if !reached[f] {
					reached[f] = true
					order = append(order, f)
				}
			}
		}
	}

	direction := map[[2]int]int64{}
	for i := len(order) - 1; i >= 0; i-- {
		ring := faces[order[i]]
		odd := false
		last := [2]int{-1, -1}
		for j, atom := range ring {
			step := [2]int{atom, ring[(j+1)%len(ring)]}
			switch direction[step] {
			case 0:
				if last[0] >= 0 {
					direction[last] = 1
					direction[[2]int{last[1], last[0]}] = -1
					odd = !odd
				}
				last = step
			case 1:
				odd = !odd
			}
		}
		sign := int64(1)
		if odd {
			sign = -1
		}
		if last[0] >= 0 {
			direction[last] = sign
			direction[[2]int{last[1], last[0]}] = -sign
		} else if !odd {
			t.Fatalf("face %v has an even number of forward bonds", ring)
		}
	}
	return direction
}

// determinant uses exact fractions, so no rounding can flatter the result.
func determinant(matrix [][]*big.Rat) *big.Rat {
	result := big.NewRat(1, 1)
	for col := range matrix {
		pivot := col
		for pivot < len(matrix) && matrix[pivot][col].Sign() == 0 {
			pivot++
		}
		if pivot == len(matrix) {
			return new(big.Rat)
		}
		if pivot != col {
			matrix[pivot], matrix[col] = matrix[col], matrix[pivot]
			result.Neg(result)
		}
		result.Mul(result, matrix[col][col])
		for row := col + 1; row < len(matrix); row++ {
			if matrix[row][col].Sign() == 0 {
				continue
			}
			factor := new(big.Rat).Quo(matrix[row][col], matrix[col][col])
			for c := col; c < len(matrix); c++ {
				matrix[row][c].Sub(matrix[row][c], new(big.Rat).Mul(factor, matrix[col][c]))
			}
		}
	}
	return result
}

// TestLargestSetsAreCubes checks what the five largest sets look like in space:
// each is eight hexagon centres at the corners of a cube, and each hexagon is in
// exactly two of the five sets.
func TestLargestSetsAreCubes(t *testing.T) {
	t.Parallel()

	g := newCoordinateGraph(t)
	hexagons := g.rings(6)
	var largest []uint32
	nonTouchingSets(touchMasks(hexagons), func(set, _ uint32) {
		if bits.OnesCount32(set) == 8 {
			largest = append(largest, set)
		}
	})
	if len(largest) != 5 {
		t.Fatalf("got %d largest sets, want 5", len(largest))
	}

	timesUsed := make([]int, len(hexagons))
	for _, set := range largest {
		var centres [][3]float64
		for _, h := range sextetList(set) {
			timesUsed[h]++
			centres = append(centres, g.centre(hexagons[h]))
		}
		checkCube(t, centres)
	}
	for h, times := range timesUsed {
		if times != 2 {
			t.Errorf("hexagon %d is in %d sets, want 2", h, times)
		}
	}
}

func (g graph) centre(ring []int) [3]float64 {
	var sum [3]float64
	for _, atom := range ring {
		for axis := range sum {
			sum[axis] += g.points[atom][axis] / float64(len(ring))
		}
	}
	return sum
}

// checkCube confirms the 28 distances between 8 corners: 12 edges of some length
// s, 12 face diagonals of s times root 2, and 4 long diagonals of s times root 3.
func checkCube(t *testing.T, corners [][3]float64) {
	t.Helper()

	var lengths []float64
	for a := range corners {
		for b := a + 1; b < len(corners); b++ {
			lengths = append(lengths, distance(corners[a], corners[b]))
		}
	}
	sort.Float64s(lengths)
	edge := lengths[0]
	for i, length := range lengths {
		want := edge * math.Sqrt2
		switch {
		case i < 12:
			want = edge
		case i >= 24:
			want = edge * math.Sqrt(3)
		}
		if math.Abs(length-want) > 1e-9 {
			t.Fatalf("distance %d is %v, want %v", i, length, want)
		}
	}
}

// TestNoDoubleBondOnPentagons finds the one structure that real C60 resembles:
// without the bonds along pentagons, the 30 bonds between pentagons are all that
// is left. They form exactly one structure, with three on each hexagon.
func TestNoDoubleBondOnPentagons(t *testing.T) {
	t.Parallel()

	mol := newC60()
	var between [][2]int
	for _, bond := range mol.bonds {
		if bond[0]/5 != bond[1]/5 { // Different pentagons.
			between = append(between, bond)
		}
	}
	rings := mol.hexagons
	mol.bonds = between

	found := 0
	solve(mol, false, func(c cover) bool {
		found++
		for h, ring := range rings {
			if got := doubleBondsOn(ring, c); got != 3 {
				t.Errorf("hexagon %d has %d double bonds, want 3", h, got)
			}
		}
		return false
	})
	if found != 1 {
		t.Fatalf("got %d structures, want 1", found)
	}
}

func doubleBondsOn(ring []int, c cover) int {
	count := 0
	for i, atom := range ring {
		bond := newBond(atom, ring[(i+1)%len(ring)])
		for _, double := range c.doubleBonds {
			if double == bond {
				count++
			}
		}
	}
	return count
}

// TestDifferentUpToSymmetry counts structures that differ by more than a turn or
// a reflection of the football. The answer matches the published 158.
func TestDifferentUpToSymmetry(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow with -race")
	}

	mol := newC60()
	bondNumber := make([][]int, mol.atoms)
	for a := range bondNumber {
		bondNumber[a] = make([]int, mol.atoms)
	}
	for i, bond := range mol.bonds {
		bondNumber[bond[0]][bond[1]], bondNumber[bond[1]][bond[0]] = i, i
	}
	turns := symmetries(mol)

	type bondSetBits [2]uint64
	less := func(a, b bondSetBits) bool { return a[0] < b[0] || (a[0] == b[0] && a[1] < b[1]) }
	classes := map[bondSetBits]bool{}
	solve(mol, false, func(c cover) bool {
		var smallest bondSetBits
		for i, turn := range turns {
			var image bondSetBits
			for _, bond := range c.doubleBonds {
				number := bondNumber[turn[bond[0]]][turn[bond[1]]]
				image[number/64] |= 1 << (number % 64)
			}
			if i == 0 || less(image, smallest) {
				smallest = image
			}
		}
		classes[smallest] = true
		return false
	})
	if len(classes) != 158 {
		t.Fatalf("got %d classes, want 158", len(classes))
	}
}

func TestFormatBonds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bonds [][2]int
		want  string
	}{
		{name: "none", want: "none"},
		{name: "one", bonds: [][2]int{{0, 1}}, want: "0-1"},
		{name: "two", bonds: [][2]int{{0, 1}, {2, 9}}, want: "0-1 2-9"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatBonds(tt.bonds); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
