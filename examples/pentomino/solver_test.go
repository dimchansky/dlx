package main

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
	"testing"
)

// pieceCells is the test's own description of the pentominoes as (row, col)
// offsets, kept apart from the solver's table so that a typo in one is caught
// by the other.
var pieceCells = map[byte][][2]int{
	'F': {{0, 1}, {1, 0}, {1, 1}, {1, 2}, {2, 2}},
	'I': {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}},
	'L': {{0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 3}},
	'N': {{0, 0}, {0, 1}, {1, 1}, {1, 2}, {1, 3}},
	'P': {{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}},
	'T': {{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}},
	'U': {{0, 0}, {0, 2}, {1, 0}, {1, 1}, {1, 2}},
	'V': {{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}},
	'W': {{0, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 2}},
	'X': {{0, 1}, {1, 0}, {1, 1}, {1, 2}, {2, 1}},
	'Y': {{0, 2}, {1, 0}, {1, 1}, {1, 2}, {1, 3}},
	'Z': {{0, 0}, {0, 1}, {1, 1}, {2, 1}, {2, 2}},
}

// testOrientations lists the images of a piece under the eight symmetries of
// the square, as offsets from the piece's first cell in reading order. That
// cell is the one a left-to-right, top-to-bottom search places first.
func testOrientations(cells [][2]int) [][][2]int {
	maps := [][4]int{
		{1, 0, 0, 1}, {0, -1, 1, 0}, {-1, 0, 0, -1}, {0, 1, -1, 0},
		{-1, 0, 0, 1}, {1, 0, 0, -1}, {0, 1, 1, 0}, {0, -1, -1, 0},
	}
	seen := map[string]bool{}
	result := [][][2]int{}
	for _, m := range maps {
		image := make([][2]int, len(cells))
		for i, c := range cells {
			image[i] = [2]int{m[0]*c[0] + m[1]*c[1], m[2]*c[0] + m[3]*c[1]}
		}
		sort.Slice(image, func(i, j int) bool {
			if image[i][0] != image[j][0] {
				return image[i][0] < image[j][0]
			}
			return image[i][1] < image[j][1]
		})
		origin := image[0]
		for i := range image {
			image[i][0] -= origin[0]
			image[i][1] -= origin[1]
		}
		if key := fmt.Sprint(image); !seen[key] {
			seen[key] = true
			result = append(result, image)
		}
	}
	return result
}

// backtracker enumerates tilings without the exact cover machinery: it fills
// the first empty cell with every piece that fits there.
type backtracker struct {
	b       board
	letters []byte
	shapes  [][][][2]int
	tiling  []byte
	visit   func(tiling string)
}

func newBacktracker(b board, visit func(string)) *backtracker {
	bt := &backtracker{b: b, tiling: make([]byte, boardArea), visit: visit}
	for letter := range pieceCells {
		bt.letters = append(bt.letters, letter)
	}
	sort.Slice(bt.letters, func(i, j int) bool { return bt.letters[i] < bt.letters[j] })
	for _, letter := range bt.letters {
		bt.shapes = append(bt.shapes, testOrientations(pieceCells[letter]))
	}
	return bt
}

func (bt *backtracker) search(filled uint64, used uint16) {
	first := bits.TrailingZeros64(^filled)
	if first == boardArea {
		bt.visit(string(bt.tiling))
		return
	}
	row, col := first/bt.b.cols, first%bt.b.cols
	for p, shapes := range bt.shapes {
		if used&(1<<p) != 0 {
			continue
		}
		for _, shape := range shapes {
			if mask, ok := bt.fit(shape, row, col, filled); ok {
				bt.mark(mask, bt.letters[p])
				bt.search(filled|mask, used|1<<p)
			}
		}
	}
}

func (bt *backtracker) fit(shape [][2]int, row, col int, filled uint64) (uint64, bool) {
	var mask uint64
	for _, offset := range shape {
		r, c := row+offset[0], col+offset[1]
		if r < 0 || r >= bt.b.rows || c < 0 || c >= bt.b.cols {
			return 0, false
		}
		bit := uint64(1) << (r*bt.b.cols + c)
		if filled&bit != 0 {
			return 0, false
		}
		mask |= bit
	}
	return mask, true
}

func (bt *backtracker) mark(mask uint64, letter byte) {
	for ; mask != 0; mask &= mask - 1 {
		bt.tiling[bits.TrailingZeros64(mask)] = letter
	}
}

// transpose turns a tiling of the transposed board into a tiling of b.
func transpose(b board, tiling string) string {
	result := make([]byte, len(tiling))
	for row := 0; row < b.rows; row++ {
		for col := 0; col < b.cols; col++ {
			result[row*b.cols+col] = tiling[col*b.rows+row]
		}
	}
	return string(result)
}

