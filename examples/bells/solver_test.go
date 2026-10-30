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
// The other totals have no published source, so TestIndependentTotals and
// TestSolve_Totals prove them against two separate methods.
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

// maxShortCovers keeps go test -short fast, even with -race, by leaving out the
// solver runs above it. maxShortCount does the same for countCovers, whose map is
// slow under -race on abcd. The walk still checks every total of extents.
const (
	maxShortCovers = 1000
	maxShortCount  = 1000000
)

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

// independent holds independentRows for each number of bells, built once because
// nearly every subtest needs it.
var independent = map[int][]string{3: independentRows(3), 4: independentRows(4)}

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
	w := walker{
		rows:    rows,
		next:    neighbours(rows, names),
		visited: make([]bool, len(rows)),
		path:    make([]int, len(rows)),
		visit:   visit,
	}
	w.visited[0] = true
	w.step(1)
}

// walker is the state of one walk, which stays on one goroutine.
type walker struct {
	rows    []string
	next    [][]int
	visited []bool
	path    []int
	visit   func(extent string)
}

// step extends the path from its first depth rows. It is marked norace because
// the walk on abcd takes millions of steps, which would cost about a second under
// -race, and it shares nothing with another goroutine.
//
//go:norace
func (w *walker) step(depth int) {
	at := w.path[depth-1]
	if depth == len(w.rows) {
		for _, to := range w.next[at] {
			if to == 0 {
				w.visit(joinPath(w.rows, w.path))
			}
		}
		return
	}
	for _, to := range w.next[at] {
		if !w.visited[to] {
			w.visited[to] = true
			w.path[depth] = to
			w.step(depth + 1)
			w.visited[to] = false
		}
	}
}

// joinPath writes out a walk through the rows, with rounds again at the end.
func joinPath(rows []string, path []int) string {
	extent := make([]string, 0, len(path)+1)
	for _, i := range path {
		extent = append(extent, rows[i])
	}
	return strings.Join(append(extent, rows[0]), " ")
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
			if want := independent[tt.bells]; !reflect.DeepEqual(sorted, want) {
				t.Errorf("rows = %v, want %v", sorted, want)
			}
		})
	}
}

// TestSolve_Totals checks the solver against a count of the cycle covers that
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

			rowsAgain := independent[tt.bells]
			if want := countCovers(rowsAgain, tt.names); covers != want {
				t.Errorf("found %d cycle covers, independent count gives %d", covers, want)
			}
			want := map[string]bool{}
			walk(rowsAgain, tt.names, func(extent string) { want[extent] = true })
			if !reflect.DeepEqual(got, want) {
				t.Errorf("found %d extents, the walk found %d others", len(got), len(want))
			}
		})
	}
}

// TestIndependentTotals checks the totals, the published one included, with
// neither the solver nor a search through the covers. Under -short it leaves out
// only the count for abcd, so the walk still checks the 10792.
func TestIndependentTotals(t *testing.T) {
	t.Parallel()

	for _, tt := range knownTotals {
		tt := tt
		t.Run(fmt.Sprintf("%d bells %s", tt.bells, tt.names), func(t *testing.T) {
			t.Parallel()

			rows := independent[tt.bells]
			if tt.covers <= maxShortCount || !testing.Short() {
				if got := countCovers(rows, tt.names); got != tt.covers {
					t.Errorf("the count found %d cycle covers, want %d", got, tt.covers)
				}
			}
			extents := 0
			walk(rows, tt.names, func(string) { extents++ })
			if extents != tt.extents {
				t.Errorf("the walk found %d extents, want %d", extents, tt.extents)
			}
		})
	}
}

