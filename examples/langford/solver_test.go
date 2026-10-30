package main

import (
	"fmt"
	"testing"
)

// knownCounts holds the published totals for n = 1 to 12, counting an
// arrangement and its reversal as one.
var knownCounts = []int{0, 0, 1, 1, 0, 0, 26, 150, 0, 0, 17792, 108144}

// backtrack enumerates all arrangements without the exact cover machinery: it
// tries every pair of positions for each number in turn, largest first.
func backtrack(n int, visit func(sequence []int)) {
	sequence := make([]int, 2*n)
	var place func(number int)
	place = func(number int) {
		if number == 0 {
			visit(sequence)
			return
		}
		for first := 0; first+number+1 < len(sequence); first++ {
			second := first + number + 1
			if sequence[first] != 0 || sequence[second] != 0 {
				continue
			}
			sequence[first], sequence[second] = number, number
			place(number - 1)
			sequence[first], sequence[second] = 0, 0
		}
	}
	place(n)
}

func isValid(n int, sequence []int) bool {
	if len(sequence) != 2*n {
		return false
	}
	firstSeen := make([]int, n+1)
	copies := make([]int, n+1)
	for position, number := range sequence {
		if number < 1 || number > n {
			return false
		}
		copies[number]++
		switch copies[number] {
		case 1:
			firstSeen[number] = position
		case 2:
			if position-firstSeen[number] != number+1 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func reversed(sequence []int) []int {
	result := make([]int, len(sequence))
	for i, number := range sequence {
		result[len(sequence)-1-i] = number
	}
	return result
}

// TestSolve_KnownCounts checks the published totals against both matrices, and
// compares the solutions with an independent search. It also checks the claim
// that removing rows keeps exactly one of each mirror pair.
func TestSolve_KnownCounts(t *testing.T) {
	t.Parallel()

	for i, want := range knownCounts {
		n, want := i+1, want
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			if n > 10 && raceEnabled {
				t.Skip("slow with -race")
			}

			all := map[string]bool{}
			solve(n, true, func(sequence []int) bool {
				all[fmt.Sprint(sequence)] = true
				if !isValid(n, sequence) {
					t.Errorf("invalid arrangement %v", sequence)
				}
				return false
			})
			if len(all) != 2*want {
				t.Errorf("found %d arrangements with mirrors, want %d", len(all), 2*want)
			}

			kept := map[string]bool{}
			solve(n, false, func(sequence []int) bool {
				key, mirror := fmt.Sprint(sequence), fmt.Sprint(reversed(sequence))
				if key == mirror {
					t.Errorf("arrangement %v is its own mirror", sequence)
				}
				if kept[key] || kept[mirror] {
					t.Errorf("arrangement %v or its mirror was reported twice", sequence)
				}
				if !all[key] || !all[mirror] {
					t.Errorf("arrangement %v or its mirror is missing from the full set", sequence)
				}
				kept[key] = true
				return false
			})
			if len(kept) != want {
				t.Errorf("found %d arrangements, want %d", len(kept), want)
			}

			independent := 0
			backtrack(n, func(sequence []int) {
				independent++
				if !all[fmt.Sprint(sequence)] {
					t.Errorf("independent search found %v, which solve did not", sequence)
				}
			})
			if independent != len(all) {
				t.Errorf("independent search found %d arrangements, solve found %d", independent, len(all))
			}
		})
	}
}

func TestSolve_StopsWhenAccepted(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(7, false, func([]int) bool {
		calls++
		return calls == 3
	})
	if calls != 3 {
		t.Errorf("accepter called %d times, want 3", calls)
	}
}
