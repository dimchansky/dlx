package main

import (
	"fmt"
	"math/bits"
	"sort"
	"testing"

	"github.com/dimchansky/dlx"
)

// distance counts the digits in which two words differ.
func distance(a, b, base, length int) int {
	differ := 0
	for i := 0; i < length; i++ {
		if a%base != b%base {
			differ++
		}
		a /= base
		b /= base
	}
	return differ
}

// isPerfect checks the definition directly, without neighbourhood or the matrix:
// every word must be at distance 0 or 1 from exactly one code word.
func isPerfect(code []int, base, length int) bool {
	for word := 0; word < wordCount(base, length); word++ {
		near := 0
		for _, codeWord := range code {
			if distance(word, codeWord, base, length) <= 1 {
				near++
			}
		}
		if near != 1 {
			return false
		}
	}
	return true
}

// collect enumerates every code and returns them as a set of printed codes.
func collect(t *testing.T, base, length int) map[string]bool {
	t.Helper()
	found := map[string]bool{}
	total := 0
	solve(base, length, func(code []int) bool {
		total++
		found[fmt.Sprint(code)] = true
		if !isPerfect(code, base, length) {
			t.Errorf("not a perfect code: %v", code)
		}
		return false
	})
	if len(found) != total {
		t.Errorf("found %d different codes among %d", len(found), total)
	}
	return found
}

func TestSolve_Totals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		base, length int
		wantCodes    int
		wantWords    int
	}{
		{name: "one bit", base: 2, length: 1, wantCodes: 2, wantWords: 1},
		{name: "two bits", base: 2, length: 2},
		{name: "three bits", base: 2, length: 3, wantCodes: 4, wantWords: 2},
		{name: "four bits", base: 2, length: 4},
		{name: "five bits", base: 2, length: 5},
		{name: "six bits", base: 2, length: 6},
		{name: "seven bits", base: 2, length: 7, wantCodes: 240, wantWords: 16},
		{name: "one ternary digit", base: 3, length: 1, wantCodes: 3, wantWords: 1},
		{name: "two ternary digits", base: 3, length: 2},
		{name: "three ternary digits", base: 3, length: 3},
		{name: "four ternary digits", base: 3, length: 4, wantCodes: 72, wantWords: 9},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if codes := len(collect(t, tt.base, tt.length)); codes != tt.wantCodes {
				t.Errorf("found %d codes, want %d", codes, tt.wantCodes)
			}
			solve(tt.base, tt.length, func(code []int) bool {
				if len(code) != tt.wantWords {
					t.Errorf("code has %d words, want %d", len(code), tt.wantWords)
				}
				return false
			})
		})
	}
}

// TestSizesFit_AgreesWithSearch shows the shortcut in solve loses nothing: the
// matrix itself has no exact cover whenever sizesFit says so. Lengths 11, 13 and 14
// in base 2 and 7 and 8 in base 3 are left out: each takes from half a minute to
// more than three minutes.
func TestSizesFit_AgreesWithSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		base, length int
		slow         bool // skipped with -short
	}{
		{base: 2, length: 1},
		{base: 2, length: 2},
		{base: 2, length: 3},
		{base: 2, length: 4},
		{base: 2, length: 5},
		{base: 2, length: 6},
		{base: 2, length: 7},
		{base: 2, length: 8},
		{base: 2, length: 9, slow: true},
		{base: 2, length: 10, slow: true},
		{base: 2, length: 12, slow: true},
		{base: 3, length: 1},
		{base: 3, length: 2},
		{base: 3, length: 3},
		{base: 3, length: 4},
		{base: 3, length: 5},
		{base: 3, length: 6, slow: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("base %d length %d", tt.base, tt.length), func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: searches the whole matrix")
			}

			covers := 0
			buildMatrix(tt.base, tt.length).Solve(dlx.SolutionAccepterFunc(func([][]int) bool {
				covers++
				return false
			}))
			if !sizesFit(tt.base, tt.length) && covers != 0 {
				t.Errorf("sizes don't fit, but %d codes exist", covers)
			}
		})
	}
}

