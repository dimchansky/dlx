package main

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The published totals. The disk has the same kites whichever way the hat is
// mirrored, so these do not depend on which of the two shapes is called
// unreflected.
var coverTotals = []struct {
	name        string
	radius      int
	withMirrors bool
	want        int
	slow        bool
	// hats and reflected give the fewest and most hats in any cover, when set.
	hats, reflected [2]int
}{
	{name: "rotations radius 2", radius: 2, want: 28},
	{name: "rotations radius 3", radius: 3, want: 46},
	{name: "rotations radius 4", radius: 4, want: 76},
	{name: "rotations radius 5", radius: 5, want: 14},
	{name: "rotations radius 6", radius: 6, want: 0},
	{name: "rotations radius 7", radius: 7, want: 0},
	{name: "rotations radius 8", radius: 8, want: 0},
	{name: "mirrors radius 2", radius: 2, withMirrors: true, want: 80},
	{name: "mirrors radius 3", radius: 3, withMirrors: true, want: 716},
	{name: "mirrors radius 4", radius: 4, withMirrors: true, want: 4228, slow: true},
	{name: "mirrors radius 5", radius: 5, withMirrors: true, want: 3676, slow: true},
	{
		name: "mirrors radius 6", radius: 6, withMirrors: true, want: 26844, slow: true,
		hats: [2]int{13, 19}, reflected: [2]int{1, 16},
	},
}

// checkedCovers is how many covers of each total get a full check for gaps and
// overlaps. Checking all of them would take seconds.
const checkedCovers = 200

func TestHatShape(t *testing.T) {
	t.Parallel()

	seen := make(map[kite]bool)
	for _, k := range hatShape {
		seen[k] = true
		for _, corner := range k.corners() {
			// Pulling a corner towards the centre of its kite puts it strictly
			// inside or strictly outside the outline.
			pulled := point{28*corner.a + k.centre().a, 28*corner.b + k.centre().b}
			if !insideOutline(hatOutline[:], 32, pulled) {
				t.Errorf("kite %v has a corner outside the outline", k)
			}
		}
	}
	if len(seen) != hatKites {
		t.Errorf("hatShape has %d distinct kites, want %d", len(seen), hatKites)
	}

	// Eight different kites inside the outline fill it if the areas match. One
	// kite has 4 of the units that doubledArea counts.
	if got, want := doubledArea(hatOutline[:]), 4*hatKites; got != want {
		t.Errorf("outline has doubled area %d, want %d", got, want)
	}
}

// TestHatOutline checks what "An aperiodic monotile" says about the shape: 13
// sides, of lengths 1, 2 and the square root of 3.
func TestHatOutline(t *testing.T) {
	t.Parallel()

	n := len(hatOutline)
	if n != 13 {
		t.Fatalf("outline has %d corners, want 13", n)
	}
	lengths := make(map[int]int)
	for i, corner := range hatOutline {
		next, after := hatOutline[(i+1)%n], hatOutline[(i+2)%n]
		side := point{next.a - corner.a, next.b - corner.b}
		lengths[norm(side)]++
		onward := point{after.a - next.a, after.b - next.b}
		if side.a*onward.b == side.b*onward.a {
			t.Errorf("corners %d to %d lie on one line", i, i+2)
		}
	}
	want := map[int]int{1: 6, 3: 6, 4: 1}
	for length, count := range want {
		if lengths[length] != count {
			t.Errorf("%d sides with squared length %d, want %d", lengths[length], length, count)
		}
	}
}

// TestHatOrientations checks the paper's remark that the hat uses four of the
// six kite orientations once each, and two opposite ones twice each.
func TestHatOrientations(t *testing.T) {
	t.Parallel()

	var used [turns]int
	for _, k := range hatShape {
		used[k.dir]++
	}
	for dir := 0; dir < turns; dir++ {
		want := 1
		if dir%3 == 1 {
			want = 2
		}
		if used[dir] != want {
			t.Errorf("direction %d is used %d times, want %d", dir, used[dir], want)
		}
	}
}

