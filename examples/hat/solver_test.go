package main

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/dimchansky/dlx"
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
	// hats gives the fewest and most hats in any cover, and rarer the fewest and
	// most hats of the rarer shape, when set.
	hats, rarer [2]int
	// splits gives the number of covers for some pairs of hats and reflected
	// hats.
	splits map[[2]int]int
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
		hats: [2]int{13, 19}, rarer: [2]int{1, 7},
		splits: map[[2]int]int{{15, 7}: 24, {15, 8}: 24},
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

			p := newPuzzle(diskKites(tt.radius), tt.withMirrors)
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

// readmeMatrixRow is one row of the small matrix in the README: its number, its
// six cells for the disk, and its count of margin kites.
var readmeMatrixRow = regexp.MustCompile(`(?m)^\| (\*\*)?h(\d+)(?:\*\*)? \|(.*)\| (\d+) \|$`)

// readmeSolution is the raw solution in the README, up to the three dots.
var readmeSolution = regexp.MustCompile(`(?m)^    (\[\[.*) \.\.\.$`)

// TestSmallMatrix compares the table and the decoding example in the README
// with the program.
func TestSmallMatrix(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	table := readmeMatrixRow.FindAllStringSubmatch(string(readme), -1)

	p := newPuzzle(diskKites(2), false)
	if len(table) != len(p.rows) {
		t.Fatalf("README has %d hat rows, want %d", len(table), len(p.rows))
	}
	var wantHats []placement
	for i, cells := range table {
		if cells[2] != strconv.Itoa(i) {
			t.Fatalf("README row %d is named h%s", i, cells[2])
		}
		var want []string
		margin := 0
		seen := make([]bool, p.disk)
		for _, column := range p.rows[i] {
			if column < p.disk {
				seen[column] = true
			} else {
				margin++
			}
		}
		for _, inDisk := range seen {
			cell := "0"
			if inDisk {
				cell = "1"
			}
			want = append(want, cell)
		}
		got := strings.Fields(strings.NewReplacer("|", " ", "*", " ").Replace(cells[3]))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("README row h%d = %v, want %v", i, got, want)
		}
		if cells[4] != strconv.Itoa(margin) {
			t.Errorf("README row h%d has %s margin kites, want %d", i, cells[4], margin)
		}
		if cells[1] != "" {
			wantHats = append(wantHats, p.placements[p.rows[i]])
		}
	}

	if got, want := fmt.Sprint(p.rows[1]), "[10 11 12 13 1 2 14 15]"; got != want {
		t.Errorf("row 1 = %s, want %s", got, want)
	}
	var first []placement
	var raw string
	p.matrix().Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		first = p.decode(exactCover)
		raw = fmt.Sprint(exactCover)
		return true
	}))
	if !reflect.DeepEqual(first, wantHats) {
		t.Errorf("first cover = %+v, want %+v", first, wantHats)
	}
	shown := readmeSolution.FindStringSubmatch(string(readme))
	if shown == nil {
		t.Fatal("README has no raw solution line")
	}
	if !strings.HasPrefix(raw, shown[1]) {
		t.Errorf("README solution %s doesn't start %s", shown[1], raw)
	}
}

// readmeSliceRow is one row of the slice of the matrix in the README.
var readmeSliceRow = regexp.MustCompile(`(?m)^\| (hat h|slack m)(\d+) +\| (\d) +\| (\d) +\| (\d) +\| (\d) +\|$`)

