package main

import (
	"strings"
	"testing"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// plainSlot computes a slot in its own way, from the last letter to the first
// and by searching the alphabet, so a mistake in letterSum doesn't repeat here.
func plainSlot(word string, shift, size int) int {
	total := shift
	for i := len(word); i > 0; i-- {
		total += i * (strings.IndexByte(alphabet, word[i-1]) + 1)
	}
	return total % size
}

// allTables tries every combination of shifts, one for each first letter, with
// no matrix at all. It returns the slot lists that have no clash.
func allTables(keys []string) []string {
	var letters []byte
	for _, key := range keys {
		if strings.IndexByte(string(letters), key[0]) < 0 {
			letters = append(letters, key[0])
		}
	}

	var found []string
	shifts := make(map[byte]int)
	var try func(depth int)
	try = func(depth int) {
		if depth < len(letters) {
			for shift := 0; shift < len(keys); shift++ {
				shifts[letters[depth]] = shift
				try(depth + 1)
			}
			return
		}
		slots := make([]string, len(keys))
		for _, key := range keys {
			slot := plainSlot(key, shifts[key[0]], len(keys))
			if slots[slot] != "" {
				return
			}
			slots[slot] = key
		}
		found = append(found, strings.Join(slots, " "))
	}
	try(0)
	return found
}

// anyTable searches letter by letter with no matrix and stops at the first
// table without a clash. It reports whether there is one.
func anyTable(keys []string) bool {
	groups := make(map[byte][]string)
	for _, key := range keys {
		groups[key[0]] = append(groups[key[0]], key)
	}
	var letters []byte
	for i := 0; i < len(alphabet); i++ {
		if groups[alphabet[i]] != nil {
			letters = append(letters, alphabet[i])
		}
	}

	used := make([]bool, len(keys))
	var try func(depth int) bool
	try = func(depth int) bool {
		if depth == len(letters) {
			return true
		}
	shiftLoop:
		for shift := 0; shift < len(keys); shift++ {
			var taken []int
			undo := func() {
				for _, slot := range taken {
					used[slot] = false
				}
			}
			for _, key := range groups[letters[depth]] {
				slot := plainSlot(key, shift, len(keys))
				if used[slot] {
					undo()
					continue shiftLoop
				}
				used[slot] = true
				taken = append(taken, slot)
			}
			if try(depth + 1) {
				return true
			}
			undo()
		}
		return false
	}
	return try(0)
}

func TestLetterSum(t *testing.T) {
	t.Parallel()

	// Worked out by hand.
	tests := []struct {
		word string
		want int
	}{
		{word: "a", want: 1},
		{word: "if", want: 9 + 2*6},
		{word: "map", want: 13 + 2*1 + 3*16},
		{word: "var", want: 22 + 2*1 + 3*18},
		{word: "for", want: 6 + 2*15 + 3*18},
		{word: "func", want: 6 + 2*21 + 3*14 + 4*3},
		{word: "abc", want: 1 + 2*2 + 3*3},
		{word: "cba", want: 3 + 2*2 + 3*1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.word, func(t *testing.T) {
			t.Parallel()

			if got := letterSum(tt.word); got != tt.want {
				t.Errorf("letterSum(%q) = %d, want %d", tt.word, got, tt.want)
			}
		})
	}
}

func TestSolve_FirstTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		keys        []string
		wantBuckets int
		wantColumns int
		wantRows    int
	}{
		{
			name:        "go keywords",
			keys:        goKeywords,
			wantBuckets: 13,
			wantColumns: 38,
			wantRows:    325,
		},
		{
			name:        "readme example",
			keys:        []string{"for", "func", "if", "map", "var"},
			wantBuckets: 4,
			wantColumns: 9,
			wantRows:    20,
		},
		{
			name:        "one key",
			keys:        []string{"a"},
			wantBuckets: 1,
			wantColumns: 2,
			wantRows:    1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var table *hashTable
			size := solve(tt.keys, func(found *hashTable) bool {
				table = found
				return true
			})

			if size != (matrixSize{columns: tt.wantColumns, rows: tt.wantRows}) {
				t.Errorf("matrix size = %+v, want %d columns and %d rows", size, tt.wantColumns, tt.wantRows)
			}
			if table == nil {
				t.Fatal("no table found")
			}
			if len(table.shifts) != tt.wantBuckets {
				t.Errorf("%d shifts, want %d", len(table.shifts), tt.wantBuckets)
			}
			checkTable(t, table, tt.keys)
		})
	}
}

