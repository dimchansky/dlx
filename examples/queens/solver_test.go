package main

import (
	"fmt"
	"testing"
)

// knownCounts holds the published n-queens totals for n = 1 to 12.
var knownCounts = []int{1, 0, 0, 2, 10, 4, 40, 92, 352, 724, 2680, 14200}

// backtrack enumerates solutions without the exact cover machinery: it places a
// queen row by row and tracks attacked columns and diagonals in bit masks.
func backtrack(n int, visit func(queens []int)) {
	queens := make([]int, n)
	var place func(row int, cols, diagonals, antidiagonals uint64)
	place = func(row int, cols, diagonals, antidiagonals uint64) {
		if row == n {
			visit(queens)
			return
		}
		for col := 0; col < n; col++ {
			colBit := uint64(1) << col
			diagonalBit := uint64(1) << (row + col)
			antidiagonalBit := uint64(1) << (row - col + n - 1)
			isAttacked := cols&colBit != 0 || diagonals&diagonalBit != 0 || antidiagonals&antidiagonalBit != 0
			if isAttacked {
				continue
			}
			queens[row] = col
			place(row+1, cols|colBit, diagonals|diagonalBit, antidiagonals|antidiagonalBit)
		}
	}
	place(0, 0, 0, 0)
}

func isValid(queens []int) bool {
	for i := range queens {
		for j := i + 1; j < len(queens); j++ {
			isSameCol := queens[i] == queens[j]
			isSameDiagonal := queens[i]+i == queens[j]+j
			isSameAntidiagonal := queens[i]-i == queens[j]-j
			if isSameCol || isSameDiagonal || isSameAntidiagonal {
				return false
			}
		}
	}
	return true
}

// TestSolve_KnownCounts checks the published totals. For each size it also
// checks every solution and compares them with an independent search, which
// shows that slack rows neither lose solutions nor report one twice.
func TestSolve_KnownCounts(t *testing.T) {
	t.Parallel()

	for i, want := range knownCounts {
		n, want := i+1, want
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			if n > 10 && raceEnabled {
				t.Skip("slow with -race")
			}

			found := map[string]bool{}
			total := 0
			solve(n, func(queens []int) bool {
				total++
				found[fmt.Sprint(queens)] = true
				if !isValid(queens) {
					t.Errorf("invalid solution %v", queens)
				}
				return false
			})
			if total != want {
				t.Errorf("found %d solutions, want %d", total, want)
			}
			if len(found) != total {
				t.Errorf("found %d different solutions among %d", len(found), total)
			}

			independent := 0
			backtrack(n, func(queens []int) {
				independent++
				if !found[fmt.Sprint(queens)] {
					t.Errorf("independent search found %v, which solve did not", queens)
				}
			})
			if independent != total {
				t.Errorf("independent search found %d solutions, solve found %d", independent, total)
			}
		})
	}
}

func TestSolve_StopsWhenAccepted(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(8, func([]int) bool {
		calls++
		return calls == 3
	})
	if calls != 3 {
		t.Errorf("accepter called %d times, want 3", calls)
	}
}

func TestSolve_LargerBoardFirstSolution(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(20, func(queens []int) bool {
		calls++
		if !isValid(queens) {
			t.Errorf("invalid solution %v", queens)
		}
		return true
	})
	if calls != 1 {
		t.Errorf("accepter called %d times, want 1", calls)
	}
}
