package dlx

// SolutionAccepter accepts solutions for the given exact cover problem.
// Each solution is passed to AcceptSolution. Solver stops immediately when AcceptSolution returns true.
type SolutionAccepter interface {
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

// Matrix represents the exact cover problem. It's a matrix consisting of 0s and 1s.
type Matrix interface {
	// AddRow adds a row to matrix.
	// Numbers are column indices (zero-based) for the nonzero elements of the row.
	AddRow(constraintsRow ...int)

	// Solve selects a subset of the rows so that the digit 1 appears
	// in each column exactly once.
	// Each solution is passed to AcceptSolution. Solver stops immediately
	// when AcceptSolution returns true.
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
}

// matrixObject is a toroidal doubly-linked list
type matrixObject struct {
	columns []columnObject
	head    *columnObject

	partialSolution []*dataObject
}

// NewMatrix creates the Matrix with the given number of columns.
func NewMatrix(nColumns int) Matrix {
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
	return &matrixObject{columns, head, nil}
}

// AddRow adds row of constraints
func (m *matrixObject) AddRow(constraintsRow ...int) {
	constraintsCount := len(constraintsRow)
	if constraintsCount == 0 {
		return
	}
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

// Solve finds a set or more of rows in which exactly one 1 appears for each column.
// Each solution is passed to accepter. It stops immediately when accepter AcceptSolution returns true.
func (m *matrixObject) Solve(accepter SolutionAccepter) {
	m.solve(accepter)
}

func (m *matrixObject) solve(accepter SolutionAccepter) bool {
	head := m.head
	headRight := head.right.column
	if headRight == head && accepter.AcceptSolution(m.getExactCover()) {
		return true
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

	for r := c.down; r != &c.dataObject; r = r.down {
		m.partialSolution[stackSize] = r

		for j := r.right; j != r; j = j.right {
			coverColumn(j.column)
		}

		if m.solve(accepter) {
			return true
		}

		for j := r.left; j != r; j = j.left {
			unCoverColumn(j.column)
		}
	}
	m.partialSolution = m.partialSolution[:stackSize]
	unCoverColumn(c)

	return false
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