// binaryHammingCodes builds every code of length 7 without any search, with the
// classic recipe. Label the seven positions 1 to 7 in some order. Compute a sum of
// the labels of a word's 1 digits, in binary addition without carries (exclusive
// or). All words with the same sum form a code, one code for each of the 8
// possible sums. Every order of labels is tried.
func binaryHammingCodes() map[string]bool {
	const length, words = 7, 128
	unique := map[[16]int]bool{}
	labels := []int{1, 2, 3, 4, 5, 6, 7}
	forEachOrder(labels, 0, func() {
		var bySum [8][16]int
		var filled [8]int
		for word := 0; word < words; word++ {
			sum := 0
			for position := 0; position < length; position++ {
				if word>>position&1 == 1 {
					sum ^= labels[position]
				}
			}
			bySum[sum][filled[sum]] = word
			filled[sum]++
		}
		for _, code := range bySum {
			unique[code] = true
		}
	})

	codes := map[string]bool{}
	for code := range unique {
		codes[fmt.Sprint(code[:])] = true
	}
	return codes
}

// forEachOrder calls visit with items rearranged into every order.
func forEachOrder(items []int, from int, visit func()) {
	if from == len(items) {
		visit()
		return
	}
	for i := from; i < len(items); i++ {
		items[from], items[i] = items[i], items[from]
		forEachOrder(items, from+1, visit)
		items[from], items[i] = items[i], items[from]
	}
}

// ternaryCodes builds every code of length 4 without any search. It starts from
// the nine words c, d, c+d, c-d, with digits 0 to 2 and sums that wrap around
// after 2. It then rearranges the positions, swaps the digits 1 and 2 at any set
// of positions, and adds one fixed word to all nine.
func ternaryCodes() map[string]bool {
	const length, shifts = 4, 81
	var seed [9][length]int
	for c := 0; c < 3; c++ {
		for d := 0; d < 3; d++ {
			seed[3*c+d] = [length]int{c, d, (c + d) % 3, (c - d + 3) % 3}
		}
	}

	// shape applies one arrangement to the seed.
	shape := func(order []int, swaps int) [9]int {
		var code [9]int
		for i, word := range seed {
			number, place := 0, 1
			for position := 0; position < length; position++ {
				digit := word[order[position]]
				if swaps>>position&1 == 1 {
					digit = (3 - digit) % 3
				}
				number += digit * place
				place *= 3
			}
			code[i] = number
		}
		sort.Ints(code[:])
		return code
	}

	unshifted := map[[9]int]bool{}
	order := []int{0, 1, 2, 3}
	forEachOrder(order, 0, func() {
		for swaps := 0; swaps < 16; swaps++ {
			unshifted[shape(order, swaps)] = true
		}
	})

	codes := map[string]bool{}
	for code := range unshifted {
		for shift := 0; shift < shifts; shift++ {
			var shifted [9]int
			for i, word := range code {
				shifted[i] = addDigits(word, shift, 3, length)
			}
			sort.Ints(shifted[:])
			codes[fmt.Sprint(shifted[:])] = true
		}
	}
	return codes
}

// addDigits adds two words digit by digit, with no carrying.
func addDigits(a, b, base, length int) int {
	sum, place := 0, 1
	for i := 0; i < length; i++ {
		sum += (a%base + b%base) % base * place
		a /= base
		b /= base
		place *= base
	}
	return sum
}

// TestSolve_MatchesConstruction compares the search with codes built by hand. The
// builders share nothing with the solver: no neighbourhoods and no matrix.
func TestSolve_MatchesConstruction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		base, length int
		construct    func() map[string]bool
		wantCodes    int
	}{
		{name: "binary length 7", base: 2, length: 7, construct: binaryHammingCodes, wantCodes: 240},
		{name: "ternary length 4", base: 3, length: 4, construct: ternaryCodes, wantCodes: 72},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			built := tt.construct()
			if len(built) != tt.wantCodes {
				t.Fatalf("built %d codes, want %d", len(built), tt.wantCodes)
			}
			found := collect(t, tt.base, tt.length)
			for code := range built {
				if !found[code] {
					t.Errorf("search missed %s", code)
				}
			}
			for code := range found {
				if !built[code] {
					t.Errorf("search found unknown code %s", code)
				}
			}
		})
	}
}

