package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dimchansky/dlx"
)

// publishedPowersOfTwo holds OEIS A003095 from a(1): the first terms of the
// sequence are also the totals for n = 1, 2, 4, 8, 16 and 32.
var publishedPowersOfTwo = []uint64{1, 2, 5, 26, 677, 458330}

// publishedSplittable holds OEIS A353860 for n = 1 to 30.
var publishedSplittable = []uint64{
	1, 2, 2, 5, 2, 12, 2, 26, 9, 36,
	2, 206, 2, 132, 40, 677, 2, 1746, 2, 3398,
	136, 2052, 2, 44388, 33, 8196, 730, 79166, 2, 263234,
}

// backtrack finds every system without dlx or the matrix: it takes the smallest
// number not yet in a class and tries each class that could hold it.
func backtrack(n int, visit func(system []class)) {
	var moduli []int
	for m := 1; m <= n; m++ {
		if n%m == 0 {
			moduli = append(moduli, m)
		}
	}
	taken := make([]bool, n)
	var system []class
	var place func(first int)
	place = func(first int) {
		for first < n && taken[first] {
			first++
		}
		if first == n {
			sorted := append([]class(nil), system...)
			sort.Slice(sorted, func(i, j int) bool {
				if sorted[i].modulus != sorted[j].modulus {
					return sorted[i].modulus < sorted[j].modulus
				}
				return sorted[i].residue < sorted[j].residue
			})
			visit(sorted)
			return
		}
		for _, m := range moduli {
			residue := first % m
			if isFree(taken, residue, m) {
				setTaken(taken, residue, m, true)
				system = append(system, class{residue: residue, modulus: m})
				place(first + 1)
				system = system[:len(system)-1]
				setTaken(taken, residue, m, false)
			}
		}
	}
	place(0)
}

func isFree(taken []bool, residue, modulus int) bool {
	for x := residue; x < len(taken); x += modulus {
		if taken[x] {
			return false
		}
	}
	return true
}

func setTaken(taken []bool, residue, modulus int, value bool) {
	for x := residue; x < len(taken); x += modulus {
		taken[x] = value
	}
}

// checkSystem tests the definition directly on a stretch of integers longer
// than n, and the fact that the fractions of the integers in the classes add up
// to exactly 1, which it works out over the common denominator n.
func checkSystem(t *testing.T, n int, system []class) {
	t.Helper()
	sum := 0
	times := make([]int, 3*n)
	for _, c := range system {
		if n%c.modulus != 0 {
			t.Fatalf("%v: modulus %d does not divide %d", system, c.modulus, n)
		}
		sum += n / c.modulus
		for x := c.residue; x < len(times); x += c.modulus {
			times[x]++
		}
	}
	if sum != n {
		t.Fatalf("%v: fractions add up to %d/%d", system, sum, n)
	}
	for x, got := range times {
		if got != 1 {
			t.Fatalf("%v: %d is in %d classes", system, x, got)
		}
	}
}

// primePowerTotal is the rule proved in the README: for a prime p, a system
// for p^(k+1) is either the single class of all integers, or p systems for p^k,
// one for each remainder when dividing by p.
func primePowerTotal(p, k int) uint64 {
	total := uint64(1)
	for i := 0; i < k; i++ {
		product := uint64(1)
		for j := 0; j < p; j++ {
			product *= total
		}
		total = product + 1
	}
	return total
}

func countSystems(n int, distinct bool) uint64 {
	var total uint64
	solve(n, distinct, func([]class) bool {
		total++
		return false
	})
	return total
}

