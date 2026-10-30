package main

import (
	"fmt"
	"testing"
)

// allCodes returns every code of the given length.
func allCodes(length int) [][]int {
	var codes [][]int
	solve(2, length, func(code []int) bool {
		codes = append(codes, code)
		return false
	})
	return codes
}

// TestPlayAll_EveryCode plays the strategy of every code. The wins need no
// solver: the team loses exactly on the code words.
func TestPlayAll_EveryCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		length   int
		patterns int
		codes    int
		wins     int
		guesses  int // right guesses, and also wrong guesses
	}{
		{length: 1, patterns: 2, codes: 2, wins: 1, guesses: 1},
		{length: 3, patterns: 8, codes: 4, wins: 6, guesses: 6},
		{length: 7, patterns: 128, codes: 240, wins: 112, guesses: 112},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("n=%d", tt.length), func(t *testing.T) {
			t.Parallel()

			codes := allCodes(tt.length)
			if len(codes) != tt.codes {
				t.Fatalf("got %d codes, want %d", len(codes), tt.codes)
			}
			want := outcome{wins: tt.wins, losses: tt.patterns - tt.wins, right: tt.guesses, wrong: tt.guesses}
			for _, code := range codes {
				if got := playAll(code, tt.length); got != want {
					t.Errorf("code %v: got %+v, want %+v", code, got, want)
				}
			}
		})
	}
}

// TestGuess_IgnoresOwnHat proves the strategy is one prisoners can really follow:
// changing a prisoner's own hat never changes what that prisoner does.
func TestGuess_IgnoresOwnHat(t *testing.T) {
	t.Parallel()

	const length = 7
	code := allCodes(length)[0]
	isCodeWord := membership(code, 1<<length)
	for hats := 0; hats < 1<<length; hats++ {
		for prisoner := 0; prisoner < length; prisoner++ {
			white, whiteSpeaks := guess(isCodeWord, withHat(hats, length, prisoner, 0), length, prisoner)
			black, blackSpeaks := guess(isCodeWord, withHat(hats, length, prisoner, 1), length, prisoner)
			if white != black || whiteSpeaks != blackSpeaks {
				t.Fatalf("hats %07b prisoner %d: answer depends on own hat", hats, prisoner)
			}
		}
	}
}

// bestWinsForThree tries every possible plan of three prisoners and returns the
// most patterns any plan wins. A prisoner sees two hats, so four sights, and can
// answer each with pass, white or black: 3^4 = 81 plans each, and 81^3 together.
// It knows nothing about codes.
func bestWinsForThree() int {
	const pass, plans = 2, 81
	answer := func(plan, sight int) int {
		for ; sight > 0; sight-- {
			plan /= 3
		}
		return plan % 3
	}
	best := 0
	for planA := 0; planA < plans; planA++ {
		for planB := 0; planB < plans; planB++ {
			for planC := 0; planC < plans; planC++ {
				wins := 0
				for hats := 0; hats < 8; hats++ {
					a, b, c := hats>>2&1, hats>>1&1, hats&1
					right, wrong := 0, 0
					for _, turn := range [][2]int{
						{answer(planA, b*2+c), a},
						{answer(planB, a*2+c), b},
						{answer(planC, a*2+b), c},
					} {
						switch turn[0] {
						case pass:
						case turn[1]:
							right++
						default:
							wrong++
						}
					}
					if right > 0 && wrong == 0 {
						wins++
					}
				}
				if wins > best {
					best = wins
				}
			}
		}
	}
	return best
}

func TestPlayAll_NoPlanBeatsCodeForThree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow: tries every plan")
	}

	code := allCodes(3)[0]
	if got, want := bestWinsForThree(), playAll(code, 3).wins; got != want {
		t.Errorf("best plan wins %d patterns, the code wins %d", got, want)
	}
}