// countCodesWithZero counts the sets of size words that contain word 0 and whose
// words all differ pairwise in 3 digits or more. It tries such sets in rising
// order and knows nothing about neighbourhoods. Such a set is a perfect code when
// size times the neighbourhood size is the number of words, because neighbourhoods
// of words that far apart can't overlap.
func countCodesWithZero(base, length, size int) int {
	words := wordCount(base, length)
	farApart := make([][]bool, words)
	for word := range farApart {
		farApart[word] = make([]bool, words)
		for other := range farApart[word] {
			farApart[word][other] = distance(word, other, base, length) >= 3
		}
	}

	var extend func(candidates []int, need int) int
	extend = func(candidates []int, need int) int {
		if need == 0 {
			return 1
		}
		total := 0
		for i, word := range candidates {
			if len(candidates)-i < need {
				break
			}
			var next []int
			for _, other := range candidates[i+1:] {
				if farApart[word][other] {
					next = append(next, other)
				}
			}
			total += extend(next, need-1)
		}
		return total
	}

	var farFromZero []int
	for word := 1; word < words; word++ {
		if farApart[0][word] {
			farFromZero = append(farFromZero, word)
		}
	}
	return extend(farFromZero, size-1)
}

// TestCountCodesWithZero is the check that no perfect code exists beyond those
// the search reports. Adding a fixed word to every word, digit by digit with wrap
// around, turns a code into a code, so each word lies in the same number of codes.
// Counting (codes)*(words in a code) word by word gives
// codes = (codes with word 0) * (all words) / (words in a code).
func TestCountCodesWithZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		base, length int
		size         int
		slow         bool // skipped with -short
		wantWithZero int
		wantCodes    int
	}{
		{name: "binary length 1", base: 2, length: 1, size: 1, wantWithZero: 1, wantCodes: 2},
		{name: "binary length 3", base: 2, length: 3, size: 2, wantWithZero: 1, wantCodes: 4},
		{name: "binary length 7", base: 2, length: 7, size: 16, slow: true, wantWithZero: 30, wantCodes: 240},
		{name: "ternary length 1", base: 3, length: 1, size: 1, wantWithZero: 1, wantCodes: 3},
		{name: "ternary length 4", base: 3, length: 4, size: 9, wantWithZero: 8, wantCodes: 72},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: tries every set of far apart words")
			}

			withZero := countCodesWithZero(tt.base, tt.length, tt.size)
			if withZero != tt.wantWithZero {
				t.Errorf("got %d codes with word 0, want %d", withZero, tt.wantWithZero)
			}
			if got := withZero * wordCount(tt.base, tt.length) / tt.size; got != tt.wantCodes {
				t.Errorf("got %d codes, want %d", got, tt.wantCodes)
			}
		})
	}
}

// TestHammingLength15 shows a code of length 15 exists, which the search does not
// find in useful time. The recipe of binaryHammingCodes makes one directly, and the
// check is the definition, sharing no code with the solver.
func TestHammingLength15(t *testing.T) {
	t.Parallel()

	const length = 15
	var code []int
	for word := 0; word < 1<<length; word++ {
		sum := 0
		for position := 0; position < length; position++ {
			if word>>position&1 == 1 {
				sum ^= position + 1
			}
		}
		if sum == 0 {
			code = append(code, word)
		}
	}
	if len(code) != 2048 {
		t.Errorf("got %d code words, want 2048", len(code))
	}
	for word := 0; word < 1<<length; word++ {
		near := 0
		for _, codeWord := range code {
			if bits.OnesCount(uint(word^codeWord)) <= 1 {
				near++
			}
		}
		if near != 1 {
			t.Fatalf("word %d is near %d code words, want 1", word, near)
		}
	}
}

func TestNeighbourhood(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		base, length, word int
		want               string
	}{
		{name: "binary 000", base: 2, length: 3, word: 0, want: "[0 1 2 4]"},
		{name: "binary 101", base: 2, length: 3, word: 5, want: "[5 4 7 1]"},
		{name: "binary 0000111", base: 2, length: 7, word: 7, want: "[7 6 5 3 15 23 39 71]"},
		{name: "ternary 12", base: 3, length: 2, word: 5, want: "[5 3 4 2 8]"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := fmt.Sprint(neighbourhood(tt.base, tt.length, tt.word)); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
