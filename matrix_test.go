package dlx

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

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

// A complete cover must end the recursion, otherwise it continues past the
// report and pushes a row that does not exist onto the partial solution.
func TestSolve_StopsSearchingAtCompleteCover(t *testing.T) {
	t.Parallel()

	m := newTestMatrix(0, nil)
	m.Solve(SolutionAccepterFunc(func([][]int) bool { return false }))

	if got := cap(m.partialSolution); got != 0 {
		t.Errorf("partial solution grew to capacity %d after reporting a complete cover", got)
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
