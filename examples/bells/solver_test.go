package main

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dimchansky/dlx"
)

// knownTotals holds the totals for every set of changes. The 10792 for abcd is
// published in "Ringing the changes" by Polster and Ross, Plus magazine, 2009.
// The other totals have no published source, so TestSolve_Totals proves them
// against two separate methods.
var knownTotals = []struct {
	bells   int
	names   string
	covers  int
	extents int
}{
	{3, "a", 1, 0},
	{3, "b", 1, 0},
	{3, "ab", 4, 2},
	{4, "a", 1, 0},
	{4, "b", 1, 0},
	{4, "c", 1, 0},
	{4, "d", 1, 0},
	{4, "ab", 256, 0},
	{4, "ac", 4096, 0},
	{4, "ad", 4096, 0},
	{4, "bc", 256, 0},
	{4, "bd", 64, 0},
	{4, "cd", 4096, 0},
	{4, "abc", 28561, 88},
	{4, "abd", 16384, 106},
	{4, "acd", 531441, 0},
	{4, "bcd", 16384, 106},
	{4, "abcd", 2655552, 10792},
}

// maxShortCovers keeps go test -short fast, even with -race. TestRun still checks
// the total of 88 in that mode, without the comparisons.
const maxShortCovers = 10000

// changesNamed picks the changes for a test without going through the command.
func changesNamed(names string) []change {
	var allowed []change
	for _, c := range allChanges {
		if strings.Contains(names, string(c.name)) {
			allowed = append(allowed, c)
		}
	}
	return allowed
}

// independentRows finds the rows by trying every string of digits and keeping
// those with no repeated digit, unlike allRows, which builds them bell by bell.
func independentRows(bells int) []string {
	end := 1
	for i := 0; i < bells; i++ {
		end *= 10
	}
	var rows []string
	for number := end / 10; number < end; number++ {
		row := fmt.Sprint(number)
		seen := map[rune]bool{}
		for _, digit := range row {
			if digit >= '1' && int(digit-'0') <= bells {
				seen[digit] = true
			}
		}
		if len(seen) == bells {
			rows = append(rows, row)
		}
	}
	return rows
}

// isAllowed reads the rules of ringing off two rows, without using the change
// type: it finds the places that differ and names the change from them.
func isAllowed(from, to, names string) bool {
	var differ []int
	for i := range from {
		if from[i] != to[i] {
			differ = append(differ, i)
		}
	}
	name := ""
	switch fmt.Sprint(differ) {
	case "[0 1]":
		name = "a"
	case "[1 2]":
		name = "b"
	case "[2 3]":
		name = "c"
	case "[0 1 2 3]":
		name = "d"
	}
	if name == "" || !strings.Contains(names, name) {
		return false
	}
	for i := 0; i < len(differ); i += 2 {
		left, right := differ[i], differ[i+1]
		if from[left] != to[right] || from[right] != to[left] {
			return false
		}
	}
	return true
}

// neighbours lists, for each row, the indices of the rows it may change to.
func neighbours(rows []string, names string) [][]int {
	result := make([][]int, len(rows))
	for i, from := range rows {
		for j, to := range rows {
			if isAllowed(from, to, names) {
				result[i] = append(result[i], j)
			}
		}
	}
	return result
}

// walk finds every extent by walking the rows step by step, with no matrix and
// no cycle covers, and passes each one to visit as a joined string.
func walk(rows []string, names string, visit func(extent string)) {
	next := neighbours(rows, names)
	visited := make([]bool, len(rows))
	path := []string{rows[0]}
	var step func(at int)
	step = func(at int) {
		if len(path) == len(rows) {
			for _, to := range next[at] {
				if to == 0 {
					visit(strings.Join(append(path, rows[0]), " "))
				}
			}
			return
		}
		for _, to := range next[at] {
			if !visited[to] {
				visited[to] = true
				path = append(path, rows[to])
				step(to)
				path = path[:len(path)-1]
				visited[to] = false
			}
		}
	}
	visited[0] = true
	step(0)
}

// countCovers counts the cycle covers without a search: it goes through the
// rows in turn and keeps, for each set of rows already used as a target, how many
// ways there are to reach that set. This count is known as the permanent.
func countCovers(rows []string, names string) int {
	ways := map[uint32]int{0: 1}
	for _, targets := range neighbours(rows, names) {
		nextWays := map[uint32]int{}
		for used, count := range ways {
			for _, target := range targets {
				if bit := uint32(1) << target; used&bit == 0 {
					nextWays[used|bit] += count
				}
			}
		}
		ways = nextWays
	}
	total := 0
	for _, count := range ways {
		total += count
	}
	return total
}

func TestAllRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bells int
		want  int
	}{
		{name: "3 bells", bells: 3, want: 6},
		{name: "4 bells", bells: 4, want: 24},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := allRows(tt.bells)
			if len(got) != tt.want {
				t.Fatalf("got %d rows, want %d", len(got), tt.want)
			}
			if want := "1234"[:tt.bells]; got[0] != want {
				t.Errorf("first row = %s, want rounds %s", got[0], want)
			}
			sorted := append([]string{}, got...)
			sort.Strings(sorted)
			if want := independentRows(tt.bells); !reflect.DeepEqual(sorted, want) {
				t.Errorf("rows = %v, want %v", sorted, want)
			}
		})
	}
}

// TestSolve_Totals checks every total against a count of the cycle covers that
// uses no search, and a walk over the rows that uses no matrix. It compares the
// extents one by one and checks each against the rules of ringing.
func TestSolve_Totals(t *testing.T) {
	t.Parallel()

	for _, tt := range knownTotals {
		tt := tt
		t.Run(fmt.Sprintf("%d bells %s", tt.bells, tt.names), func(t *testing.T) {
			t.Parallel()
			if tt.covers > maxShortCovers && testing.Short() {
				t.Skip("slow: goes through every cycle cover")
			}

			rows := allRows(tt.bells)
			covers := 0
			got := map[string]bool{}
			solve(rows, changesNamed(tt.names), func(next []int) bool {
				covers++
				if extent := extentOf(rows, next); extent != nil {
					checkExtent(t, extent, len(rows), tt.names)
					got[strings.Join(extent, " ")] = true
				}
				return false
			})
			if covers != tt.covers {
				t.Errorf("found %d cycle covers, want %d", covers, tt.covers)
			}
			if len(got) != tt.extents {
				t.Errorf("found %d different extents, want %d", len(got), tt.extents)
			}

			independent := independentRows(tt.bells)
			if want := countCovers(independent, tt.names); covers != want {
				t.Errorf("found %d cycle covers, independent count gives %d", covers, want)
			}
			want := map[string]bool{}
			walk(independent, tt.names, func(extent string) { want[extent] = true })
			if !reflect.DeepEqual(got, want) {
				t.Errorf("found %d extents, the walk found %d others", len(got), len(want))
			}
		})
	}
}

// checkExtent applies the three rules of ringing to one extent.
func checkExtent(t *testing.T, extent []string, rowCount int, names string) {
	t.Helper()

	rounds := extent[0]
	if rounds != "1234"[:len(rounds)] {
		t.Errorf("extent starts with %s", rounds)
	}
	if last := extent[len(extent)-1]; last != rounds {
		t.Errorf("extent ends with %s, want %s", last, rounds)
	}
	seen := map[string]bool{}
	for i, row := range extent[1:] {
		if seen[row] {
			t.Errorf("row %s comes twice", row)
		}
		seen[row] = true
		if !isAllowed(extent[i], row, names) {
			t.Errorf("step %s to %s is not allowed", extent[i], row)
		}
	}
	if len(seen) != rowCount {
		t.Errorf("extent visits %d rows, want %d", len(seen), rowCount)
	}
}

