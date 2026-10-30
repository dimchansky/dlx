package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dimchansky/dlx"
)

// bergRow is the row of Berg's Lyric Suite, F E C A G D Ab Db Eb Gb Bb B, as
// pitch numbers with C as 0.
var bergRow = []int{5, 4, 0, 9, 7, 2, 8, 1, 3, 6, 10, 11}

// backtrack enumerates the series that start on first without the exact cover
// machinery: it extends the series one pitch at a time and rejects a repeated
// pitch or a repeated interval. Unlike the matrix, it doesn't assume where a
// series ends, so it also checks that rule.
func backtrack(n, first int, visit func(series []int)) {
	series := []int{first}
	usedPitch := make([]bool, n)
	usedInterval := make([]bool, n)
	usedPitch[first] = true

	var extend func()
	extend = func() {
		if len(series) == n {
			visit(series)
			return
		}
		current := series[len(series)-1]
		for next := 0; next < n; next++ {
			interval := (next - current + n) % n
			if usedPitch[next] || usedInterval[interval] {
				continue
			}
			usedPitch[next], usedInterval[interval] = true, true
			series = append(series, next)
			extend()
			series = series[:len(series)-1]
			usedPitch[next], usedInterval[interval] = false, false
		}
	}
	extend()
}

// isAllInterval checks a series directly from the definition.
func isAllInterval(n int, series []int) bool {
	if len(series) != n {
		return false
	}
	pitches := make(map[int]bool)
	intervals := make(map[int]bool)
	for i, pitch := range series {
		if pitch < 0 || pitch >= n {
			return false
		}
		pitches[pitch] = true
		if i > 0 {
			intervals[(pitch-series[i-1]+n)%n] = true
		}
	}
	delete(intervals, 0)
	return len(pitches) == n && len(intervals) == n-1
}

// collect returns the series that start on first, as printed strings.
func collect(n, first int) map[string]bool {
	found := make(map[string]bool)
	solve(n, first, func(series []int) bool {
		found[fmt.Sprint(series)] = true
		return false
	})
	return found
}

// TestSolve_KnownCounts checks the totals and compares every series with an
// independent search. The total for 12 pitches is published. The smaller ones
// rest on the independent search alone.
func TestSolve_KnownCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want int
	}{
		{n: 2, want: 1},
		{n: 3, want: 0},
		{n: 4, want: 2},
		{n: 5, want: 0},
		{n: 6, want: 4},
		{n: 7, want: 0},
		{n: 8, want: 24},
		{n: 9, want: 0},
		{n: 10, want: 288},
		{n: 12, want: 3856},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("n=%d", tt.n), func(t *testing.T) {
			t.Parallel()
			if tt.n > 8 && testing.Short() {
				t.Skip("slow: enumerates every series")
			}

			found := make(map[string]bool)
			solve(tt.n, 0, func(series []int) bool {
				if !isAllInterval(tt.n, series) {
					t.Errorf("invalid series %v", series)
				}
				found[fmt.Sprint(series)] = true
				return false
			})
			if len(found) != tt.want {
				t.Errorf("found %d series, want %d", len(found), tt.want)
			}

			independent := 0
			backtrack(tt.n, 0, func(series []int) {
				independent++
				if !found[fmt.Sprint(series)] {
					t.Errorf("independent search found %v, which solve did not", series)
				}
			})
			if independent != tt.want {
				t.Errorf("independent search found %d series, want %d", independent, tt.want)
			}
		})
	}
}

// TestSolve_AnyFirst checks that the series from each starting pitch are
// shifted copies of the series from pitch 0, and that the totals add up. For 12
// pitches that total is the published 46272.
func TestSolve_AnyFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want int
	}{
		{n: 2, want: 2},
		{n: 4, want: 8},
		{n: 6, want: 24},
		{n: 8, want: 192},
		{n: 10, want: 2880},
		{n: 12, want: 46272},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("n=%d", tt.n), func(t *testing.T) {
			t.Parallel()
			if tt.n > 8 && testing.Short() {
				t.Skip("slow: enumerates every series")
			}

			base := collect(tt.n, 0)
			if got := tt.n * len(base); got != tt.want {
				t.Errorf("%d starting pitches and %d series each make %d, want %d", tt.n, len(base), got, tt.want)
			}

			independent := 0
			for first := 0; first < tt.n; first++ {
				backtrack(tt.n, first, func([]int) { independent++ })
			}
			if independent != tt.want {
				t.Errorf("independent search found %d series, want %d", independent, tt.want)
			}

			for first := 0; first < tt.n; first++ {
				first := first
				t.Run(fmt.Sprintf("first=%d", first), func(t *testing.T) {
					t.Parallel()

					total := 0
					solve(tt.n, first, func(series []int) bool {
						total++
						shifted := make([]int, len(series))
						for i, pitch := range series {
							shifted[i] = (pitch - first + tt.n) % tt.n
						}
						if series[0] != first || !base[fmt.Sprint(shifted)] {
							t.Errorf("series %v is not a shifted copy of a series from 0", series)
						}
						return false
					})
					if total != len(base) {
						t.Errorf("found %d series, want %d", total, len(base))
					}
				})
			}
		})
	}
}