// TestPlacementKitesMatchOutline shows that the moves on kites and the moves on
// outline corners agree, for all twelve orientations.
func TestPlacementKitesMatchOutline(t *testing.T) {
	t.Parallel()

	for _, mirrored := range []bool{false, true} {
		for steps := 0; steps < turns; steps++ {
			hat := placement{mirrored: mirrored, turns: steps, q: 3, r: -2}
			outline := hat.outline()
			for _, k := range hat.kites() {
				centre := k.centre()
				if !insideOutline(outline, 4, centre) {
					t.Errorf("%+v: kite %v is outside the outline", hat, k)
				}
			}
		}
	}
}

func TestNewPuzzle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		radius       int
		withMirrors  bool
		wantDisk     int
		wantColumns  int
		wantHatsRows int
	}{
		{name: "smallest disk", radius: 2, wantDisk: 6, wantColumns: 36, wantHatsRows: 18},
		{name: "last disk with rotations", radius: 5, wantDisk: 48, wantColumns: 120, wantHatsRows: 78},
		{name: "first disk that needs mirrors", radius: 6, wantDisk: 72, wantColumns: 174, wantHatsRows: 120},
		{name: "with mirrors", radius: 6, withMirrors: true, wantDisk: 72, wantColumns: 186, wantHatsRows: 240},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := newPuzzle(tt.radius, tt.withMirrors)
			if p.disk != tt.wantDisk {
				t.Errorf("disk has %d kites, want %d", p.disk, tt.wantDisk)
			}
			if len(p.kites) != tt.wantColumns {
				t.Errorf("%d columns, want %d", len(p.kites), tt.wantColumns)
			}
			if len(p.rows) != tt.wantHatsRows {
				t.Errorf("%d hat rows, want %d", len(p.rows), tt.wantHatsRows)
			}
			// Two placements with the same kites would collapse in this map.
			if len(p.placements) != len(p.rows) {
				t.Errorf("%d placements for %d rows", len(p.placements), len(p.rows))
			}
		})
	}
}

// TestSmallMatrix pins the table and the decoding example in the README.
func TestSmallMatrix(t *testing.T) {
	t.Parallel()

	want := []string{
		"0 1 4 5", "1 2", "3 4", "0 1 2 5", "2 3", "4 5",
		"0 1 2 3", "0 5", "3 4", "0 1", "1 2 3 4", "4 5",
		"0 5", "1 2", "2 3 4 5", "0 3 4 5", "0 1", "2 3",
	}
	p := newPuzzle(2, false)
	if len(p.rows) != len(want) {
		t.Fatalf("%d hat rows, want %d", len(p.rows), len(want))
	}
	for i, row := range p.rows {
		var inDisk []int
		for _, column := range row {
			if column < p.disk {
				inDisk = append(inDisk, column)
			}
		}
		sort.Ints(inDisk)
		if got := strings.Trim(fmt.Sprint(inDisk), "[]"); got != want[i] {
			t.Errorf("row %d covers disk kites %s, want %s", i, got, want[i])
		}
	}

	if got, want := fmt.Sprint(p.rows[1]), "[10 11 12 13 1 2 14 15]"; got != want {
		t.Errorf("row 1 = %s, want %s", got, want)
	}
	var first []placement
	p.solve(func(hats []placement) bool {
		first = hats
		return true
	})
	wantHats := []placement{p.placements[p.rows[1]], p.placements[p.rows[2]], p.placements[p.rows[7]]}
	if !reflect.DeepEqual(first, wantHats) {
		t.Errorf("first cover = %+v, want %+v", first, wantHats)
	}
}

// TestDisk compares the disk of the solver with the published kite counts and
// with the disk of the independent search.
func TestDisk(t *testing.T) {
	t.Parallel()

	for radius, want := range map[int]int{2: 6, 3: 18, 4: 30, 5: 48, 6: 72, 7: 84, 8: 114} {
		got := newPuzzle(radius, false)
		if got.disk != want {
			t.Errorf("radius %d: %d kites, want %d", radius, got.disk, want)
		}
		checker := newCoverChecker(radius, false)
		if len(checker.disk) != want {
			t.Errorf("radius %d: independent disk has %d kites, want %d", radius, len(checker.disk), want)
		}
		for _, k := range got.kites[:got.disk] {
			if !k.inDisk(radius) || distance(k.centre()) > float64(radius) {
				t.Errorf("radius %d: kite %v is outside the disk", radius, k)
			}
		}
	}
}