// TestMatrixSlice compares the table with margin columns in the README with the
// matrix.
func TestMatrixSlice(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	p := newPuzzle(diskKites(2), false)
	columns := []int{1, 10, 14, 23}
	table := readmeSliceRow.FindAllStringSubmatch(string(readme), -1)
	if len(table) != 5 {
		t.Fatalf("README has %d rows in the slice, want 5", len(table))
	}
	for _, cells := range table {
		number, err := strconv.Atoi(cells[2])
		if err != nil {
			t.Fatal(err)
		}
		row := []int{number}
		if cells[1] == "hat h" {
			row = p.rows[number][:]
		} else if number < p.disk || number >= len(p.kites) {
			t.Fatalf("%s%s is not a margin kite", cells[1], cells[2])
		}
		for i, column := range columns {
			want := "0"
			for _, used := range row {
				if used == column {
					want = "1"
				}
			}
			if cells[3+i] != want {
				t.Errorf("%s%s has %s in column %d, want %s", cells[1], cells[2], cells[3+i], column, want)
			}
		}
	}
}

// TestDisk compares the disk of the solver with the published kite counts and
// with the disk of the independent search.
func TestDisk(t *testing.T) {
	t.Parallel()

	for radius, want := range map[int]int{2: 6, 3: 18, 4: 30, 5: 48, 6: 72, 7: 84, 8: 114} {
		got := newPuzzle(diskKites(radius), false)
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

// TestKiteCentre checks that a centre is the average of the corners, as the
// README says.
func TestKiteCentre(t *testing.T) {
	t.Parallel()

	for dir := 0; dir < turns; dir++ {
		k := kite{2, -1, dir}
		var sum point
		for _, corner := range k.corners() {
			sum = point{sum.a + corner.a, sum.b + corner.b}
		}
		if sum != k.centre() {
			t.Errorf("kite %v has corner total %v, want %v", k, sum, k.centre())
		}
	}
}

// TestMarginColumns counts the covers of a matrix with no margin columns. The
// hats then may overlap outside the disk, so most of these covers are false.
func TestMarginColumns(t *testing.T) {
	t.Parallel()

	for radius, want := range map[int]int{2: 28, 3: 174, 4: 156, 5: 198, 6: 38} {
		got := 0
		solveDiskOnly(radius, func([]placement) bool {
			got++
			return false
		})
		if got != want {
			t.Errorf("radius %d: %d covers without margin columns, want %d", radius, got, want)
		}
	}
}

// solveDiskOnly passes each cover of a matrix with no margin columns, for hats
// without mirrors, to accept until accept returns true.
func solveDiskOnly(radius int, accept func(hats []placement) bool) {
	p := newPuzzle(diskKites(radius), false)
	m := dlx.NewMatrix(p.disk)
	hats := make(map[string]placement)
	for _, row := range p.rows {
		var inDisk []int
		for _, column := range row {
			if column < p.disk {
				inDisk = append(inDisk, column)
			}
		}
		m.AddRow(inDisk...)
		hats[fmt.Sprint(inDisk)] = p.placements[row]
	}
	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		cover := make([]placement, len(exactCover))
		for i, row := range exactCover {
			cover[i] = hats[fmt.Sprint(row)]
		}
		return accept(cover)
	}))
}

// TestRoundDisk asks for every kite with any part strictly inside the circle,
// not only the kites with their centre inside it.
func TestRoundDisk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		radius, kites, covers int
	}{
		{radius: 2, kites: 18, covers: 46},
		{radius: 3, kites: 18, covers: 46},
		{radius: 4, kites: 42, covers: 64},
		{radius: 5, kites: 60, covers: 0},
		{radius: 6, kites: 72, covers: 0},
		{radius: 7, kites: 114, covers: 0},
	}
	for _, tt := range tests {
		var disk []kite
		for q := -tt.radius; q <= tt.radius; q++ {
			for r := -tt.radius; r <= tt.radius; r++ {
				for dir := 0; dir < turns; dir++ {
					if k := (kite{q, r, dir}); k.crossesCircle(tt.radius) {
						disk = append(disk, k)
					}
				}
			}
		}
		if len(disk) != tt.kites {
			t.Errorf("radius %d: %d kites, want %d", tt.radius, len(disk), tt.kites)
		}
		got := 0
		newPuzzle(disk, false).solve(func([]placement) bool {
			got++
			return false
		})
		if got != tt.covers {
			t.Errorf("radius %d: %d covers, want %d", tt.radius, got, tt.covers)
		}
	}

	// At radius 6 the two rules pick the same kites, so the result that mirror
	// images are needed doesn't depend on the rule.
	for _, k := range newPuzzle(diskKites(6), false).kites[:72] {
		if !k.crossesCircle(6) {
			t.Errorf("kite %v of the disk has no part inside the circle", k)
		}
	}
}

