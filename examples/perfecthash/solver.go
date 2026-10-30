package main

import "github.com/dimchansky/dlx"

// bucket holds the keys that share a first letter. They all get the same shift.
type bucket struct {
	letter byte
	keys   []string
}

// hashTable is a finished minimal perfect hash.
type hashTable struct {
	shifts map[byte]int // shift for each first letter in use
	slots  []string     // the key stored in each slot
}

// matrixSize says how large the exact cover matrix was.
type matrixSize struct {
	columns, rows int
}

// lookup returns the slot of word, and false if word isn't one of the keys. A
// perfect hash sends a stranger to some slot as well, so the stored key has to
// be compared.
func (t *hashTable) lookup(word string) (int, bool) {
	if word == "" {
		return 0, false
	}
	shift, ok := t.shifts[word[0]]
	if !ok {
		return 0, false
	}
	slot := slotOf(word, shift, len(t.slots))
	return slot, t.slots[slot] == word
}

// letterSum numbers the letters a=1 to z=26, multiplies each by its position in
// the word, counting from 1, and adds the results. Multiplying by the position
// keeps anagrams such as "abc" and "cba" apart.
func letterSum(word string) int {
	sum := 0
	for i := 0; i < len(word); i++ {
		sum += (i + 1) * (int(word[i]) - 'a' + 1)
	}
	return sum
}

// slotOf returns the slot of word in a table of the given size. Only lookup can
// pass a word with bytes below 'a', so the remainder is made non-negative.
func slotOf(word string, shift, size int) int {
	return ((letterSum(word)+shift)%size + size) % size
}

// solve passes each hash table for keys to accept until accept returns true, and
// returns the size of the matrix. The keys must be distinct.
func solve(keys []string, accept func(table *hashTable) bool) matrixSize {
	buckets := groupByFirstLetter(keys)
	matrix, size := buildMatrix(buckets, len(keys))
	matrix.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		return accept(decode(buckets, len(keys), exactCover))
	}))
	return size
}

// groupByFirstLetter returns the buckets in alphabetical order. Keys keep their
// order inside a bucket.
func groupByFirstLetter(keys []string) []bucket {
	byLetter := make(map[byte][]string)
	for _, key := range keys {
		byLetter[key[0]] = append(byLetter[key[0]], key)
	}
	buckets := make([]bucket, 0, len(byLetter))
	for letter := byte('a'); letter <= 'z'; letter++ {
		if group, ok := byLetter[letter]; ok {
			buckets = append(buckets, bucket{letter: letter, keys: group})
		}
	}
	return buckets
}

// buildMatrix has a column for each bucket, then a column for each slot. Each
// row gives one bucket one shift: it covers the bucket's column and the slots
// its keys land on. So every bucket gets exactly one shift and every slot gets
// exactly one key. Slots and keys are equal in number, so no slot is optional
// and no slack rows are needed.
//
// A shift moves all keys of a bucket together, so two keys of a bucket that land
// on one slot do so under every shift. Such a bucket has no rows, its column
// can't be covered, and there is no solution.
func buildMatrix(buckets []bucket, slots int) (dlx.Matrix, matrixSize) {
	size := matrixSize{columns: len(buckets) + slots}
	matrix := dlx.NewMatrix(size.columns)
	for i, b := range buckets {
		for shift := 0; shift < slots; shift++ {
			if row, ok := rowFor(b, i, shift, len(buckets), slots); ok {
				matrix.AddRow(row...)
				size.rows++
			}
		}
	}
	return matrix, size
}

// rowFor returns the columns of one bucket under one shift, with the bucket's
// column first and then one slot column per key, in the order of the bucket's
// keys. It returns false if two of the keys land on one slot.
func rowFor(b bucket, index, shift, firstSlot, slots int) ([]int, bool) {
	row := make([]int, 0, 1+len(b.keys))
	row = append(row, index)
	seen := make(map[int]bool, len(b.keys))
	for _, key := range b.keys {
		slot := slotOf(key, shift, slots)
		if seen[slot] {
			return nil, false
		}
		seen[slot] = true
		row = append(row, firstSlot+slot)
	}
	return row, true
}

// decode relies on dlx keeping each row's columns in AddRow order: the bucket
// comes first, then one slot for each of its keys. The shift is the distance
// from where the first key lands with no shift to where it landed.
func decode(buckets []bucket, slots int, exactCover [][]int) *hashTable {
	table := &hashTable{shifts: make(map[byte]int, len(buckets)), slots: make([]string, slots)}
	for _, row := range exactCover {
		b := buckets[row[0]]
		for i, key := range b.keys {
			table.slots[row[1+i]-len(buckets)] = key
		}
		firstSlot := row[1] - len(buckets)
		table.shifts[b.letter] = (firstSlot - slotOf(b.keys[0], 0, slots) + slots) % slots
	}
	return table
}