// TestBacktrack_LastPitch checks the rule buildRows relies on, with a search
// that doesn't assume it: the last pitch is n/2 above the first.
func TestBacktrack_LastPitch(t *testing.T) {
	t.Parallel()

	sizes := []int{2, 4, 6, 8, 10, 12}
	if testing.Short() {
		sizes = sizes[:4]
	}
	for _, n := range sizes {
		backtrack(n, 0, func(series []int) {
			if last := series[n-1]; last != n/2 {
				t.Errorf("series %v ends on %d, want %d", series, last, n/2)
			}
		})
	}
}

// TestSolve_FindsBergRow checks that the historical row passes the definition
// and is among the series the matrix finds.
func TestSolve_FindsBergRow(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow: searches many series")
	}

	if !isAllInterval(12, bergRow) {
		t.Errorf("%v is not all-interval", bergRow)
	}
	found := false
	solve(12, bergRow[0], func(series []int) bool {
		found = reflect.DeepEqual(series, bergRow)
		return found
	})
	if !found {
		t.Errorf("%v was not found", bergRow)
	}
}

// TestWedge checks the construction in the README that gives a series for every
// even size: 0, n-1, 1, n-2, and so on.
func TestWedge(t *testing.T) {
	t.Parallel()

	for n := minSize; n <= maxSize; n += 2 {
		wedge := make([]int, n)
		for i := range wedge {
			if i%2 == 0 {
				wedge[i] = i / 2
			} else {
				wedge[i] = n - 1 - i/2
			}
		}
		if !isAllInterval(n, wedge) {
			t.Errorf("wedge %v is not all-interval", wedge)
		}
	}
}

func TestSolve_StopsWhenAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		first int
	}{
		{name: "one starting pitch", first: 0},
		{name: "stops across starting pitches", first: anyFirst},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			solve(8, tt.first, func([]int) bool {
				calls++
				return calls == 3
			})
			if calls != 3 {
				t.Errorf("accepter called %d times, want 3", calls)
			}
		})
	}
}

// TestBuildRows_Sizes checks the matrix sizes quoted in the README.
func TestBuildRows_Sizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n, wantColumns, wantRows int
	}{
		{n: 4, wantColumns: 18, wantRows: 18},
		{n: 12, wantColumns: 154, wantRows: 1210},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("n=%d", tt.n), func(t *testing.T) {
			t.Parallel()

			if got := (columns{tt.n}).count(); got != tt.wantColumns {
				t.Errorf("%d columns, want %d", got, tt.wantColumns)
			}
			rows := buildRows(tt.n, 0)
			if len(rows) != tt.wantRows {
				t.Errorf("%d rows, want %d", len(rows), tt.wantRows)
			}
			for _, row := range rows {
				for _, column := range row {
					if column < 0 || column >= tt.wantColumns {
						t.Fatalf("row %v has a column out of range", row)
					}
				}
			}
		})
	}
}

// TestBuildRows_WorkedExample checks the three rows of the series 0 3 1 2 that
// the README spells out, with column numbers worked out by hand.
func TestBuildRows_WorkedExample(t *testing.T) {
	t.Parallel()

	want := [][]int{
		{0, 3, 9, 13},
		{1, 6, 8, 15, 10, 11, 12},
		{2, 4, 7, 5, 14, 16, 17},
	}
	rows := make(map[string]bool)
	for _, row := range buildRows(4, 0) {
		rows[fmt.Sprint(row)] = true
	}
	for _, row := range want {
		if !rows[fmt.Sprint(row)] {
			t.Errorf("row %v is missing", row)
		}
	}

	var series [][]int
	solve(4, 0, func(s []int) bool {
		series = append(series, append([]int(nil), s...))
		return false
	})
	sort.Slice(series, func(i, j int) bool { return fmt.Sprint(series[i]) < fmt.Sprint(series[j]) })
	if wantSeries := [][]int{{0, 1, 3, 2}, {0, 3, 1, 2}}; !reflect.DeepEqual(series, wantSeries) {
		t.Errorf("series = %v, want %v", series, wantSeries)
	}
}

// TestEqualityTrick checks the claim in isolation: one row covers column v, the
// other covers every column except w, and the pair is an exact cover exactly
// when v equals w.
func TestEqualityTrick(t *testing.T) {
	t.Parallel()

	const size = 4
	for v := 0; v < size; v++ {
		for w := 0; w < size; w++ {
			var except []int
			for column := 0; column < size; column++ {
				if column != w {
					except = append(except, column)
				}
			}
			m := dlx.NewMatrix(size)
			m.AddRow(v)
			m.AddRow(except...)

			solutions := 0
			m.Solve(dlx.SolutionAccepterFunc(func([][]int) bool {
				solutions++
				return false
			}))
			if (solutions == 1) != (v == w) {
				t.Errorf("v=%d w=%d: %d solutions", v, w, solutions)
			}
		}
	}
}