// crossesCircle reports whether any part of k is strictly less than radius from
// the origin. All values are whole numbers, so nothing is rounded.
func (k kite) crossesCircle(radius int) bool {
	corners := k.corners()
	for i, from := range corners {
		to := corners[(i+1)%len(corners)]
		side := point{to.a - from.a, to.b - from.b}
		// along is twice the dot product of from and side, and length is twice
		// the squared length of side.
		along := 2*from.a*side.a + 2*from.b*side.b + from.a*side.b + from.b*side.a
		length := 2 * norm(side)
		switch {
		case along >= 0:
			if norm(from) < radius*radius {
				return true
			}
		case -along >= length:
			if norm(to) < radius*radius {
				return true
			}
		default:
			if 2*norm(from)*length-along*along < 2*radius*radius*length {
				return true
			}
		}
	}
	return false
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

			got := coverStats{}
			checked := 0
			checker := newCoverChecker(tt.radius, tt.withMirrors)
			newPuzzle(diskKites(tt.radius), tt.withMirrors).solve(func(cover []placement) bool {
				got[[2]int{len(cover), countReflected(cover)}]++
				if checked < checkedCovers {
					checked++
					if problem := checker.check(cover); problem != "" {
						t.Fatal(problem)
					}
				}
				return false
			})
			checkStats(t, got, tt.want, tt.hats, tt.rarer, tt.splits)
		})
	}
}

// TestIndependentSearch repeats every total and range with a separate search,
// which shares no code with the solver.
func TestIndependentSearch(t *testing.T) {
	t.Parallel()

	for _, tt := range coverTotals {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow total")
			}

			got := newCoverChecker(tt.radius, tt.withMirrors).count()
			checkStats(t, got, tt.want, tt.hats, tt.rarer, tt.splits)
		})
	}
}

// coverStats counts covers by their number of hats and of reflected hats.
type coverStats map[[2]int]int

