package main

import (
	"fmt"
	"testing"
)

func TestNewProblem_Totals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rays        []ray
		dimension   int
		removed     int
		wantBases   int
		wantPairs   int
		wantColumns int
		wantRows    int
		want        int
	}{
		{
			name: "peres33", rays: peres33, dimension: 3, removed: noRay,
			wantBases: 16, wantPairs: 24, wantColumns: 40, wantRows: 57, want: 0,
		},
		{
			name: "peres33 without ray 1", rays: peres33, dimension: 3, removed: 0,
			wantBases: 12, wantPairs: 28, wantColumns: 40, wantRows: 60, want: 152,
		},
		{
			name: "peres33 without ray 4", rays: peres33, dimension: 3, removed: 3,
			wantBases: 14, wantPairs: 26, wantColumns: 40, wantRows: 58, want: 16,
		},
		{
			name: "peres33 without ray 10", rays: peres33, dimension: 3, removed: 9,
			wantBases: 15, wantPairs: 23, wantColumns: 38, wantRows: 55, want: 16,
		},
		{
			name: "peres33 without ray 22", rays: peres33, dimension: 3, removed: 21,
			wantBases: 15, wantPairs: 23, wantColumns: 38, wantRows: 55, want: 16,
		},
		{
			name: "peres24", rays: peres24, dimension: 4, removed: noRay,
			wantBases: 24, wantPairs: 0, wantColumns: 24, wantRows: 24, want: 0,
		},
		{
			name: "cabello18", rays: cabello18, dimension: 4, removed: noRay,
			wantBases: 9, wantPairs: 9, wantColumns: 18, wantRows: 27, want: 0,
		},
		{
			name: "cabello18 without ray 1", rays: cabello18, dimension: 4, removed: 0,
			wantBases: 7, wantPairs: 14, wantColumns: 21, wantRows: 31, want: 12,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := newProblem(tt.rays, tt.dimension, tt.removed)
			if len(p.bases) != tt.wantBases || len(p.pairs) != tt.wantPairs {
				t.Errorf("%d bases and %d pairs, want %d and %d", len(p.bases), len(p.pairs), tt.wantBases, tt.wantPairs)
			}
			if p.columns() != tt.wantColumns || p.matrixRows() != tt.wantRows {
				t.Errorf("%d columns and %d rows, want %d and %d", p.columns(), p.matrixRows(), tt.wantColumns, tt.wantRows)
			}
			if got := len(colourings(p)); got != tt.want {
				t.Errorf("%d colourings, want %d", got, tt.want)
			}
		})
	}
}

// TestBasesOnly checks the total with the rule for pairs dropped. It is the only
// total that needs no colouring to be decoded.
func TestBasesOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rays      []ray
		dimension int
		want      int
	}{
		{name: "peres33", rays: peres33, dimension: 3, want: 3072},
		{name: "peres24", rays: peres24, dimension: 4, want: 0},
		{name: "cabello18", rays: cabello18, dimension: 4, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := newProblem(tt.rays, tt.dimension, noRay)
			p.pairs = nil
			if got := len(colourings(p)); got != tt.want {
				t.Errorf("%d colourings, want %d", got, tt.want)
			}
		})
	}
}

// TestSmallExample checks the small matrix drawn in the README: rays 3, 4, 10,
// 13, 24 and 27 of the 33.
func TestSmallExample(t *testing.T) {
	t.Parallel()

	rays := []ray{peres33[3], peres33[12], peres33[9], peres33[2], peres33[26], peres33[23]}
	p := newProblem(rays, 3, noRay)

	if want := [][]int{{0, 1, 2}, {3, 4, 5}}; !equalGroups(p.bases, want) {
		t.Errorf("bases = %v, want %v", p.bases, want)
	}
	if want := [][]int{{2, 5}}; !equalGroups(p.pairs, want) {
		t.Errorf("pairs = %v, want %v", p.pairs, want)
	}
	if p.columns() != 3 || p.matrixRows() != 7 {
		t.Errorf("%d columns and %d rows, want 3 and 7", p.columns(), p.matrixRows())
	}
	if got := len(colourings(p)); got != 8 {
		t.Errorf("%d colourings, want 8", got)
	}
}