func TestDivisors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want []int
	}{
		{n: 1, want: []int{1}},
		{n: 7, want: []int{1, 7}},
		{n: 12, want: []int{1, 2, 3, 4, 6, 12}},
	}
	for _, tt := range tests {
		if got := divisors(tt.n); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("divisors(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

// TestDecode covers the two kinds of row, including a one-number class, whose
// size must not be confused with a slack row, and the sort order.
func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		n     int
		cover [][]int
		want  []class
	}{
		{
			name:  "plain rows",
			n:     4,
			cover: [][]int{{3}, {0, 2}, {1}},
			want:  []class{{0, 2}, {1, 4}, {3, 4}},
		},
		{
			name:  "slack rows are dropped",
			n:     4,
			cover: [][]int{{3, 5}, {4}, {1, 5}},
			want:  []class{{1, 4}, {3, 4}},
		},
		{
			name:  "modulus column does not count as a number",
			n:     4,
			cover: [][]int{{0, 2, 4}, {1, 5}, {3, 5}},
			want:  []class{{0, 2}, {1, 4}, {3, 4}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := decode(tt.n, tt.cover); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decode = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSolve_PrimePowers checks the published totals and the rule that predicts
// the total for any power of a prime.
func TestSolve_PrimePowers(t *testing.T) {
	t.Parallel()

	for k, published := range publishedPowersOfTwo {
		if got := primePowerTotal(2, k); got != published {
			t.Errorf("primePowerTotal(2, %d) = %d, want the published %d", k, got, published)
		}
	}

	tests := []struct {
		prime, power int
		slow         bool
	}{
		{prime: 2, power: 0},
		{prime: 2, power: 1},
		{prime: 2, power: 2},
		{prime: 2, power: 3},
		{prime: 2, power: 4},
		{prime: 2, power: 5, slow: true},
		{prime: 3, power: 1},
		{prime: 3, power: 2},
		{prime: 3, power: 3},
		{prime: 5, power: 1},
		{prime: 5, power: 2},
		{prime: 7, power: 1},
		{prime: 7, power: 2},
	}
	for _, tt := range tests {
		tt := tt
		n := 1
		for i := 0; i < tt.power; i++ {
			n *= tt.prime
		}
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: enumerates every system")
			}
			want := primePowerTotal(tt.prime, tt.power)
			if got := countSystems(n, false); got != want {
				t.Errorf("found %d systems, want %d", got, want)
			}
		})
	}
}

// TestSolve_MatchesBacktrack compares the two searches and tests each system
// against the definition. The fingerprints add up to the same value in any
// order, so equal sums make it overwhelmingly likely, though not certain, that
// the systems match and not only their number. It also counts the systems that
// splitting builds, which match a published sequence for n up to 30, and which
// for n=30 are not all of them.
func TestSolve_MatchesBacktrack(t *testing.T) {
	t.Parallel()

	for n := 1; n <= len(publishedSplittable); n++ {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			if n > 21 && testing.Short() {
				t.Skip("slow: enumerates every system")
			}

			var total, buildable, sum uint64
			solve(n, false, func(system []class) bool {
				checkSystem(t, n, system)
				total++
				sum += fingerprint(system)
				if splittable(system, 0, 1) {
					buildable++
				}
				return false
			})

			if want := publishedSplittable[n-1]; buildable != want {
				t.Errorf("splitting builds %d systems, want the published %d", buildable, want)
			}
			wantUnsplittable := uint64(0)
			if n == 30 {
				wantUnsplittable = 13500
			}
			if got := total - buildable; got != wantUnsplittable {
				t.Errorf("%d systems can't be built by splitting, want %d", got, wantUnsplittable)
			}

			var backtrackTotal, backtrackSum uint64
			backtrack(n, func(system []class) {
				backtrackTotal++
				backtrackSum += fingerprint(system)
				if len(system) > 1 && hasDistinctModuli(system) {
					t.Errorf("%s has distinct moduli", formatSystem(system))
				}
			})
			if total != backtrackTotal {
				t.Errorf("found %d systems, backtracking found %d", total, backtrackTotal)
			}
			if sum != backtrackSum {
				t.Errorf("found different systems from backtracking")
			}
		})
	}
}

// hasDistinctModuli relies on the system being sorted by modulus.
func hasDistinctModuli(system []class) bool {
	for i := 1; i < len(system); i++ {
		if system[i].modulus == system[i-1].modulus {
			return false
		}
	}
	return true
}