func TestSolve_ThreeBellsCovers(t *testing.T) {
	t.Parallel()

	rows := allRows(3)
	var got [][]string
	solve(rows, changesNamed("ab"), func(next []int) bool {
		var steps []string
		for from, to := range next {
			steps = append(steps, rows[from]+">"+rows[to])
		}
		got = append(got, steps)
		return false
	})

	// The rows in order are 123 132 213 231 312 321, and the two changes join
	// them into one ring: 123 213 231 321 312 132.
	want := [][]string{
		{"123>213", "132>312", "213>123", "231>321", "312>132", "321>231"},
		{"123>213", "132>123", "213>231", "231>321", "312>132", "321>312"},
		{"123>132", "132>312", "213>123", "231>213", "312>321", "321>231"},
		{"123>132", "132>123", "213>231", "231>213", "312>321", "321>312"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("covers = %v, want %v", got, want)
	}
}

// TestUnreachableRows checks the reason the README gives for the sets of changes
// with no extent: starting from rounds, they cannot reach every row.
func TestUnreachableRows(t *testing.T) {
	t.Parallel()

	for _, tt := range knownTotals {
		tt := tt
		t.Run(fmt.Sprintf("%d bells %s", tt.bells, tt.names), func(t *testing.T) {
			t.Parallel()

			rows := independentRows(tt.bells)
			next := neighbours(rows, tt.names)
			reached := map[int]bool{0: true}
			queue := []int{0}
			for len(queue) > 0 {
				for _, to := range next[queue[0]] {
					if !reached[to] {
						reached[to] = true
						queue = append(queue, to)
					}
				}
				queue = queue[1:]
			}
			if isComplete := len(reached) == len(rows); isComplete != (tt.extents > 0) {
				t.Errorf("reached %d of %d rows, with %d extents", len(reached), len(rows), tt.extents)
			}
		})
	}
}

// TestBuildMatrix_RawCover checks the column numbers quoted in the README. The
// step from 123 to 213 leaves row 0 and arrives at row 2 of 6, so its columns
// are 0 and 6+2.
func TestBuildMatrix_RawCover(t *testing.T) {
	t.Parallel()

	found := false
	m := buildMatrix(allRows(3), changesNamed("ab"))
	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		for _, step := range exactCover {
			found = found || reflect.DeepEqual(step, []int{0, 8})
		}
		return false
	}))
	if !found {
		t.Error("no cover holds the step [0 8]")
	}
}

// TestPlainBobMinimus checks that a method rung by real ringers is among the
// extents: on 4 bells, repeat d b d b d b d c three times.
func TestPlainBobMinimus(t *testing.T) {
	t.Parallel()

	rows := allRows(4)
	allowed := changesNamed("bcd")
	byName := map[rune]change{}
	for _, c := range allowed {
		byName[rune(c.name)] = c
	}

	row := rows[0]
	var method []string
	method = append(method, row)
	for _, name := range strings.Repeat("dbdbdbdc", 3) {
		row = byName[name].apply(row)
		method = append(method, row)
	}
	checkExtent(t, method, len(rows), "bcd")

	want := strings.Join(method, " ")
	found := false
	solve(rows, allowed, func(next []int) bool {
		found = strings.Join(extentOf(rows, next), " ") == want
		return found
	})
	if !found {
		t.Errorf("extents do not include %s", want)
	}
}

func TestExtentOf(t *testing.T) {
	t.Parallel()

	rows := []string{"123", "132", "213", "231", "312", "321"}
	tests := []struct {
		name string
		next []int
		want []string
	}{
		{
			name: "one loop through every row",
			next: []int{2, 0, 3, 5, 1, 4},
			want: []string{"123", "213", "231", "321", "312", "132", "123"},
		},
		{name: "loops of two", next: []int{1, 0, 3, 2, 5, 4}},
		{name: "loop of three and a smaller one left over", next: []int{2, 4, 3, 0, 5, 1}},
		{name: "rounds in a loop of two, the rest in a longer loop", next: []int{1, 0, 3, 5, 2, 4}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := extentOf(rows, tt.next); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extentOf = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCoverLowerBounds proves the figures in the Limits section of the README.
// Changes a and c split the rows into groups of four, which can be covered
// independently of each other in 4 ways each, so there are at least 4 to the
// power of the number of groups. Any larger set of changes has at least as many.
func TestCoverLowerBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		bells      int
		wantGroups int
		wantDigits int
	}{
		{name: "4 bells", bells: 4, wantGroups: 6, wantDigits: 4},
		{name: "5 bells", bells: 5, wantGroups: 30, wantDigits: 19},
		{name: "6 bells", bells: 6, wantGroups: 180, wantDigits: 109},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seen := map[string]bool{}
			groups := 0
			for _, row := range allRows(tt.bells) {
				if seen[row] {
					continue
				}
				groups++
				for _, c := range []string{"", "a", "c", "ac"} {
					member := row
					for _, named := range changesNamed(c) {
						member = named.apply(member)
					}
					if seen[member] {
						t.Fatalf("row %s is in two groups", member)
					}
					seen[member] = true
				}
			}
			if groups != tt.wantGroups {
				t.Errorf("found %d groups, want %d", groups, tt.wantGroups)
			}
			power := new(big.Int).Exp(big.NewInt(4), big.NewInt(int64(groups)), nil)
			if digits := len(power.String()); digits != tt.wantDigits {
				t.Errorf("4^%d has %d digits, want %d", groups, digits, tt.wantDigits)
			}
		})
	}
}
