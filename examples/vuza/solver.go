package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dimchansky/dlx"
)

// solve passes each complement of tile in a loop of n beats to accept until
// accept returns true. A complement is the sorted list of beats on which a voice
// enters, so that the voices together strike every beat exactly once. The tile
// must not repeat, or two entries would give the same row.
func solve(n int, tile []int, accept func(entries []int) bool) {
	buildMatrix(n, tile).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(n, tile, exactCover))
	}))
}

// buildMatrix has a column for each beat of the loop and a row for each beat a
// voice can enter on. A row lists the beats that voice strikes, wrapping round
// the loop. Every beat is struck exactly once, so the picked rows are the
// voices of a canon.
func buildMatrix(n int, tile []int) dlx.Matrix {
	m := dlx.NewMatrix(n)
	row := make([]int, len(tile))
	for entry := 0; entry < n; entry++ {
		for i, beat := range tile {
			row[i] = (beat + entry) % n
		}
		m.AddRow(row...)
	}
	return m
}

// decode relies on dlx keeping each row's columns in AddRow order: the first
// column of a row is the first note of the tile, moved on by the voice's entry.
func decode(n int, tile []int, exactCover [][]int) []int {
	entries := make([]int, len(exactCover))
	for i, row := range exactCover {
		entries[i] = (row[0] - tile[0] + n) % n
	}
	sort.Ints(entries)
	return entries
}

// period returns the smallest p in 1 to n-1 such that turning the beats on by p
// gives the same beats, or 0 if there is none. The set is then called aperiodic.
// Only divisors of n need trying: a set that repeats every p also repeats every
// gcd(p, n).
func period(n int, beats []int) int {
	struck := make([]bool, n)
	for _, beat := range beats {
		struck[beat] = true
	}
	for p := 1; p < n; p++ {
		if n%p == 0 && repeatsEvery(struck, p) {
			return p
		}
	}
	return 0
}

func repeatsEvery(struck []bool, p int) bool {
	n := len(struck)
	for beat, isStruck := range struck {
		if isStruck != struck[(beat+p)%n] {
			return false
		}
	}
	return true
}

// parseTile reads a comma-separated list of different beats in 0 to n-1. It
// refuses a tile that repeats, because the voices of such a tile can't be told
// apart and a Vuza canon needs an aperiodic tile anyway.
func parseTile(text string, n int) ([]int, error) {
	fields := strings.Split(text, ",")
	tile := make([]int, len(fields))
	seen := make(map[int]bool, len(fields))
	for i, field := range fields {
		beat, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return nil, fmt.Errorf("beat %q is not a number", field)
		}
		if beat < 0 || beat >= n {
			return nil, fmt.Errorf("beat %d is outside 0 to %d", beat, n-1)
		}
		if seen[beat] {
			return nil, fmt.Errorf("beat %d appears twice", beat)
		}
		seen[beat] = true
		tile[i] = beat
	}
	switch p := period(n, tile); p {
	case 0:
		return tile, nil
	case 1:
		return nil, errors.New("it repeats every beat")
	default:
		return nil, fmt.Errorf("it repeats every %d beats", p)
	}
}
