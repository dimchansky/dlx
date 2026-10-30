package dlx

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestNewMatrix_NegativeColumnsPanics(t *testing.T) {
	t.Parallel()

	assertPanicsWith(t, "dlx: NewMatrix: negative column count -1", func() { NewMatrix(-1) })
}

func TestAddRow_InvalidRowPanics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nColumns int
		row      []int
		want     string
	}{
		{
			name:     "negative index",
			nColumns: 3,
			row:      []int{-1},
			want:     "dlx: AddRow: column -1 out of range [0, 3)",
		},
		{
			name:     "lowest index",
			nColumns: 3,
			row:      []int{math.MinInt},
			want:     fmt.Sprintf("dlx: AddRow: column %d out of range [0, 3)", math.MinInt),
		},
		{
			name:     "index equal to column count",
			nColumns: 3,
			row:      []int{3},
			want:     "dlx: AddRow: column 3 out of range [0, 3)",
		},
		{
			name:     "highest index",
			nColumns: 3,
			row:      []int{math.MaxInt},
			want:     fmt.Sprintf("dlx: AddRow: column %d out of range [0, 3)", math.MaxInt),
		},
		{
			name:     "invalid index after valid ones",
			nColumns: 3,
			row:      []int{0, 2, 7},
			want:     "dlx: AddRow: column 7 out of range [0, 3)",
		},
		{
			name:     "any index without columns",
			nColumns: 0,
			row:      []int{0},
			want:     "dlx: AddRow: column 0 out of range [0, 0)",
		},
		{
			name:     "adjacent duplicates",
			nColumns: 3,
			row:      []int{0, 0},
			want:     "dlx: AddRow: duplicate column 0",
		},
		{
			name:     "separated duplicates",
			nColumns: 3,
			row:      []int{2, 0, 1, 2},
			want:     "dlx: AddRow: duplicate column 2",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var rows [][]int
			if tt.nColumns > 0 {
				rows = [][]int{{0, 1, 2}, {0}, {1, 2}}
			}
			m := newTestMatrix(tt.nColumns, rows)
			before := takeSnapshot(t, m)
			want := toString(allSolutions(m))

			assertPanicsWith(t, tt.want, func() { m.AddRow(tt.row...) })

			assertUnchanged(t, m, before)
			if got := toString(allSolutions(m)); got != want {
				t.Errorf("solutions changed after the panic\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestAddRow_UsableAfterPanic(t *testing.T) {
	t.Parallel()

	m := newTestMatrix(2, nil)
	assertPanicsWith(t, "dlx: AddRow: duplicate column 1", func() { m.AddRow(0, 1, 1) })
	m.AddRow(0, 1)

	got := toString(allSolutions(m))
	want := toString([][][]int{{{0, 1}}})
	if got != want {
		t.Errorf("wrong solutions\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAddRow_DoesNotRetainRow(t *testing.T) {
	t.Parallel()

	row := []int{0, 1}
	m := newTestMatrix(2, nil)
	m.AddRow(row...)
	row[0], row[1] = 1, 0

	got := toString(allSolutions(m))
	want := toString([][][]int{{{0, 1}}})
	if got != want {
		t.Errorf("wrong solutions\ngot:\n%s\nwant:\n%s", got, want)
	}
}

var solveTests = []struct {
	name      string
	nColumns  int
	rows      [][]int
	solutions [][][]int
}{
	{
		name:     "no columns has one empty solution",
		nColumns: 0,
		solutions: [][][]int{
			{},
		},
	},
	{
		name:     "column without rows has no solutions",
		nColumns: 2,
		rows: [][]int{
			{0},
		},
	},
	{
		name:     "rows cover only some columns",
		nColumns: 4,
		rows: [][]int{
			{2, 3},
		},
	},
	{
		name:     "single solution",
		nColumns: 7,
		rows: [][]int{
			{2, 4, 5},
			{0, 3, 6},
			{1, 2, 5},
			{0, 3},
			{1, 6},
			{3, 4, 6},
		},
		solutions: [][][]int{
			{
				{2, 4, 5},
				{0, 3},
				{1, 6},
			},
		},
	},
	{
		name:     "overlapping rows have no solutions",
		nColumns: 4,
		rows: [][]int{
			{0, 1},
			{0, 2},
			{1, 2},
		},
	},
	{
		name:     "several solutions",
		nColumns: 4,
		rows: [][]int{
			{0, 1, 2},
			{0, 2},
			{1},
			{3},
		},
		solutions: [][][]int{
			{
				{0, 1, 2},
				{3},
			},
			{
				{0, 2},
				{1},
				{3},
			},
		},
	},
	{
		name:     "empty rows are ignored",
		nColumns: 1,
		rows: [][]int{
			{},
			{0},
			{},
		},
		solutions: [][][]int{
			{
				{0},
			},
		},
	},
	{
		name:     "duplicate rows give separate solutions",
		nColumns: 2,
		rows: [][]int{
			{0, 1},
			{0, 1},
		},
		solutions: [][][]int{
			{
				{0, 1},
			},
			{
				{0, 1},
			},
		},
	},
	{
		name:     "row values keep their AddRow order",
		nColumns: 3,
		rows: [][]int{
			{2, 0, 1},
		},
		solutions: [][][]int{
			{
				{2, 0, 1},
			},
		},
	},
}

func TestSolve(t *testing.T) {
	t.Parallel()

	for _, tt := range solveTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toString(allSolutions(newTestMatrix(tt.nColumns, tt.rows)))
			if want := toString(tt.solutions); got != want {
				t.Errorf("wrong solutions\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestSolve_RepeatsAfterEarlyStop(t *testing.T) {
	t.Parallel()

	for _, tt := range solveTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertSolveRestores(t, newTestMatrix(tt.nColumns, tt.rows))
		})
	}
}

func TestSolve_AddRowAfterEarlyStop(t *testing.T) {
	t.Parallel()

	m := newTestMatrix(2, [][]int{{0}, {1}})
	m.Solve(SolutionAccepterFunc(func([][]int) bool { return true }))
	m.AddRow(0, 1)

	got := toString(allSolutions(m))
	want := toString([][][]int{
		{{0}, {1}},
		{{0, 1}},
	})
	if got != want {
		t.Errorf("wrong solutions\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSolve_NilAccepterPanics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nColumns int
	}{
		{name: "matrix with a solution", nColumns: 0},
		{name: "matrix without solutions", nColumns: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMatrix(tt.nColumns, nil)
			assertPanicsWith(t, "dlx: Solve: nil SolutionAccepter", func() { m.Solve(nil) })
		})
	}
}

func TestSolve_NilSolutionAccepterFunc(t *testing.T) {
	t.Parallel()

	var accepter SolutionAccepterFunc

	t.Run("not called without solutions", func(t *testing.T) {
		t.Parallel()

		newTestMatrix(1, nil).Solve(accepter)
	})

	t.Run("runtime panic at the first solution", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if _, isRuntimeError := recover().(runtime.Error); !isRuntimeError {
				t.Error("want a runtime error panic")
			}
		}()
		newTestMatrix(0, nil).Solve(accepter)
	})
}

func TestSolve_SolutionsAreSafeToRetain(t *testing.T) {
	t.Parallel()

	m := newTestMatrix(2, [][]int{{0, 1}, {0, 1}})
	want := toString(allSolutions(m))

	var retained [][][]int
	m.Solve(SolutionAccepterFunc(func(exactCover [][]int) bool {
		retained = append(retained, exactCover)
		return false
	}))
	if got := toString(retained); got != want {
		t.Errorf("retained solutions changed\ngot:\n%s\nwant:\n%s", got, want)
	}

	retained[0][0][0] = 42
	if got := toString(allSolutions(m)); got != want {
		t.Errorf("changing a retained solution changed the matrix\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSolve_MatchesBruteForce(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(1))
	matricesWithSolutions := 0
	for i := 0; i < 3000; i++ {
		nColumns := rng.Intn(maxBruteForceColumns + 1)
		rows := randomRows(rng, nColumns, rng.Intn(maxBruteForceRows+1))

		want := bruteForce(nColumns, rows)
		if len(want) > 0 {
			matricesWithSolutions++
		}

		m := newTestMatrix(nColumns, rows)
		if got, want := toString(allSolutions(m)), toString(want); got != want {
			t.Fatalf("matrix %d, %d columns, rows %v\ngot:\n%s\nwant:\n%s", i, nColumns, rows, got, want)
		}
		assertSolveRestores(t, m)
	}

	// Guards the generator: a run with hardly any solutions would prove nothing.
	if matricesWithSolutions < 300 {
		t.Errorf("only %d matrices had solutions", matricesWithSolutions)
	}
}

func TestSolve_InterleavedWithAddRow(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 500; i++ {
		nColumns := 1 + rng.Intn(maxBruteForceColumns)
		rows := randomRows(rng, nColumns, maxBruteForceRows)

		m := newTestMatrix(nColumns, nil)
		for added, row := range rows {
			m.AddRow(row...)

			got := toString(allSolutions(m))
			want := toString(bruteForce(nColumns, rows[:added+1]))
			if got != want {
				t.Fatalf("matrix %d after %d rows of %v\ngot:\n%s\nwant:\n%s", i, added+1, rows, got, want)
			}
			m.Solve(SolutionAccepterFunc(func([][]int) bool { return true }))
		}
	}
}

func FuzzSolve(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{3, 0b001, 0b010, 0b100})
	f.Add([]byte{4, 0b0111, 0b0101, 0b0010, 0b1000})
	f.Add([]byte{8, 0xff, 0xff, 0x0f, 0xf0, 0x33, 0xcc, 0x55, 0xaa, 0, 1, 2, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		nColumns, rows := decodeMatrix(data)
		m := newTestMatrix(nColumns, rows)

		got := toString(allSolutions(m))
		want := toString(bruteForce(nColumns, rows))
		if got != want {
			t.Fatalf("%d columns, rows %v\ngot:\n%s\nwant:\n%s", nColumns, rows, got, want)
		}
		assertSolveRestores(t, m)
	})
}

// The brute force solver tries 2^rows subsets, which keeps these small.
const (
	maxBruteForceColumns = 8
	maxBruteForceRows    = 12
)

// bruteForce solves an exact cover problem by trying every subset of rows. It
// shares no code with the solver under test. Empty rows are left out because
// AddRow ignores them; they would otherwise double the solutions.
func bruteForce(nColumns int, rows [][]int) [][][]int {
	nonEmpty := make([][]int, 0, len(rows))
	for _, row := range rows {
		if len(row) > 0 {
			nonEmpty = append(nonEmpty, row)
		}
	}

	solutions := [][][]int{}
	for subset := uint(0); subset < 1<<uint(len(nonEmpty)); subset++ {
		solution := [][]int{}
		covered, isExact := uint(0), true
		for i, row := range nonEmpty {
			if subset>>uint(i)&1 == 0 {
				continue
			}
			solution = append(solution, row)
			for _, c := range row {
				isExact = isExact && covered>>uint(c)&1 == 0
				covered |= 1 << uint(c)
			}
		}
		if isExact && bits.OnesCount(covered) == nColumns {
			solutions = append(solutions, solution)
		}
	}
	return solutions
}

// randomRows returns rows of distinct columns in random order. Some are empty.
func randomRows(rng *rand.Rand, nColumns, nRows int) [][]int {
	rows := make([][]int, 0, nRows)
	for i := 0; i < nRows; i++ {
		rows = append(rows, rng.Perm(nColumns)[:rng.Intn(nColumns+1)])
	}
	return rows
}

// decodeMatrix turns fuzz input into a small matrix. The first byte selects the
// column count and each later byte is a bitmask of the columns of one row. A row
// starts at a column that depends on its position, so that rows are not always
// in ascending order.
func decodeMatrix(data []byte) (nColumns int, rows [][]int) {
	if len(data) == 0 {
		return 0, nil
	}

	nColumns = int(data[0]) % (maxBruteForceColumns + 1)
	masks := data[1:]
	if len(masks) > maxBruteForceRows {
		masks = masks[:maxBruteForceRows]
	}
	if nColumns == 0 {
		return 0, nil
	}

	for i, mask := range masks {
		row := []int{}
		for offset := 0; offset < nColumns; offset++ {
			if c := (i + offset) % nColumns; mask>>uint(c)&1 == 1 {
				row = append(row, c)
			}
		}
		rows = append(rows, row)
	}
	return nColumns, rows
}

// newTestMatrix returns the concrete matrix, so that tests can inspect its links.
func newTestMatrix(nColumns int, rows [][]int) *matrixObject {
	m, ok := NewMatrix(nColumns).(*matrixObject)
	if !ok {
		panic("NewMatrix did not return *matrixObject")
	}
	for _, row := range rows {
		m.AddRow(row...)
	}
	return m
}

// allSolutions returns every solution in the order Solve reports them.
func allSolutions(m Matrix) [][][]int {
	solutions := [][][]int{}
	m.Solve(SolutionAccepterFunc(func(exactCover [][]int) bool {
		solutions = append(solutions, exactCover)
		return false
	}))
	return solutions
}

// assertSolveRestores stops Solve at every possible solution and checks that
// each stop leaves the links intact and the same solutions, in the same order,
// for the next Solve.
func assertSolveRestores(t *testing.T, m *matrixObject) {
	t.Helper()

	before := takeSnapshot(t, m)
	want := fmt.Sprint(allSolutions(m))
	assertUnchanged(t, m, before)

	for stopAfter, total := 1, len(allSolutions(m)); stopAfter <= total; stopAfter++ {
		calls := 0
		m.Solve(SolutionAccepterFunc(func([][]int) bool {
			calls++
			return calls == stopAfter
		}))
		if calls != stopAfter {
			t.Fatalf("accepter called %d times after asking to stop at solution %d", calls, stopAfter)
		}
		assertUnchanged(t, m, before)

		if got := fmt.Sprint(allSolutions(m)); got != want {
			t.Fatalf("solutions changed after a stop at solution %d\ngot:  %s\nwant: %s", stopAfter, got, want)
		}
	}
}

func assertUnchanged(t *testing.T, m *matrixObject, want snapshot) {
	t.Helper()

	got := takeSnapshot(t, m)
	if !got.equal(want) {
		t.Fatal("matrix links changed")
	}
}

func assertPanicsWith(t *testing.T, want string, f func()) {
	t.Helper()

	defer func() {
		t.Helper()

		got := recover()
		if got == nil {
			t.Fatalf("no panic, want %q", want)
		}
		if got != want {
			t.Fatalf("panic value %#v, want %q", got, want)
		}
	}()
	f()
}

// snapshot records every link of a matrix, and the sizes of its columns.
type snapshot struct {
	nodes           []dataObject
	sizes           []int
	partialSolution int
}

func takeSnapshot(t *testing.T, m *matrixObject) snapshot {
	t.Helper()

	s := snapshot{partialSolution: len(m.partialSolution)}
	limit := len(m.columns)
	for i := range m.columns {
		limit += m.columns[i].size
	}
	for i := range m.columns {
		column := &m.columns[i]
		s.sizes = append(s.sizes, column.size)
		s.nodes = append(s.nodes, column.dataObject)
		for n := column.down; n != &column.dataObject; n = n.down {
			if len(s.nodes) > limit {
				t.Fatalf("column %d does not link back to its header", i-1)
			}
			s.nodes = append(s.nodes, *n)
		}
	}
	return s
}

func (s snapshot) equal(other snapshot) bool {
	if s.partialSolution != other.partialSolution || len(s.nodes) != len(other.nodes) {
		return false
	}
	for i := range s.nodes {
		if s.nodes[i] != other.nodes[i] {
			return false
		}
	}
	for i := range s.sizes {
		if s.sizes[i] != other.sizes[i] {
			return false
		}
	}
	return true
}

// toString returns a form of solutions that ignores the order of the solutions
// and of the rows within each solution, but not the order of the row values.
func toString(solutions [][][]int) string {
	solutionsStrs := []string{}
	for _, solution := range solutions {
		solutionStrs := []string{}
		for _, row := range solution {
			solutionStrs = append(solutionStrs, fmt.Sprintf("  %v", row))
		}
		sort.Strings(solutionStrs)
		solutionsStrs = append(solutionsStrs, "[\n"+strings.Join(solutionStrs, ",\n")+"\n]")
	}
	sort.Strings(solutionsStrs)
	return strings.Join(solutionsStrs, ",\n")
}