// fingerprint mixes the classes of a sorted system into one number.
func fingerprint(system []class) uint64 {
	var hash uint64
	for _, c := range system {
		hash = hash*1000003 + uint64(c.residue)
		hash = hash*1000003 + uint64(c.modulus)
	}
	return hash
}

// TestSolve_Distinct checks the theorem quoted in the README as far as is cheap.
func TestSolve_Distinct(t *testing.T) {
	t.Parallel()

	last := 120
	if testing.Short() {
		last = 60
	}
	for n := 1; n <= last; n++ {
		if got := countSystems(n, true); got != 0 {
			t.Errorf("n=%d: found %d systems with distinct moduli", n, got)
		}
	}
}

// TestBuildMatrix_DistinctRowsWork makes sure the empty results above come from
// the problem and not from a broken matrix. It adds spare classes of the largest
// modulus that skip the modulus column, so that modulus may repeat. Near misses
// must then appear, with every other modulus still used at most once.
func TestBuildMatrix_DistinctRowsWork(t *testing.T) {
	t.Parallel()

	const n = 8
	m := buildMatrix(n, true)
	for number := 0; number < n; number++ {
		m.AddRow(number)
	}

	const nearMiss = "0 mod 2, 1 mod 4, 3 mod 8, 7 mod 8"
	seen := false
	var covers int
	accept := func(exactCover [][]int) bool {
		covers++
		system := decode(n, exactCover)
		checkSystem(t, n, system)
		uses := make(map[int]int)
		for _, c := range system {
			uses[c.modulus]++
			if c.modulus != n && uses[c.modulus] > 1 {
				t.Errorf("%v repeats modulus %d", system, c.modulus)
			}
		}
		seen = seen || formatSystem(system) == nearMiss
		return false
	}
	m.Solve(dlx.SolutionAccepterFunc(accept))
	if !seen {
		t.Errorf("%q not found among %d covers", nearMiss, covers)
	}
}

// splittable reports whether the classes can be grown from the single class
// residue mod modulus by repeatedly splitting one class into equal parts.
func splittable(classes []class, residue, modulus int) bool {
	if len(classes) == 1 {
		return classes[0] == class{residue: residue, modulus: modulus}
	}
	for parts := 2; parts*modulus <= classes[0].modulus; parts++ {
		next := parts * modulus
		branches := make([][]class, parts)
		fits := true
		for _, c := range classes {
			if c.modulus%next != 0 {
				fits = false
				break
			}
			part := (c.residue%next - residue) / modulus
			branches[part] = append(branches[part], c)
		}
		if !fits {
			continue
		}
		all := true
		for part, branch := range branches {
			if len(branch) == 0 || !splittable(branch, residue+part*modulus, next) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// noFirstSplit is the system shown in the README. It is one of the exact
// covering systems for 30, but no prime divides all its moduli.
var noFirstSplit = []class{
	{0, 6},
	{2, 6},
	{1, 10},
	{3, 10},
	{5, 10},
	{9, 10},
	{7, 15},
	{4, 30},
	{10, 30},
	{16, 30},
	{17, 30},
	{27, 30},
	{28, 30},
}

func TestSplittable(t *testing.T) {
	t.Parallel()

	checkSystem(t, 30, noFirstSplit)

	tests := []struct {
		name   string
		system []class
		want   bool
	}{
		{name: "whole", system: []class{{0, 1}}, want: true},
		{name: "halves", system: []class{{0, 2}, {1, 2}}, want: true},
		{name: "halves then thirds", system: []class{{0, 2}, {1, 6}, {3, 6}, {5, 6}}, want: true},
		{name: "a gap", system: []class{{0, 2}, {1, 4}}, want: false},
		{name: "no first split exists", system: noFirstSplit, want: false},
	}
	for _, tt := range tests {
		if got := splittable(tt.system, 0, 1); got != tt.want {
			t.Errorf("%s: splittable = %v, want %v", tt.name, got, tt.want)
		}
	}
}
