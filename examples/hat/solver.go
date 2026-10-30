package main

import "github.com/dimchansky/dlx"

// hatKites is the number of kites in one hat.
const hatKites = 8

// turns is the number of steps of 60 degrees in a full turn.
const turns = 6

// kite is one kite of the grid. The plane is covered by hexagons, and each
// hexagon is cut into six kites, one pointing at each of its corners.
//
// The pair q, r names a hexagon: start at the hexagon at the origin, take q
// steps to the neighbour at 30 degrees and r steps to the neighbour at 90
// degrees. The value dir names the kite in that hexagon: it points at the
// corner at dir*60 degrees.
type kite struct {
	q, r, dir int
}

// hatShape is the hat as eight kites. TestHatShape proves that these kites fill
// hatOutline exactly.
var hatShape = [hatKites]kite{
	{0, 0, 0},
	{0, 0, 1},
	{0, 0, 4},
	{0, 0, 5},
	{1, -1, 1},
	{1, -1, 2},
	{1, 0, 3},
	{1, 0, 4},
}

// turned returns k rotated 60 degrees anticlockwise around the origin.
func (k kite) turned() kite {
	return kite{-k.r, k.q + k.r, (k.dir + 1) % turns}
}

// mirrored returns k reflected in the line through the origin at 30 degrees.
func (k kite) mirrored() kite {
	return kite{k.q + k.r, -k.r, (turns + 1 - k.dir) % turns}
}

// inDisk reports whether the centre of k is at most radius from the origin. The
// centre is the average of the four corners of k, which lies 5/4 from the middle
// of its hexagon, on the way to the corner the kite points at.
//
// Four times the centre of k has whole coordinates a, b along the directions at
// 0 and 60 degrees, and a point with those coordinates is at a distance whose
// square is a*a + a*b + b*b. Comparing whole numbers avoids rounding.
func (k kite) inDisk(radius int) bool {
	direction := [turns][2]int{{1, 0}, {0, 1}, {-1, 1}, {-1, 0}, {0, -1}, {1, -1}}[k.dir]
	a := 4*(2*k.q-2*k.r) + 5*direction[0]
	b := 4*(2*k.q+4*k.r) + 5*direction[1]
	return a*a+a*b+b*b <= 16*radius*radius
}

// placement is one way to lay a hat on the grid: take hatShape, mirror it if
// mirrored is set, turn it by turns steps of 60 degrees, and move it by q, r.
type placement struct {
	mirrored bool
	turns    int
	q, r     int
}

// orient returns hatShape mirrored as asked and turned by the given steps.
func orient(mirrored bool, steps int) [hatKites]kite {
	shape := hatShape
	for i := range shape {
		if mirrored {
			shape[i] = shape[i].mirrored()
		}
		for step := 0; step < steps; step++ {
			shape[i] = shape[i].turned()
		}
	}
	return shape
}

// kites returns the kites of p, in hatShape order.
func (p placement) kites() [hatKites]kite {
	shape := orient(p.mirrored, p.turns)
	for i := range shape {
		shape[i].q += p.q
		shape[i].r += p.r
	}
	return shape
}

// puzzle is one covering problem.
type puzzle struct {
	// kites lists the columns: the kites of the disk first, then the kites
	// outside it that some hat reaches, which are the margin.
	kites []kite
	// disk is how many of the kites are in the disk.
	disk int
	// rows lists the matrix rows of the hats, one per placement.
	rows [][hatKites]int
	// placements maps the columns of a matrix row back to its placement.
	placements map[[hatKites]int]placement
}

// newPuzzle sets up the disk of the given radius. Without withMirrors,
// placements are rotated copies of the hat only.
func newPuzzle(radius int, withMirrors bool) *puzzle {
	var disk []kite
	for q := -radius; q <= radius; q++ {
		for r := -radius; r <= radius; r++ {
			for dir := 0; dir < turns; dir++ {
				if k := (kite{q, r, dir}); k.inDisk(radius) {
					disk = append(disk, k)
				}
			}
		}
	}
	return newPuzzleOn(disk, withMirrors)
}

// newPuzzleOn lists every placement that touches the given kites, which are the
// kites to cover.
func newPuzzleOn(disk []kite, withMirrors bool) *puzzle {
	p := &puzzle{placements: make(map[[hatKites]int]placement)}
	p.kites = append(p.kites, disk...)
	p.disk = len(p.kites)

	columns := make(map[kite]int)
	for column, k := range p.kites {
		columns[k] = column
	}

	seen := make(map[placement]bool)
	for _, mirrored := range []bool{false, true} {
		if mirrored && !withMirrors {
			continue
		}
		for steps := 0; steps < turns; steps++ {
			shape := orient(mirrored, steps)
			for _, target := range p.kites[:p.disk] {
				for _, k := range shape {
					// Moving the hat so that this kite lands on the target
					// gives every placement that covers the target.
					if k.dir != target.dir {
						continue
					}
					hat := placement{mirrored, steps, target.q - k.q, target.r - k.r}
					if !seen[hat] {
						seen[hat] = true
						p.addRow(hat, columns)
					}
				}
			}
		}
	}
	return p
}

// addRow adds the row of hat, giving each new margin kite its column.
func (p *puzzle) addRow(hat placement, columns map[kite]int) {
	var row [hatKites]int
	for i, k := range hat.kites() {
		column, ok := columns[k]
		if !ok {
			column = len(p.kites)
			columns[k] = column
			p.kites = append(p.kites, k)
		}
		row[i] = column
	}
	p.rows = append(p.rows, row)
	p.placements[row] = hat
}

// matrix has one column per kite. Each hat row covers its eight kites. Each
// margin kite also has a slack row, because a margin kite may stay empty, while
// a kite of the disk must be covered.
func (p *puzzle) matrix() dlx.Matrix {
	m := dlx.NewMatrix(len(p.kites))
	for i := range p.rows {
		m.AddRow(p.rows[i][:]...)
	}
	for column := p.disk; column < len(p.kites); column++ {
		m.AddRow(column)
	}
	return m
}

// solve passes each cover to accept until accept returns true.
func (p *puzzle) solve(accept func(hats []placement) bool) {
	p.matrix().Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(p.decode(exactCover))
	}))
}

// decode skips the slack rows, which have one column, and looks up the rest.
func (p *puzzle) decode(exactCover [][]int) []placement {
	var hats []placement
	for _, row := range exactCover {
		if len(row) != hatKites {
			continue
		}
		var key [hatKites]int
		copy(key[:], row)
		hats = append(hats, p.placements[key])
	}
	return hats
}