func TestSolve_Totals(t *testing.T) {
	t.Parallel()

	for _, tt := range coverTotals {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow total")
			}

			got := 0
			hats, reflected := [2]int{math.MaxInt32, 0}, [2]int{math.MaxInt32, 0}
			checker := newCoverChecker(tt.radius, tt.withMirrors)
			newPuzzle(tt.radius, tt.withMirrors).solve(func(cover []placement) bool {
				got++
				hats = widen(hats, len(cover))
				reflected = widen(reflected, countReflected(cover))
				if got <= checkedCovers {
					if problem := checker.check(cover); problem != "" {
						t.Fatal(problem)
					}
				}
				return false
			})
			if got != tt.want {
				t.Errorf("%d covers, want %d", got, tt.want)
			}
			if tt.hats != [2]int{} && hats != tt.hats {
				t.Errorf("covers use %v hats, want %v", hats, tt.hats)
			}
			if tt.reflected != [2]int{} && reflected != tt.reflected {
				t.Errorf("covers use %v reflected hats, want %v", reflected, tt.reflected)
			}
		})
	}
}

// widen stretches a range of smallest and largest values to include value.
func widen(r [2]int, value int) [2]int {
	if value < r[0] {
		r[0] = value
	}
	if value > r[1] {
		r[1] = value
	}
	return r
}

// TestIndependentSearch counts the covers with a separate search that doesn't
// use hatShape or the matrix.
func TestIndependentSearch(t *testing.T) {
	t.Parallel()

	for _, tt := range coverTotals {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow total")
			}

			if got := newCoverChecker(tt.radius, tt.withMirrors).count(); got != tt.want {
				t.Errorf("%d covers, want %d", got, tt.want)
			}
		})
	}
}

// coverChecker knows the covering problem through the hat outline alone. A kite
// is known only by its centre, and a hat only by the centres inside its outline.
type coverChecker struct {
	// disk lists the kite centres to cover, nearest first.
	disk []point
	// hats lists, for each centre in disk, the hats that contain it.
	hats map[point][][]point
	// placed maps a placement to its hat, to check reported covers.
	placed map[placement][]point
}

func newCoverChecker(radius int, withMirrors bool) *coverChecker {
	c := &coverChecker{hats: make(map[point][][]point), placed: make(map[placement][]point)}

	// A hat that touches the disk has its reference hexagon within this many
	// steps of the middle in each direction: hexagons are over 3 apart, and a
	// hat reaches less than 8.
	reach := radius/2 + 5
	for q := -reach; q <= reach; q++ {
		for r := -reach; r <= reach; r++ {
			for _, centre := range hexagonKites(q, r) {
				if distance(centre) <= float64(radius) {
					c.disk = append(c.disk, centre)
				}
			}
		}
	}
	sort.Slice(c.disk, func(i, j int) bool {
		a, b := c.disk[i], c.disk[j]
		if norm(a) != norm(b) {
			return norm(a) < norm(b)
		}
		return a.a < b.a || (a.a == b.a && a.b < b.b)
	})
	inDisk := make(map[point]bool, len(c.disk))
	for _, centre := range c.disk {
		inDisk[centre] = true
	}

	for _, mirrored := range []bool{false, true} {
		if mirrored && !withMirrors {
			continue
		}
		for steps := 0; steps < turns; steps++ {
			shape := centresInside(placement{mirrored: mirrored, turns: steps}.outline())
			if len(shape) != hatKites {
				panic("a hat does not hold eight kite centres")
			}
			for q := -reach; q <= reach; q++ {
				for r := -reach; r <= reach; r++ {
					c.add(placement{mirrored, steps, q, r}, shape, inDisk)
				}
			}
		}
	}
	return c
}

// add records hat, which is shape moved by q, r, if it touches the disk.
func (c *coverChecker) add(hat placement, shape []point, inDisk map[point]bool) {
	moved := make([]point, len(shape))
	touched := []point{}
	for i, centre := range shape {
		moved[i] = point{centre.a + 4*(2*hat.q-2*hat.r), centre.b + 4*(2*hat.q+4*hat.r)}
		if inDisk[moved[i]] {
			touched = append(touched, moved[i])
		}
	}
	if len(touched) == 0 {
		return
	}
	c.placed[hat] = moved
	for _, centre := range touched {
		c.hats[centre] = append(c.hats[centre], moved)
	}
}

