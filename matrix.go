package dlx

import "fmt"

// SolutionAccepter receives the solutions of an exact cover problem.
type SolutionAccepter interface {
	// AcceptSolution receives one solution and returns true to stop the search.
	//
	// exactCover holds one element per selected row, each listing the row's
	// column indices in the order they were passed to AddRow. The rows appear
	// in the order the search selected them, not in AddRow order. Both levels
	// of the slice are freshly allocated, so it is safe to retain and modify.
	//
	// AcceptSolution must not call AddRow or Solve on the matrix being solved.
	// If it panics, the matrix is left in an undefined state and must be
	// discarded.
	AcceptSolution(exactCover [][]int) bool
}

// The SolutionAccepterFunc type is an adapter to allow the use of
// ordinary functions as SolutionAccepter. If f is a function with the appropriate signature,
// SolutionAccepterFunc(f) is a SolutionAccepter that calls f.
type SolutionAccepterFunc func([][]int) bool

// AcceptSolution calls f(exactCover) and returns its result.
func (f SolutionAccepterFunc) AcceptSolution(exactCover [][]int) bool {
	return f(exactCover)
}

// Matrix represents an exact cover problem: a matrix of 0s and 1s, for which
// Solve looks for a subset of rows with exactly one 1 in every column.
//
// A Matrix is not safe for concurrent use, including concurrent calls to Solve.
type Matrix interface {
	// AddRow adds a row to the matrix. constraintsRow lists the zero-based
	// column indices of the row's 1s, in any order. The slice is not retained.
	//
	// A row with no indices is ignored: it could not cover anything. Rows have
	// no identity, so a row added twice is two rows and yields separate
	// solutions.
	//
	// AddRow panics if an index is outside [0, nColumns) or appears twice in
	// constraintsRow. The matrix is unchanged after such a panic.
	AddRow(constraintsRow ...int)

	// Solve searches for every subset of rows that has exactly one 1 in each
	// column, and passes each one to accepter. The search ends when accepter
	// returns true or when no solutions remain.
	//
	// A matrix with no columns has exactly one solution, which is empty. A
	// matrix with a column that no row covers has none.
	//
	// The order of the search is deterministic: matrices built by the same
	// sequence of AddRow calls report the same solutions in the same order.
	// It branches on the leftmost column with the fewest remaining rows, trying
	// that column's rows in AddRow order.
	//
	// Solve restores the matrix before returning, even after accepter stops the
	// search, so it can be called again and more rows can be added. Recursion
	// depth is bounded by the number of columns.
	//
	// Solve panics with a "dlx:" message if accepter is a nil interface value.
	// A nil SolutionAccepterFunc is not detected: it panics with a runtime error
	// when the first solution is reported, like any panicking accepter.
	Solve(accepter SolutionAccepter)
}

type dataObject struct {
	column                *columnObject
	up, down, left, right *dataObject

	rowStart *dataObject
}

type columnObject struct {
	dataObject
	size  int
	index int
	// lastRow is the serial of the last row that AddRow validated against this
	// column. A row that meets its own serial here names the column twice.
	lastRow uint64
}

// matrixObject is a toroidal doubly-linked list
type matrixObject struct {
	columns []columnObject
	head    *columnObject

	// rowSerial identifies the most recent AddRow call that had a nonempty row.
	rowSerial uint64

	partialSolution []*dataObject
}

// NewMatrix creates the Matrix with the given number of columns.
// It panics with a "dlx:" message if nColumns is negative. An extremely large
// nColumns fails with a runtime allocation panic instead.
func NewMatrix(nColumns int) Matrix {
	if nColumns < 0 {
		panic(fmt.Sprintf("dlx: NewMatrix: negative column count %d", nColumns))
	}

	columns := make([]columnObject, nColumns+1)
	// initializing head
	head := &columns[0]
	headDO := &head.dataObject
	head.column = head
	head.left = &columns[nColumns].dataObject
	head.up = headDO
	head.down = headDO
	head.index = -1
	// last column points to head
	columns[nColumns].right = headDO
	// initializing column headers
	prevColumn := head
	for i := range columns[1:] {
		column := &columns[i+1]
		columnDO := &column.dataObject
		column.index = i
		column.column = column
		column.up = columnDO
		column.down = columnDO
		column.left = &prevColumn.dataObject
		prevColumn.right = columnDO
		prevColumn = column
	}
	return &matrixObject{columns: columns, head: head}
}