// checkStats compares stats with the published values. The hats and rarer ranges
// and the splits are checked when set.
func checkStats(t *testing.T, stats coverStats, want int, hats, rarer [2]int, splits map[[2]int]int) {
	t.Helper()

	total := 0
	gotHats, gotRarer := [2]int{math.MaxInt32, 0}, [2]int{math.MaxInt32, 0}
	for split, covers := range stats {
		total += covers
		gotHats = widen(gotHats, split[0])
		gotRarer = widen(gotRarer, min(split[1], split[0]-split[1]))
	}
	if total != want {
		t.Errorf("%d covers, want %d", total, want)
	}
	if hats != [2]int{} && gotHats != hats {
		t.Errorf("covers use %v hats, want %v", gotHats, hats)
	}
	if rarer != [2]int{} && gotRarer != rarer {
		t.Errorf("covers use %v hats of the rarer shape, want %v", gotRarer, rarer)
	}
	for split, covers := range splits {
		if stats[split] != covers {
			t.Errorf("%d covers with %v hats and reflected hats, want %d", stats[split], split, covers)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// spot is a point of the plane for the independent search. It lies at x/8, and
// at y times the square root of 3, divided by 8. These are not the coordinates
// of the solver, and they make all the moves below whole numbers.
type spot struct {
	x, y int
}

// turned returns s rotated 60 degrees anticlockwise around the origin.
func (s spot) turned() spot {
	return spot{(s.x - 3*s.y) / 2, (s.x + s.y) / 2}
}

// flipped returns s reflected in the line through the origin at 30 degrees.
func (s spot) flipped() spot {
	return spot{(s.x + 3*s.y) / 2, (s.x - s.y) / 2}
}

// inDisk reports whether s is at most radius from the origin.
func (s spot) inDisk(radius int) bool {
	return s.x*s.x+3*s.y*s.y <= 64*radius*radius
}

// inside reports whether s is inside the outline, by counting the sides that a
// ray from s crosses. Stretching the plane along y keeps this true, so the
// coordinates can be used as they are.
func (s spot) inside(outline []spot) bool {
	inside := false
	for i := range outline {
		from, to := outline[i], outline[(i+1)%len(outline)]
		if (from.y > s.y) == (to.y > s.y) {
			continue
		}
		left := (to.x-from.x)*(s.y-from.y)-(to.y-from.y)*(s.x-from.x) > 0
		if left == (to.y > from.y) {
			inside = !inside
		}
	}
	return inside
}

// hexagonMiddle returns the middle of a hexagon, which is 6q, 2q+4r in units of
// 1/4 of those of a spot. Its neighbours are 2 times the square root of 3 away.
func hexagonMiddle(q, r int) spot {
	return spot{4 * 6 * q, 4 * (2*q + 4*r)}
}

// kiteCentres returns the centres of the six kites of a hexagon. Each lies 5/4
// from the middle, towards a corner of the hexagon.
func kiteCentres(q, r int) [turns]spot {
	var centres [turns]spot
	middle := hexagonMiddle(q, r)
	arm := spot{10, 0}
	for dir := range centres {
		centres[dir] = spot{middle.x + arm.x, middle.y + arm.y}
		arm = arm.turned()
	}
	return centres
}

// checkedOutline is the published outline of the hat in the units of spot, before
// the scale of 4 that kiteCentres uses. It is written out again here so that the
// independent search takes nothing from the solver, not even this list.
var checkedOutline = []spot{
	{0, 0},
	{-3, -1},
	{-2, -2},
	{2, -2},
	{3, -1},
	{6, -2},
	{9, -1},
	{8, 0},
	{6, 0},
	{6, 2},
	{3, 3},
	{2, 2},
	{0, 2},
}

// checkedHat is a hat of the independent search: the kite centres it holds.
type checkedHat struct {
	centres  []spot
	mirrored bool
}

// coverChecker knows the covering problem through the hat outline alone. A kite
// is known only by its centre, and a hat only by the centres inside its outline.
type coverChecker struct {
	// disk lists the kite centres to cover, nearest first.
	disk []spot
	// hats lists, for each centre in disk, the hats that contain it.
	hats map[spot][]*checkedHat
	// placed maps the way the solver names a placement to its hat, to check
	// reported covers.
	placed map[placement]*checkedHat
}

func newCoverChecker(radius int, withMirrors bool) *coverChecker {
	c := &coverChecker{hats: make(map[spot][]*checkedHat), placed: make(map[placement]*checkedHat)}

	// A hat that touches the disk has its reference hexagon within this many
	// steps of the middle in each direction: hexagons are over 3 apart, and a
	// hat reaches less than 8.
	reach := radius/2 + 5
	for q := -reach; q <= reach; q++ {
		for r := -reach; r <= reach; r++ {
			for _, centre := range kiteCentres(q, r) {
				if centre.inDisk(radius) {
					c.disk = append(c.disk, centre)
				}
			}
		}
	}
	sort.Slice(c.disk, func(i, j int) bool {
		a, b := c.disk[i], c.disk[j]
		if da, db := a.x*a.x+3*a.y*a.y, b.x*b.x+3*b.y*b.y; da != db {
			return da < db
		}
		return a.x < b.x || (a.x == b.x && a.y < b.y)
	})
	inDisk := make(map[spot]bool, len(c.disk))
	for _, centre := range c.disk {
		inDisk[centre] = true
	}

	for _, mirrored := range []bool{false, true} {
		if mirrored && !withMirrors {
			continue
		}
		outline := make([]spot, len(checkedOutline))
		for i, corner := range checkedOutline {
			if mirrored {
				corner = corner.flipped()
			}
			outline[i] = spot{4 * corner.x, 4 * corner.y}
		}
		for steps := 0; steps < turns; steps++ {
			shape := centresInside(outline)
			if len(shape) != hatKites {
				panic("a hat does not hold eight kite centres")
			}
			for q := -reach; q <= reach; q++ {
				for r := -reach; r <= reach; r++ {
					c.add(placement{mirrored, steps, q, r}, shape, inDisk)
				}
			}
			for i, corner := range outline {
				outline[i] = corner.turned()
			}
		}
	}
	return c
}

// add records shape moved by name.q, name.r, if that touches the disk.
func (c *coverChecker) add(name placement, shape []spot, inDisk map[spot]bool) {
	hat := &checkedHat{centres: make([]spot, len(shape)), mirrored: name.mirrored}
	move := hexagonMiddle(name.q, name.r)
	touched := []spot{}
	for i, centre := range shape {
		hat.centres[i] = spot{centre.x + move.x, centre.y + move.y}
		if inDisk[hat.centres[i]] {
			touched = append(touched, hat.centres[i])
		}
	}
	if len(touched) == 0 {
		return
	}
	c.placed[name] = hat
	for _, centre := range touched {
		c.hats[centre] = append(c.hats[centre], hat)
	}
}

// count finds the first kite of the disk that is still empty, tries every hat
// that fits on it, and goes on from there.
func (c *coverChecker) count() coverStats {
	stats := coverStats{}
	taken := make(map[spot]bool)
	var search func(next, hats, reflected int)
	search = func(next, hats, reflected int) {
		for next < len(c.disk) && taken[c.disk[next]] {
			next++
		}
		if next == len(c.disk) {
			stats[[2]int{hats, reflected}]++
			return
		}
		for _, hat := range c.hats[c.disk[next]] {
			if anyTaken(taken, hat) {
				continue
			}
			setTaken(taken, hat, true)
			if hat.mirrored {
				search(next+1, hats+1, reflected+1)
			} else {
				search(next+1, hats+1, reflected)
			}
			setTaken(taken, hat, false)
		}
	}
	search(0, 0, 0)
	return stats
}

func anyTaken(taken map[spot]bool, hat *checkedHat) bool {
	for _, centre := range hat.centres {
		if taken[centre] {
			return true
		}
	}
	return false
}

func setTaken(taken map[spot]bool, hat *checkedHat, value bool) {
	for _, centre := range hat.centres {
		taken[centre] = value
	}
}

// check returns a message unless hats overlap nowhere and fill the disk.
func (c *coverChecker) check(hats []placement) string {
	taken := make(map[spot]bool)
	for _, name := range hats {
		hat, ok := c.placed[name]
		if !ok {
			return "cover has a hat that misses the disk"
		}
		if anyTaken(taken, hat) {
			return "cover has overlapping hats"
		}
		setTaken(taken, hat, true)
	}
	for _, centre := range c.disk {
		if !taken[centre] {
			return "cover leaves a kite of the disk empty"
		}
	}
	return ""
}

// centresInside returns every kite centre inside outline, which is at the scale
// of the centres.
func centresInside(outline []spot) []spot {
	const reach = 8
	var inside []spot
	for q := -reach; q <= reach; q++ {
		for r := -reach; r <= reach; r++ {
			for _, centre := range kiteCentres(q, r) {
				if centre.inside(outline) {
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
	corners := k.corners()
	middle, tip := corners[0], corners[2]
	return point{4*middle.a + 5*(tip.a-middle.a)/2, 4*middle.b + 5*(tip.b-middle.b)/2}
}