// timeMatrix builds the larger matrix from the Limits section of the README, the
// one that holds the single loop rule itself. With n rows of bells, its columns are:
//
//	u                  a step leaves row u
//	n + t*n + u        row u at time t is dealt with on the way out
//	n + n*n + t*n + u  row u at time t is dealt with on the way in
//
// A step from u to v at time t covers the first, the second, and the third for v
// at time t+1. A slack row covers the second and third for one u and t, and means
// that the ringing isn't at u at time t. Only rounds is at time 0, and rounds is at
// no other time, so one walker sets off and every row of bells is on its path.
func timeMatrix(rows []string, names string) (m dlx.Matrix, steps, slack int) {
	n := len(rows)
	next := neighbours(rows, names)
	out := func(t, u int) int { return n + t*n + u }
	in := func(t, u int) int { return n + n*n + t*n + u }

	m = dlx.NewMatrix(n + 2*n*n)
	for t := 0; t < n; t++ {
		for u := 0; u < n; u++ {
			if t != 0 || u != 0 {
				m.AddRow(out(t, u), in(t, u))
				slack++
			}
			if (t == 0) != (u == 0) {
				continue
			}
			arrival := (t + 1) % n
			for _, v := range next[u] {
				if (arrival == 0) == (v == 0) {
					m.AddRow(u, out(t, u), in(arrival, v))
					steps++
				}
			}
		}
	}
	return m, steps, slack
}

// TestTimeMatrix checks the README's claim that a matrix can hold the single loop
// rule: with no filter at all, its solutions are exactly the extents of the walk.
func TestTimeMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		bells       int
		names       string
		wantColumns int
		wantSteps   int
		wantSlack   int
		slow        bool
	}{
		{name: "3 bells", bells: 3, names: "ab", wantColumns: 78, wantSteps: 36, wantSlack: 35},
		{name: "4 bells", bells: 4, names: "abc", wantColumns: 1176, wantSteps: 1458, wantSlack: 575, slow: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: the matrix is much larger than the one in solver.go")
			}

			rows := independent[tt.bells]
			n := len(rows)
			if columns := n + 2*n*n; columns != tt.wantColumns {
				t.Errorf("%d columns, want %d", columns, tt.wantColumns)
			}
			m, steps, slack := timeMatrix(rows, tt.names)
			if steps != tt.wantSteps || slack != tt.wantSlack {
				t.Errorf("%d steps and %d slack rows, want %d and %d", steps, slack, tt.wantSteps, tt.wantSlack)
			}

			got := map[string]bool{}
			m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
				extent := make([]string, n+1)
				extent[n] = rows[0]
				for _, row := range exactCover {
					if len(row) == 3 {
						extent[(row[1]-n)/n] = rows[row[0]]
					}
				}
				got[strings.Join(extent, " ")] = true
				return false
			}))

			want := map[string]bool{}
			walk(rows, tt.names, func(extent string) { want[extent] = true })
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

			rows := independent[tt.bells]
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
// extents: on 4 bells, repeat d b d b d b d c three times. That is the place
// notation -14-14,12 of method 10460 on CompLib, https://complib.org/method/10460.
func TestPlainBobMinimus(t *testing.T) {
	t.Parallel()

	rows := allRows(4)
	allowed := changesNamed("bcd")
	byName := map[rune]change{}
	for _, c := range allowed {
		byName[rune(c.name)] = c
	}

	index := map[string]int{}
	for i, row := range rows {
		index[row] = i
	}

	row := rows[0]
	var method []string
	method = append(method, row)
	next := make([]int, len(rows))
	for _, name := range strings.Repeat("dbdbdbdc", 3) {
		from := row
		row = byName[name].apply(row)
		method = append(method, row)
		next[index[from]] = index[row]
	}
	checkExtent(t, method, len(rows), "bcd")
	if got := extentOf(rows, next); !reflect.DeepEqual(got, method) {
		t.Errorf("extentOf = %v, want %v", got, method)
	}
	if testing.Short() {
		return
	}

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

// TestCoverLowerBounds checks the figures in the Limits section of the README.
// Changes a and c split the rows into groups of four. The solver covers every
// group on its own in 4 ways, and a step with a or c never leaves a group, so
// there are at least 4 to the power of the number of groups. Any larger set of
// changes has at least as many.
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

			split := changesNamed("ac")
			seen := map[string]bool{}
			groups := 0
			for _, row := range allRows(tt.bells) {
				if seen[row] {
					continue
				}
				groups++
				var group []string
				for _, c := range []string{"", "a", "c", "ac"} {
					member := row
					for _, named := range changesNamed(c) {
						member = named.apply(member)
					}
					if seen[member] {
						t.Fatalf("row %s is in two groups", member)
					}
					seen[member] = true
					group = append(group, member)
				}
				covers := 0
				solve(group, split, func([]int) bool {
					covers++
					return false
				})
				if covers != 4 {
					t.Errorf("group %v has %d covers, want 4", group, covers)
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