// removals lists every set with every choice of ray to leave out, or none. With
// -short it stops after the first ray, because the sweep is slow with -race.
func removals() []struct {
	name    string
	set     raySet
	removed int
} {
	var all []struct {
		name    string
		set     raySet
		removed int
	}
	for _, set := range raySets {
		last := len(set.rays) - 1
		if testing.Short() {
			last = 0
		}
		for removed := noRay; removed <= last; removed++ {
			all = append(all, struct {
				name    string
				set     raySet
				removed int
			}{fmt.Sprintf("%s without %d", set.name, removed+1), set, removed})
		}
	}
	return all
}

// TestBuildMatrix_EveryRayHasItsOwnRow proves the condition that solve relies on
// to decode: no ray has an empty row, a row equal to another ray's, or a row
// equal to a slack row.
func TestBuildMatrix_EveryRayHasItsOwnRow(t *testing.T) {
	t.Parallel()

	for _, tt := range removals() {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := newProblem(tt.set.rays, tt.set.dimension, tt.removed)
			_, rayOfRow := p.buildMatrix()

			want := len(tt.set.rays)
			if tt.removed != noRay {
				want-- // the removed ray has only the empty row
			}
			rayOfRowWithoutEmpty := len(rayOfRow)
			if _, ok := rayOfRow["[]"]; ok {
				rayOfRowWithoutEmpty--
				if tt.removed == noRay {
					t.Error("a ray belongs to no group")
				}
			}
			if rayOfRowWithoutEmpty != want {
				t.Errorf("%d different rows, want %d", rayOfRowWithoutEmpty, want)
			}
			for i := range p.pairs {
				if r, ok := rayOfRow[fmt.Sprint([]int{len(p.bases) + i})]; ok {
					t.Errorf("ray %d has the same row as a slack row", r+1)
				}
			}
		})
	}
}

// TestColourings_ObeyTheRules checks every decoded colouring again: no two green
// rays are perpendicular, going by the coordinates, and each basis has one.
func TestColourings_ObeyTheRules(t *testing.T) {
	t.Parallel()

	for _, tt := range removals() {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := newProblem(tt.set.rays, tt.set.dimension, tt.removed)
			seen := make(map[string]bool)
			for _, green := range colourings(p) {
				if key := fmt.Sprint(green); seen[key] {
					t.Errorf("colouring %s appears twice", key)
				} else {
					seen[key] = true
				}
				if contains(green, tt.removed) {
					t.Errorf("colouring %v uses the removed ray", green)
				}
				for i, u := range green {
					for _, v := range green[i+1:] {
						if tt.set.rays[u].perpendicular(tt.set.rays[v]) {
							t.Errorf("colouring %v: rays %d and %d are perpendicular", green, u+1, v+1)
						}
					}
				}
				for _, basis := range p.bases {
					count := 0
					for _, r := range basis {
						if contains(green, r) {
							count++
						}
					}
					if count != 1 {
						t.Errorf("colouring %v: %d green rays in basis %v", green, count, basis)
					}
				}
			}
		})
	}
}

// TestRemoval_MatchesPublishedCriticality checks what the papers say about
// taking away rays: arXiv:0909.4502 calls the 33 rays critical, meaning that
// every removal leaves a colourable set. arXiv:quant-ph/9706009 says the same
// of its 18 rays, and that the 24 rays of Peres are not critical.
func TestRemoval_MatchesPublishedCriticality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		set  raySet
		// wantTotals maps a colouring total to the number of rays giving it.
		wantTotals map[int]int
	}{
		{name: "peres33", set: raySets[0], wantTotals: map[int]int{152: 3, 16: 30}},
		{name: "peres24", set: raySets[1], wantTotals: map[int]int{0: 24}},
		{name: "cabello18", set: raySets[2], wantTotals: map[int]int{12: 18}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := make(map[int]int)
			for r := range tt.set.rays {
				got[len(colourings(newProblem(tt.set.rays, tt.set.dimension, r)))]++
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.wantTotals) {
				t.Errorf("totals = %v, want %v", got, tt.wantTotals)
			}
		})
	}
}

// TestGouldAravindColouring finds the colouring given in the paper for the 33
// rays without ray 1. It has no green ray in the pair left of basis 1 29 30.
func TestGouldAravindColouring(t *testing.T) {
	t.Parallel()

	green := fromOne([][]int{{2, 4, 8, 12, 14, 16, 19, 23, 27}})[0]
	want := fmt.Sprint(green)
	for _, got := range colourings(newProblem(peres33, 3, 0)) {
		if fmt.Sprint(got) == want {
			return
		}
	}
	t.Errorf("colouring %v was not found", want)
}