func TestParseBoard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		size    string
		want    board
		wantErr string
	}{
		{name: "default size", size: "6x10", want: board{rows: 6, cols: 10}},
		{name: "rows first", size: "10x6", want: board{rows: 10, cols: 6}},
		{name: "single row", size: "1x60", want: board{rows: 1, cols: 60}},
		{name: "no separator", size: "60", wantErr: "want rows and columns"},
		{name: "not a number", size: "ax10", wantErr: "invalid syntax"},
		{name: "bad columns", size: "6xb", wantErr: "invalid syntax"},
		{name: "wrong area", size: "6x6", wantErr: "must have 60 cells"},
		{name: "negative factors", size: "-6x-10", wantErr: "must have 60 cells"},
		{name: "zero", size: "0x60", wantErr: "must have 60 cells"},
		{
			// These multiply to 60 when int wraps around at 64 bits.
			name:    "overflowing factors",
			size:    "4611686018427387908x15",
			wantErr: "must have 60 cells",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseBoard(tt.size)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestOrientations(t *testing.T) {
	t.Parallel()

	want := map[byte]int{
		'F': 8, 'I': 2, 'L': 8, 'N': 8, 'P': 8, 'T': 4,
		'U': 4, 'V': 4, 'W': 4, 'X': 1, 'Y': 8, 'Z': 4,
	}
	for _, piece := range pieces {
		shape := parseShape(piece.shape)
		if len(shape) != 5 {
			t.Errorf("piece %c has %d cells, want 5", piece.letter, len(shape))
		}
		if got := len(orientations(shape)); got != want[piece.letter] {
			t.Errorf("piece %c has %d orientations, want %d", piece.letter, got, want[piece.letter])
		}
		wantShapes := len(testOrientations(pieceCells[piece.letter]))
		if got := len(orientations(shape)); got != wantShapes {
			t.Errorf("piece %c has %d orientations, the test table has %d", piece.letter, got, wantShapes)
		}
	}
}

func TestPlacementCount(t *testing.T) {
	t.Parallel()

	b := board{rows: 6, cols: 10}
	placements := 0
	for _, piece := range pieces {
		for _, orientation := range orientations(parseShape(piece.shape)) {
			height, width := extent(orientation)
			placements += (b.rows - height + 1) * (b.cols - width + 1)
		}
	}
	if placements != 2056 {
		t.Errorf("6x10 board has %d placements, want 2056", placements)
	}
}

// TestSolve_KnownTotals checks the published totals, and that an independent
// search finds the same number. Each total counts all rotations and reflections
// of the whole board.
func TestSolve_KnownTotals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		size         string
		wantTotal    int
		wantDistinct int
		isSlow       bool
	}{
		{size: "1x60"},
		{size: "2x30"},
		{size: "3x20", wantTotal: 8, wantDistinct: 2},
		{size: "4x15", wantTotal: 1472, wantDistinct: 368, isSlow: true},
		{size: "5x12", wantTotal: 4040, wantDistinct: 1010, isSlow: true},
		{size: "6x10", wantTotal: 9356, wantDistinct: 2339, isSlow: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.size, func(t *testing.T) {
			t.Parallel()
			if tt.isSlow && testing.Short() {
				t.Skip("slow: enumerates every tiling")
			}

			b, err := parseBoard(tt.size)
			if err != nil {
				t.Fatal(err)
			}

			total, distinct := 0, 0
			found := map[string]bool{}
			solve(b, func(tiling string) bool {
				total++
				found[tiling] = true
				if isCanonical(b, tiling) {
					distinct++
				}
				return false
			})
			if total != tt.wantTotal || distinct != tt.wantDistinct {
				t.Errorf("found %d tilings, %d distinct, want %d and %d",
					total, distinct, tt.wantTotal, tt.wantDistinct)
			}
			if len(found) != total {
				t.Errorf("found %d different tilings among %d", len(found), total)
			}

			// Filling across the short side of the board is far faster, so the
			// independent search works on the transposed board.
			independent := 0
			newBacktracker(board{rows: b.cols, cols: b.rows}, func(tiling string) {
				independent++
				if !found[transpose(b, tiling)] {
					t.Errorf("independent search found a tiling that solve did not:\n%s", tiling)
				}
			}).search(0, 0)
			if independent != total {
				t.Errorf("independent search found %d tilings, solve found %d", independent, total)
			}
		})
	}
}

func TestSolve_StopsWhenAccepted(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(board{rows: 6, cols: 10}, func(string) bool {
		calls++
		return calls == 3
	})
	if calls != 3 {
		t.Errorf("accepter called %d times, want 3", calls)
	}
}

func TestIsCanonical(t *testing.T) {
	t.Parallel()

	b := board{rows: 3, cols: 20}
	solve(b, func(tiling string) bool {
		images := []string{
			tiling,
			flipTiling(b, tiling, true, false),
			flipTiling(b, tiling, false, true),
			flipTiling(b, tiling, true, true),
		}
		canonical := 0
		seen := map[string]bool{}
		for _, image := range images {
			seen[image] = true
			if isCanonical(b, image) {
				canonical++
			}
		}
		if len(seen) != 4 || canonical != 1 {
			t.Errorf("%d different images and %d canonical, want 4 and 1:\n%s", len(seen), canonical, tiling)
		}
		return false
	})
}