// AddRow implements Matrix.
func (m *matrixObject) AddRow(constraintsRow ...int) {
	constraintsCount := len(constraintsRow)
	if constraintsCount == 0 {
		return
	}
	m.validateRow(constraintsRow)

	rowDOs := make([]dataObject, constraintsCount)
	rowStart := &rowDOs[0]
	for i, c := range constraintsRow {
		// increment column size
		column := &m.columns[c+1]
		column.size++
		// get the neighbors
		prevUpDO := column.up
		downDO := &column.dataObject
		leftDO := &rowDOs[(i+constraintsCount-1)%constraintsCount]
		rightDO := &rowDOs[(i+1)%constraintsCount]
		// create data object for constraint
		do := &rowDOs[i]
		do.column = column
		do.up = prevUpDO
		do.down = downDO
		do.left = leftDO
		do.right = rightDO
		do.rowStart = rowStart
		// change links of the neighbors
		prevUpDO.down, downDO.up, leftDO.right, rightDO.left = do, do, do, do
	}
}

// validateRow panics unless row holds distinct valid column indices. It runs
// before any linking, so a rejected row leaves no trace in the matrix: the
// lastRow marks it may leave behind differ from every later serial.
func (m *matrixObject) validateRow(row []int) {
	nColumns := len(m.columns) - 1
	m.rowSerial++
	for _, c := range row {
		if c < 0 || c >= nColumns {
			panic(fmt.Sprintf("dlx: AddRow: column %d out of range [0, %d)", c, nColumns))
		}
		column := &m.columns[c+1]
		if column.lastRow == m.rowSerial {
			panic(fmt.Sprintf("dlx: AddRow: duplicate column %d", c))
		}
		column.lastRow = m.rowSerial
	}
}

// Solve implements Matrix.
func (m *matrixObject) Solve(accepter SolutionAccepter) {
	if accepter == nil {
		panic("dlx: Solve: nil SolutionAccepter")
	}
	m.solve(accepter)
}

// solve reports whether the accepter asked to stop. It undoes all its covering
// before returning, so a stop leaves the matrix as intact as an exhausted search.
func (m *matrixObject) solve(accepter SolutionAccepter) bool {
	head := m.head
	headRight := head.right.column
	if headRight == head {
		return accepter.AcceptSolution(m.getExactCover())
	}

	c := headRight
	minSize := headRight.size

	for jc := headRight.right.column; jc != head; jc = jc.right.column {
		jSize := jc.size
		if jSize >= minSize {
			continue
		}
		c, minSize = jc, jSize
	}

	coverColumn(c)

	stackSize := len(m.partialSolution)
	m.partialSolution = append(m.partialSolution, nil)

	stopped := false
	for r := c.down; r != &c.dataObject && !stopped; r = r.down {
		m.partialSolution[stackSize] = r

		for j := r.right; j != r; j = j.right {
			coverColumn(j.column)
		}

		stopped = m.solve(accepter)

		for j := r.left; j != r; j = j.left {
			unCoverColumn(j.column)
		}
	}
	m.partialSolution = m.partialSolution[:stackSize]
	unCoverColumn(c)

	return stopped
}

func (m *matrixObject) getExactCover() [][]int {
	ec := make([][]int, len(m.partialSolution))

	for i, do := range m.partialSolution {
		// transform row to constraints row
		row := ec[i]
		rowStart := do.rowStart
		row = append(row, rowStart.column.index)
		for j := rowStart.right; j != rowStart; j = j.right {
			row = append(row, j.column.index)
		}
		ec[i] = row
	}

	return ec
}

func coverColumn(c *columnObject) {
	cDO := &c.dataObject
	c.right.left, c.left.right = c.left, c.right
	for i := c.down; i != cDO; i = i.down {
		for j := i.right; j != i; j = j.right {
			j.down.up, j.up.down = j.up, j.down
			j.column.size--
		}
	}
}

func unCoverColumn(c *columnObject) {
	cDO := &c.dataObject
	for i := c.up; i != cDO; i = i.up {
		for j := i.left; j != i; j = j.left {
			j.column.size++
			j.down.up, j.up.down = j, j
		}
	}
	c.right.left, c.left.right = cDO, cDO
}