// TestColourings_AgainstBacktracking compares every total with a plain search
// over green and red. It shares no code or data with the solver: its groups come
// from the papers' tables and from whole-number coordinates in the test.
func TestColourings_AgainstBacktracking(t *testing.T) {
	t.Parallel()

	cabelloBases, cabelloPairs := cabelloGroups()
	tests := []struct {
		name  string
		rays  int
		bases [][]int
		pairs [][]int
		set   raySet
	}{
		{
			name: "peres33", rays: 33, set: raySets[0],
			bases: fromOne(gouldAravindTriads),
			pairs: fromOne(gouldAravindDyads),
		},
		{
			name: "peres24", rays: 24, set: raySets[1],
			bases: fromOne(waegellAravindBases),
		},
		{
			name: "cabello18", rays: 18, set: raySets[2],
			bases: cabelloBases,
			pairs: cabelloPairs,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			last := tt.rays - 1
			if testing.Short() {
				last = 0 // the search is slow with -race
			}
			for removed := noRay; removed <= last; removed++ {
				got := len(colourings(newProblem(tt.set.rays, tt.set.dimension, removed)))
				if want := countByBacktracking(tt.rays, tt.bases, tt.pairs, removed); got != want {
					t.Errorf("without ray %d: %d colourings, want %d", removed+1, got, want)
				}
			}

			p := newProblem(tt.set.rays, tt.set.dimension, noRay)
			p.pairs = nil
			got := len(colourings(p))
			if want := countByBacktracking(tt.rays, tt.bases, nil, noRay); got != want {
				t.Errorf("bases only: %d colourings, want %d", got, want)
			}
		})
	}
}

// cabelloGroups numbers the rays by first use in the equations, which is the
// numbering of cabello18, and finds the pairs by whole-number products.
func cabelloGroups() (bases, pairs [][]int) {
	var vectors [][]int
	number := func(v []int) int {
		for i, known := range vectors {
			if fmt.Sprint(known) == fmt.Sprint(v) {
				return i
			}
		}
		vectors = append(vectors, v)
		return len(vectors) - 1
	}

	for _, equation := range cabelloEquations {
		group := make([]int, len(equation))
		for i, vector := range equation {
			group[i] = number(vector)
		}
		bases = append(bases, group)
	}
	for i := range vectors {
		for j := i + 1; j < len(vectors); j++ {
			if dotInts(vectors[i], vectors[j]) == 0 && !inSameGroup(bases, i, j) {
				pairs = append(pairs, []int{i, j})
			}
		}
	}
	return bases, pairs
}

func inSameGroup(groups [][]int, i, j int) bool {
	for _, group := range groups {
		if contains(group, i) && contains(group, j) {
			return true
		}
	}
	return false
}

// countByBacktracking colours the rays one at a time and counts the colourings
// that break no rule. A removed ray takes no colour, a basis that held it stops
// being a basis, and the rest of that basis must hold at most one green ray.
func countByBacktracking(rays int, bases, pairs [][]int, removed int) int {
	var exactlyOne, atMostOne [][]int
	for _, basis := range bases {
		if contains(basis, removed) {
			atMostOne = append(atMostOne, basis)
		} else {
			exactlyOne = append(exactlyOne, basis)
		}
	}
	atMostOne = append(atMostOne, pairs...)

	const unknown, red, green = 0, 1, 2
	colour := make([]int, rays)
	if removed != noRay {
		colour[removed] = red
	}

	allowed := func() bool {
		for _, group := range exactlyOne {
			greens, unknowns := 0, 0
			for _, r := range group {
				switch colour[r] {
				case green:
					greens++
				case unknown:
					unknowns++
				}
			}
			if greens > 1 || greens+unknowns == 0 {
				return false
			}
		}
		for _, group := range atMostOne {
			greens := 0
			for _, r := range group {
				if colour[r] == green {
					greens++
				}
			}
			if greens > 1 {
				return false
			}
		}
		return true
	}

	total := 0
	var paint func(r int)
	paint = func(r int) {
		if r == rays {
			total++
			return
		}
		if r == removed {
			paint(r + 1)
			return
		}
		for _, c := range []int{red, green} {
			colour[r] = c
			if allowed() {
				paint(r + 1)
			}
		}
		colour[r] = unknown
	}
	paint(0)
	return total
}

// colourings collects every colouring of a problem.
func colourings(p problem) [][]int {
	var all [][]int
	p.solve(func(green []int) bool {
		all = append(all, green)
		return false
	})
	return all
}
