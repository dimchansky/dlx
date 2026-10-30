package main

import "github.com/dimchansky/dlx"

// anyFirst asks solve for every series, whatever pitch it starts on.
const anyFirst = -1

// columns gives the column numbers of the matrix for n pitches. The families
// are laid out one after another: steps, pitches, intervals, then links.
type columns struct {
	n int
}

// step is the column of step s, the move from position s to position s+1.
func (c columns) step(s int) int { return s }

// pitch is the column of a pitch, which must be used exactly once.
func (c columns) pitch(p int) int { return c.n - 1 + p }

// interval is the column of an interval from 1 to n-1.
func (c columns) interval(d int) int { return 2*c.n - 1 + d - 1 }

// link is the column of a position shared by two steps and one pitch. Only the
// inner positions 1 to n-2 are shared.
func (c columns) link(position, pitch int) int {
	return 3*c.n - 2 + (position-1)*c.n + pitch
}

// count is the number of columns.
func (c columns) count() int { return 3*c.n - 2 + (c.n-2)*c.n }

// solve passes each all-interval series of n pitches to accept until accept
// returns true. A series lists the pitches in order. It starts on the pitch
// first, or on every pitch in turn if first is anyFirst.
func solve(n, first int, accept func(series []int) bool) {
	if first != anyFirst {
		solveFrom(n, first, accept)
		return
	}
	for f := 0; f < n; f++ {
		stopped := false
		solveFrom(n, f, func(series []int) bool {
			stopped = accept(series)
			return stopped
		})
		if stopped {
			return
		}
	}
}

func solveFrom(n, first int, accept func(series []int) bool) {
	m := dlx.NewMatrix(columns{n}.count())
	for _, row := range buildRows(n, first) {
		m.AddRow(row...)
	}
	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(n, exactCover))
	}))
}

// buildRows returns one matrix row for each step and each pair of different
// pitches on that step. A row covers its step, the pitch it leaves, and the
// interval it makes. The last step also covers the pitch it reaches, so all n
// pitches are covered by n-1 steps.
//
// Exact cover can't compare two rows, so neighbouring steps agree on their
// shared pitch through link columns. The step that arrives at a position with
// pitch y covers the single link column (position, y). The step that leaves it
// from pitch x covers (position, p) for every p except x. Together they cover
// each of the n link columns exactly once if and only if x equals y.
//
// The first step leaves only from first. The last step arrives only at
// first+n/2: the n-1 intervals add up to n(n-1)/2, which for even n is n/2 more
// than a whole number of turns round the circle of pitches. No series exists
// for odd n, so the rule can't lose one there either. Fixing the last pitch
// removes few rows but makes the search much faster.
func buildRows(n, first int) [][]int {
	c := columns{n}
	lastStep := n - 2
	last := (first + n/2) % n

	var rows [][]int
	for s := 0; s <= lastStep; s++ {
		for x := 0; x < n; x++ {
			if s == 0 && x != first {
				continue
			}
			for y := 0; y < n; y++ {
				if x == y || (s == lastStep && y != last) {
					continue
				}
				row := []int{c.step(s), c.pitch(x), c.interval((y - x + n) % n)}
				if s == lastStep {
					row = append(row, c.pitch(y))
				} else {
					row = append(row, c.link(s+1, y))
				}
				if s > 0 {
					for p := 0; p < n; p++ {
						if p != x {
							row = append(row, c.link(s, p))
						}
					}
				}
				rows = append(rows, row)
			}
		}
	}
	return rows
}

// decode relies on dlx keeping each row's columns in AddRow order: the step
// comes first, then the pitch it leaves. On the last step the fourth column is
// the pitch it reaches.
func decode(n int, exactCover [][]int) []int {
	c := columns{n}
	series := make([]int, n)
	for _, row := range exactCover {
		s := row[0]
		series[s] = row[1] - c.pitch(0)
		if s == n-2 {
			series[n-1] = row[3] - c.pitch(0)
		}
	}
	return series
}
