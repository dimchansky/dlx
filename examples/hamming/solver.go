package main

import (
	"sort"

	"github.com/dimchansky/dlx"
)

// solve passes each perfect code to accept until accept returns true. A code is
// a sorted list of word numbers. Words have length digits in the given base and
// are numbered by reading them as numbers in that base, so in base 2 the word
// 101 is number 5.
func solve(base, length int, accept func(code []int) bool) {
	if !sizesFit(base, length) {
		return
	}
	buildMatrix(base, length).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(exactCover))
	}))
}

// sizesFit reports whether the words could be shared out into equal
// neighbourhoods at all. The neighbourhoods of a perfect code don't overlap and
// together hold every word, so the neighbourhood size must divide the number of
// words. The search proves the same thing quickly up to length 10 in base 2, but
// takes over a minute for length 14 and over three minutes for lengths 11 and 13.
func sizesFit(base, length int) bool {
	return wordCount(base, length)%neighbourhoodSize(base, length) == 0
}

// wordCount returns base to the power length.
func wordCount(base, length int) int {
	count := 1
	for i := 0; i < length; i++ {
		count *= base
	}
	return count
}

// neighbourhoodSize counts a word and the words one digit change away from it:
// each of the length digits can change to base-1 other values.
func neighbourhoodSize(base, length int) int {
	return 1 + length*(base-1)
}

// neighbourhood lists word first, then every word that differs from it in
// exactly one digit.
func neighbourhood(base, length, word int) []int {
	words := []int{word}
	place := 1 // the value of the digit being changed: 1, base, base*base, ...
	for i := 0; i < length; i++ {
		digit := word / place % base
		for other := 0; other < base; other++ {
			if other != digit {
				words = append(words, word+(other-digit)*place)
			}
		}
		place *= base
	}
	return words
}

// buildMatrix makes one column for each word, which must end up in exactly one
// neighbourhood, and one row for each word as a candidate code word, which
// covers that word's whole neighbourhood.
func buildMatrix(base, length int) dlx.Matrix {
	words := wordCount(base, length)
	m := dlx.NewMatrix(words)
	for word := 0; word < words; word++ {
		m.AddRow(neighbourhood(base, length, word)...)
	}
	return m
}

// decode relies on dlx keeping each row's columns in AddRow order: the first
// column of a row is its code word.
func decode(exactCover [][]int) []int {
	code := make([]int, 0, len(exactCover))
	for _, row := range exactCover {
		code = append(code, row[0])
	}
	sort.Ints(code)
	return code
}