// checkTable proves a table right without trusting the solver: it recomputes
// each key's slot, and each slot must hold that key alone.
func checkTable(t *testing.T, table *hashTable, keys []string) {
	t.Helper()

	if len(table.slots) != len(keys) {
		t.Fatalf("%d slots for %d keys", len(table.slots), len(keys))
	}
	counts := make(map[string]int)
	for _, key := range table.slots {
		counts[key]++
	}
	for _, key := range keys {
		if counts[key] != 1 {
			t.Errorf("key %q is in %d slots, want 1", key, counts[key])
		}
		want := plainSlot(key, table.shifts[key[0]], len(keys))
		if table.slots[want] != key {
			t.Errorf("slot %d holds %q, want %q", want, table.slots[want], key)
		}
		slot, ok := table.lookup(key)
		if !ok || slot != want {
			t.Errorf("lookup(%q) = %d, %v, want %d, true", key, slot, ok, want)
		}
	}
}

func TestHashTable_LookupRejectsStrangers(t *testing.T) {
	t.Parallel()

	var table *hashTable
	solve(goKeywords, func(found *hashTable) bool {
		table = found
		return true
	})
	if table == nil {
		t.Fatal("no table found")
	}

	// Most of these start with a letter in use, so they reach a slot and only
	// the comparison stops them. "gO" and "var " have bytes below 'a', which
	// make the letter sum smaller or even negative.
	if slot, ok := table.lookup("goto"); slot != 9 || !ok {
		t.Errorf("lookup(goto) = %d, %v, want the README's 9, true", slot, ok)
	}
	if slot, ok := table.lookup("gopher"); slot != 2 || ok {
		t.Errorf("lookup(gopher) = %d, %v, want the README's 2, false", slot, ok)
	}
	for _, word := range []string{"", "G", "Go", "gO", "x", "gopher", "var ", "fo", "typeof", "returns"} {
		if slot, ok := table.lookup(word); ok {
			t.Errorf("lookup(%q) found slot %d", word, slot)
		}
	}
}

func TestSolve_MatchesEveryCombination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keys      []string
		wantTotal int
	}{
		{name: "readme example", keys: []string{"for", "func", "if", "map", "var"}, wantTotal: 30},
		{name: "one key", keys: []string{"a"}, wantTotal: 1},
		{name: "two letters", keys: []string{"a", "b"}, wantTotal: 2},
		{name: "same letter", keys: []string{"a", "aaa"}, wantTotal: 2},
		{name: "same letter clash", keys: []string{"ab", "ad"}, wantTotal: 0},
		{name: "shifts do not fit together", keys: []string{"aa", "ab", "ba", "baa"}, wantTotal: 0},
		{name: "all letters differ", keys: []string{"a", "b", "c", "d"}, wantTotal: 24},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			brute := allTables(tt.keys)
			if len(brute) != tt.wantTotal {
				t.Errorf("trying every combination found %d tables, want %d", len(brute), tt.wantTotal)
			}
			want := make(map[string]bool, len(brute))
			for _, slots := range brute {
				want[slots] = true
			}

			total := 0
			solve(tt.keys, func(table *hashTable) bool {
				checkTable(t, table, tt.keys)
				if slots := strings.Join(table.slots, " "); !want[slots] {
					t.Errorf("dlx found the unknown table %q", slots)
				}
				total++
				return false
			})

			if total != len(want) {
				t.Errorf("dlx found %d tables, every combination found %d", total, len(want))
			}
		})
	}
}

func TestSolve_MatchesPlainSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		keys []string
		want bool
	}{
		{name: "go keywords", keys: goKeywords, want: true},
		{name: "first ten keywords clash", keys: goKeywords[:10], want: false},
		{name: "three keys", keys: []string{"cap", "complex", "ab"}, want: true},
		{name: "shifts do not fit together", keys: []string{"aa", "ab", "ba", "baa"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := anyTable(tt.keys); got != tt.want {
				t.Errorf("plain search found a table: %v, want %v", got, tt.want)
			}
			found := false
			solve(tt.keys, func(*hashTable) bool {
				found = true
				return true
			})
			if found != tt.want {
				t.Errorf("dlx found a table: %v, want %v", found, tt.want)
			}
		})
	}
}

func TestSolve_ClashingListHasNoTable(t *testing.T) {
	t.Parallel()

	// The README's list of 31 words: cap and complex land on one slot, and so
	// do imag and int, and panic and print.
	words := strings.Split("bool,byte,cap,clear,close,complex,copy,delete,error,false,float,imag,"+
		"int,iota,len,make,max,min,new,nil,panic,print,println,real,recover,rune,string,true,uint,"+
		"uintptr,any", ",")
	if len(words) != 31 {
		t.Fatalf("%d words, want 31", len(words))
	}
	for _, pair := range [][2]string{{"cap", "complex"}, {"imag", "int"}, {"panic", "print"}} {
		if plainSlot(pair[0], 0, 31) != plainSlot(pair[1], 0, 31) {
			t.Errorf("%s and %s don't clash", pair[0], pair[1])
		}
	}

	solve(words, func(*hashTable) bool {
		t.Error("found a table")
		return true
	})
}