// count finds the first kite of the disk that is still empty, tries every hat
// that fits on it, and goes on from there.
func (c *coverChecker) count() int {
	taken := make(map[point]bool)
	var search func(next int) int
	search = func(next int) int {
		for next < len(c.disk) && taken[c.disk[next]] {
			next++
		}
		if next == len(c.disk) {
			return 1
		}
		total := 0
		for _, hat := range c.hats[c.disk[next]] {
			if anyTaken(taken, hat) {
				continue
			}
			setTaken(taken, hat, true)
			total += search(next + 1)
			setTaken(taken, hat, false)
		}
		return total
	}
	return search(0)
}

func anyTaken(taken map[point]bool, hat []point) bool {
	for _, centre := range hat {
		if taken[centre] {
			return true
		}
	}
	return false
}

func setTaken(taken map[point]bool, hat []point, value bool) {
	for _, centre := range hat {
		taken[centre] = value
	}
}

// check returns a message unless hats overlap nowhere and fill the disk.
func (c *coverChecker) check(hats []placement) string {
	taken := make(map[point]bool)
	for _, hat := range hats {
		shape, ok := c.placed[hat]
		if !ok {
			return "cover has a hat that misses the disk"
		}
		if anyTaken(taken, shape) {
			return "cover has overlapping hats"
		}
		setTaken(taken, shape, true)
	}
	for _, centre := range c.disk {
		if !taken[centre] {
			return "cover leaves a kite of the disk empty"
		}
	}
	return ""
}

// hexagonKites returns four times the centres of the six kites of a hexagon. A
// centre is 5/4 of the way from the middle of the hexagon to its corner.
func hexagonKites(q, r int) []point {
	middle := point{2*q - 2*r, 2*q + 4*r}
	var centres []point
	for _, corner := range []point{{1, 0}, {0, 1}, {-1, 1}, {-1, 0}, {0, -1}, {1, -1}} {
		centres = append(centres, point{4*middle.a + 5*corner.a, 4*middle.b + 5*corner.b})
	}
	return centres
}

// centresInside returns four times every kite centre inside outline.
func centresInside(outline []point) []point {
	const reach = 8
	var inside []point
	for q := -reach; q <= reach; q++ {
		for r := -reach; r <= reach; r++ {
			for _, centre := range hexagonKites(q, r) {
				if insideOutline(outline, 4, centre) {
					inside = append(inside, centre)
				}
			}
		}
	}
	return inside
}

// insideOutline reports whether p, given at the stated scale, is inside the
// outline. It counts the outline's crossings of a ray from p. All values are
// whole numbers, so nothing is rounded.
func insideOutline(outline []point, scale int, p point) bool {
	inside := false
	for i := range outline {
		from, to := outline[i], outline[(i+1)%len(outline)]
		from = point{scale * from.a, scale * from.b}
		to = point{scale * to.a, scale * to.b}
		if (from.b > p.b) == (to.b > p.b) {
			continue
		}
		// The sign says on which side of the edge p lies.
		side := (to.a-from.a)*(p.b-from.b) - (to.b-from.b)*(p.a-from.a)
		if (to.b > from.b) == (side > 0) {
			inside = !inside
		}
	}
	return inside
}

// doubledArea returns twice the area of the outline, in units of a small
// parallelogram with sides 1 along the directions at 0 and 60 degrees.
func doubledArea(outline []point) int {
	area := 0
	for i := range outline {
		from, to := outline[i], outline[(i+1)%len(outline)]
		area += from.a*to.b - from.b*to.a
	}
	if area < 0 {
		return -area
	}
	return area
}

// norm returns the squared length of p.
func norm(p point) int {
	return p.a*p.a + p.a*p.b + p.b*p.b
}

// distance returns the length of a vector given at four times its size.
func distance(centre point) float64 {
	x, y := centre.position()
	return math.Hypot(x, y) / 4
}

// centre returns four times the centre of k.
func (k kite) centre() point {
	return hexagonKites(k.q, k.r)[k.dir]
}
