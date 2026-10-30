package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dimchansky/dlx"
)

// pieceCount is the number of pentominoes, and so the number of columns that
// come before the cell columns in the matrix.
const pieceCount = 12

// boardArea is the number of cells that 12 pieces of 5 cells each can fill.
const boardArea = pieceCount * 5

// pieces holds each pentomino in one orientation, with its usual letter.
var pieces = [pieceCount]struct {
	letter byte
	shape  []string
}{
	{'F', []string{".XX", "XX.", ".X."}},
	{'I', []string{"XXXXX"}},
	{'L', []string{"X.", "X.", "X.", "XX"}},
	{'N', []string{".X", ".X", "XX", "X."}},
	{'P', []string{"XX", "XX", "X."}},
	{'T', []string{"XXX", ".X.", ".X."}},
	{'U', []string{"X.X", "XXX"}},
	{'V', []string{"X..", "X..", "XXX"}},
	{'W', []string{"X..", "XX.", ".XX"}},
	{'X', []string{".X.", "XXX", ".X."}},
	{'Y', []string{".X", "XX", ".X", ".X"}},
	{'Z', []string{"XX.", ".X.", ".XX"}},
}

// cell is a position on the board.
type cell struct {
	row, col int
}

// board is a rectangle of boardArea cells.
type board struct {
	rows, cols int
}

// parseBoard reads a size such as "6x10", rows first.
func parseBoard(size string) (board, error) {
	rowsText, colsText, found := strings.Cut(size, "x")
	if !found {
		return board{}, fmt.Errorf("invalid size %q: want rows and columns, such as 6x10", size)
	}
	rows, err := strconv.Atoi(rowsText)
	if err != nil {
		return board{}, fmt.Errorf("invalid size %q: %w", size, err)
	}
	cols, err := strconv.Atoi(colsText)
	if err != nil {
		return board{}, fmt.Errorf("invalid size %q: %w", size, err)
	}
	// Checking the ranges first keeps the product from overflowing.
	inRange := rows >= 1 && rows <= boardArea && cols >= 1 && cols <= boardArea
	if !inRange || rows*cols != boardArea {
		return board{}, fmt.Errorf("invalid size %q: the board must have %d cells", size, boardArea)
	}
	return board{rows: rows, cols: cols}, nil
}

// solve passes each tiling of b to accept until accept returns true. A tiling
// is the piece letter of every cell, row by row. All symmetric images of a
// tiling are reported, each as a separate tiling.
func solve(b board, accept func(tiling string) bool) {
	buildMatrix(b).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(b, exactCover))
	}))
}

// buildMatrix has a column for each piece, then a column for each cell. Each
// row is one placement of one piece: it covers that piece's column and the
// columns of the five cells it occupies. So every piece is used exactly once
// and every cell is filled exactly once.
func buildMatrix(b board) dlx.Matrix {
	m := dlx.NewMatrix(pieceCount + b.rows*b.cols)
	for pieceIndex, piece := range pieces {
		for _, orientation := range orientations(parseShape(piece.shape)) {
			addPlacements(m, b, pieceIndex, orientation)
		}
	}
	return m
}

// addPlacements adds a row for each position of an orientation on the board.
func addPlacements(m dlx.Matrix, b board, pieceIndex int, orientation []cell) {
	height, width := extent(orientation)
	for row := 0; row+height <= b.rows; row++ {
		for col := 0; col+width <= b.cols; col++ {
			columns := []int{pieceIndex}
			for _, c := range orientation {
				columns = append(columns, pieceCount+(row+c.row)*b.cols+col+c.col)
			}
			m.AddRow(columns...)
		}
	}
}

// decode relies on dlx keeping each row's columns in AddRow order: the piece
// comes first, then its cells.
func decode(b board, exactCover [][]int) string {
	tiling := make([]byte, b.rows*b.cols)
	for _, row := range exactCover {
		for _, column := range row[1:] {
			tiling[column-pieceCount] = pieces[row[0]].letter
		}
	}
	return string(tiling)
}

func parseShape(shape []string) []cell {
	cells := []cell{}
	for row, line := range shape {
		for col, symbol := range line {
			if symbol == 'X' {
				cells = append(cells, cell{row: row, col: col})
			}
		}
	}
	return cells
}

// orientations returns the distinct rotations and reflections of a shape.
func orientations(shape []cell) [][]cell {
	seen := map[string]bool{}
	result := [][]cell{}
	for _, start := range [][]cell{shape, reflect(shape)} {
		current := start
		for turn := 0; turn < 4; turn++ {
			current = rotate(current)
			normal := normalize(current)
			key := fmt.Sprint(normal)
			if !seen[key] {
				seen[key] = true
				result = append(result, normal)
			}
		}
	}
	return result
}

func rotate(shape []cell) []cell {
	rotated := make([]cell, len(shape))
	for i, c := range shape {
		rotated[i] = cell{row: c.col, col: -c.row}
	}
	return rotated
}

func reflect(shape []cell) []cell {
	reflected := make([]cell, len(shape))
	for i, c := range shape {
		reflected[i] = cell{row: c.row, col: -c.col}
	}
	return reflected
}

// normalize moves a shape against the top and left edges and sorts its cells,
// so that equal shapes compare equal.
func normalize(shape []cell) []cell {
	minRow, minCol := shape[0].row, shape[0].col
	for _, c := range shape {
		if c.row < minRow {
			minRow = c.row
		}
		if c.col < minCol {
			minCol = c.col
		}
	}
	normal := make([]cell, len(shape))
	for i, c := range shape {
		normal[i] = cell{row: c.row - minRow, col: c.col - minCol}
	}
	sort.Slice(normal, func(i, j int) bool {
		if normal[i].row != normal[j].row {
			return normal[i].row < normal[j].row
		}
		return normal[i].col < normal[j].col
	})
	return normal
}

// extent returns the height and width of a normalized shape.
func extent(shape []cell) (height, width int) {
	for _, c := range shape {
		if c.row >= height {
			height = c.row + 1
		}
		if c.col >= width {
			width = c.col + 1
		}
	}
	return height, width
}

// isCanonical reports whether tiling is the smallest of its images under the
// board's symmetries: the two mirror flips and their combination, a half turn.
// Each tiling has exactly four different images, because the F piece has no
// symmetry and appears only once, so no flip can map a tiling to itself. The
// board is never square, which rules out quarter turns. Exactly one image in
// each group of four is therefore canonical.
func isCanonical(b board, tiling string) bool {
	flips := []struct{ rows, cols bool }{
		{rows: true},
		{cols: true},
		{rows: true, cols: true},
	}
	for _, flip := range flips {
		if flipTiling(b, tiling, flip.rows, flip.cols) < tiling {
			return false
		}
	}
	return true
}

func flipTiling(b board, tiling string, flipRows, flipCols bool) string {
	flipped := make([]byte, len(tiling))
	for row := 0; row < b.rows; row++ {
		for col := 0; col < b.cols; col++ {
			imageRow, imageCol := row, col
			if flipRows {
				imageRow = b.rows - 1 - row
			}
			if flipCols {
				imageCol = b.cols - 1 - col
			}
			flipped[imageRow*b.cols+imageCol] = tiling[row*b.cols+col]
		}
	}
	return string(flipped)
}
