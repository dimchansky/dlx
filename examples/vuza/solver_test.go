package main

import (
	"fmt"
	"math/bits"
	"reflect"
	"sort"
	"testing"

	"github.com/dimchansky/dlx"
)

var (
	rhythm72 = []int{0, 8, 16, 18, 26, 34}
	// partner72 is the first aperiodic complement of rhythm72. Its voices play
	// twelve notes, and rhythm72 is a complement of it in turn.
	partner72 = []int{0, 1, 5, 6, 12, 25, 29, 36, 42, 48, 49, 53}
)

// publishedSixNote and publishedTwelveNote are the gaps between the notes of the
// rhythms in Fripertinger's list of canons of length 72:
// https://imsc.uni-graz.at/fripertinger/canons/canon_3.html
// Jedrzejewski's Table 1 (https://arxiv.org/abs/1304.6609) gives the same totals
// of 3 and 6 for a loop of 72.
var (
	publishedSixNote = [][]int{
		{38, 8, 8, 2, 8, 8},
		{18, 14, 8, 10, 8, 14},
		{22, 16, 2, 14, 2, 16},
	}
	publishedTwelveNote = [][]int{
		{19, 1, 4, 1, 6, 13, 4, 7, 6, 6, 1, 4},
		{17, 4, 3, 6, 6, 5, 4, 15, 5, 1, 3, 3},
		{20, 1, 3, 3, 6, 11, 4, 9, 6, 5, 1, 3},
		{19, 4, 1, 6, 6, 7, 4, 13, 6, 1, 4, 1},
		{20, 3, 1, 5, 6, 9, 4, 11, 6, 3, 3, 1},
		{17, 3, 3, 1, 5, 15, 4, 5, 6, 6, 3, 4},
	}
)

// searchByHand finds all complements without the exact cover machinery: it takes
// the first beat nobody strikes yet and tries every voice that could strike it.
func searchByHand(n int, tile []int, visit func(entries []int)) {
	searchByHandFrom(n, tile, nil, visit)
}

// searchByHandFrom is searchByHand with the given voices already placed.
func searchByHandFrom(n int, tile, placed []int, visit func(entries []int)) {
	struck := make([]bool, n)
	for _, entry := range placed {
		for _, note := range tile {
			struck[(note+entry)%n] = true
		}
	}
	entries := append([]int(nil), placed...)
	var fill func(beat int)
	fill = func(beat int) {
		for beat < n && struck[beat] {
			beat++
		}
		if beat == n {
			sorted := append([]int(nil), entries...)
			sort.Ints(sorted)
			visit(sorted)
			return
		}
		for _, note := range tile {
			entry := (beat - note + n) % n
			fits := true
			for _, other := range tile {
				fits = fits && !struck[(other+entry)%n]
			}
			if !fits {
				continue
			}
			for _, other := range tile {
				struck[(other+entry)%n] = true
			}
			entries = append(entries, entry)
			fill(beat)
			entries = entries[:len(entries)-1]
			for _, other := range tile {
				struck[(other+entry)%n] = false
			}
		}
	}
	fill(0)
}

// strikesEveryBeatOnce adds up the two sets directly, the definition of a
// complement.
func strikesEveryBeatOnce(n int, tile, entries []int) bool {
	strikes := make([]int, n)
	for _, note := range tile {
		for _, entry := range entries {
			strikes[(note+entry)%n]++
		}
	}
	for _, count := range strikes {
		if count != 1 {
			return false
		}
	}
	return true
}

// isAperiodic tries every turn of the loop, the definition of not repeating.
func isAperiodic(n int, beats []int) bool {
	struck := make([]bool, n)
	for _, beat := range beats {
		struck[beat] = true
	}
	for turn := 1; turn < n; turn++ {
		same := true
		for _, beat := range beats {
			same = same && struck[(beat+turn)%n]
		}
		if same {
			return false
		}
	}
	return true
}

// shape returns the distances between neighbouring notes round the loop, turned
// so that the smallest list comes first. Rhythms that differ only by a shift have
// the same shape.
func shape(n int, beats []int) string {
	sorted := append([]int(nil), beats...)
	sort.Ints(sorted)
	list := make([]int, len(sorted))
	for i, beat := range sorted {
		list[i] = (sorted[(i+1)%len(sorted)] - beat + n) % n
	}
	best := ""
	for turn := range list {
		turned := fmt.Sprintf("%3d", append(append([]int(nil), list[turn:]...), list[:turn]...))
		if best == "" || turned < best {
			best = turned
		}
	}
	return best
}

// beatsFromGaps turns a list of distances between notes into beats from 0.
func beatsFromGaps(gapList []int) []int {
	beats := make([]int, len(gapList))
	at := 0
	for i, gap := range gapList {
		beats[i] = at
		at += gap
	}
	return beats
}

func shapes(n int, gapLists [][]int) map[string]bool {
	classes := map[string]bool{}
	for _, gapList := range gapLists {
		classes[shape(n, beatsFromGaps(gapList))] = true
	}
	return classes
}

// aperiodicComplements returns how many complements there are and those that
// don't repeat, after checking every complement against the definition.
func aperiodicComplements(t *testing.T, n int, tile []int) (total int, aperiodic [][]int) {
	t.Helper()
	solve(n, tile, func(entries []int) bool {
		total++
		if !strikesEveryBeatOnce(n, tile, entries) {
			t.Errorf("complement %v does not strike every beat once", entries)
		}
		if isAperiodic(n, entries) {
			aperiodic = append(aperiodic, entries)
		}
		return false
	})
	return total, aperiodic
}

func TestBuildMatrix_MatchesReadmeExample(t *testing.T) {
	t.Parallel()

	var got [][]int
	solve(6, []int{0, 1, 2}, func(entries []int) bool {
		got = append(got, entries)
		return false
	})
	want := [][]int{{0, 3}, {1, 4}, {2, 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSolve_KnownTotals checks the 72-beat totals in the README and the small
// examples against the separate search, and every complement against the
// definition. TestSolve_EntryOnBeatZero checks the 144-beat totals.
func TestSolve_KnownTotals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		n             int
		tile          []int
		wantTotal     int
		wantAperiodic int
	}{
		{name: "six-note rhythm", n: 72, tile: rhythm72, wantTotal: 1296, wantAperiodic: 432},
		{name: "twelve-note partner", n: 72, tile: partner72, wantTotal: 216, wantAperiodic: 216},
		{name: "readme example", n: 6, tile: []int{0, 1, 2}, wantTotal: 3},
		{name: "two notes in four beats", n: 4, tile: []int{0, 1}, wantTotal: 2},
		{name: "does not tile", n: 6, tile: []int{0, 1, 3}},
		{name: "size does not divide the loop", n: 7, tile: []int{0, 1}},
		{name: "tile need not start at zero", n: 6, tile: []int{1, 2, 3}, wantTotal: 3},
		{name: "single note", n: 5, tile: []int{0}, wantTotal: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			total, aperiodic := aperiodicComplements(t, tt.n, tt.tile)
			if total != tt.wantTotal {
				t.Errorf("found %d complements, want %d", total, tt.wantTotal)
			}
			if len(aperiodic) != tt.wantAperiodic {
				t.Errorf("found %d aperiodic complements, want %d", len(aperiodic), tt.wantAperiodic)
			}

			found := map[string]bool{}
			solve(tt.n, tt.tile, func(entries []int) bool {
				found[fmt.Sprint(entries)] = true
				return false
			})
			independent := 0
			searchByHand(tt.n, tt.tile, func(entries []int) {
				independent++
				if !found[fmt.Sprint(entries)] {
					t.Errorf("separate search found %v, which solve did not", entries)
				}
			})
			if independent != total {
				t.Errorf("separate search found %d complements, solve found %d", independent, total)
			}
		})
	}
}

// TestSolve_AllSmallTiles compares solve with generate and test: every subset of
// the loop is tried as a complement, so nothing about the search is shared.
func TestSolve_AllSmallTiles(t *testing.T) {
	t.Parallel()

	for n := 2; n <= 12; n++ {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			if n > 10 && testing.Short() {
				t.Skip("slow: tries every rhythm against every subset")
			}
			for tileMask := 1; tileMask < 1<<n; tileMask += 2 {
				tile := beatsOf(tileMask, n)
				if !isAperiodic(n, tile) {
					continue
				}
				want := map[string]bool{}
				for mask := 1; mask < 1<<n; mask++ {
					if bits.OnesCount(uint(mask))*len(tile) != n {
						continue
					}
					if entries := beatsOf(mask, n); strikesEveryBeatOnce(n, tile, entries) {
						want[fmt.Sprint(entries)] = true
					}
				}
				got := map[string]bool{}
				solve(n, tile, func(entries []int) bool {
					got[fmt.Sprint(entries)] = true
					return false
				})
				if !reflect.DeepEqual(got, want) {
					t.Errorf("tile %v: got %v, want %v", tile, got, want)
				}
			}
		})
	}
}

func beatsOf(mask, n int) []int {
	var beats []int
	for beat := 0; beat < n; beat++ {
		if mask&(1<<beat) != 0 {
			beats = append(beats, beat)
		}
	}
	return beats
}

// TestSolve_PublishedRhythms rebuilds the published grid of 3 by 6 rhythms: each
// six-note rhythm must give exactly the six twelve-note rhythms, shifted every
// possible way, and each twelve-note rhythm exactly the three six-note ones.
func TestSolve_PublishedRhythms(t *testing.T) {
	t.Parallel()

	const n = 72
	sixNote, twelveNote := shapes(n, publishedSixNote), shapes(n, publishedTwelveNote)
	if len(sixNote) != 3 || len(twelveNote) != 6 {
		t.Fatalf("published lists hold %d and %d classes, want 3 and 6", len(sixNote), len(twelveNote))
	}

	tests := []struct {
		name          string
		rhythms       [][]int
		partners      map[string]bool
		wantTotal     int
		wantAperiodic int
	}{
		{name: "six-note rhythms", rhythms: publishedSixNote, partners: twelveNote, wantTotal: 1296, wantAperiodic: 432},
		{name: "twelve-note rhythms", rhythms: publishedTwelveNote, partners: sixNote, wantTotal: 216, wantAperiodic: 216},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rhythms := tt.rhythms
			if testing.Short() {
				rhythms = rhythms[:1]
			}
			for _, gapList := range rhythms {
				tile := beatsFromGaps(gapList)
				if !isAperiodic(n, tile) {
					t.Errorf("rhythm %v repeats", tile)
				}

				total, aperiodic := aperiodicComplements(t, n, tile)
				if total != tt.wantTotal {
					t.Errorf("rhythm %v: found %d complements, want %d", tile, total, tt.wantTotal)
				}
				if len(aperiodic) != tt.wantAperiodic {
					t.Errorf("rhythm %v: found %d aperiodic complements, want %d", tile, len(aperiodic), tt.wantAperiodic)
				}

				classes := map[string]int{}
				for _, entries := range aperiodic {
					classes[shape(n, entries)]++
				}
				if len(classes) != len(tt.partners) {
					t.Errorf("rhythm %v: found %d classes, want %d", tile, len(classes), len(tt.partners))
				}
				for class, shifts := range classes {
					if !tt.partners[class] {
						t.Errorf("rhythm %v: class %s is not published", tile, class)
					}
					if shifts != n {
						t.Errorf("rhythm %v: class %s has %d shifts, want %d", tile, class, shifts, n)
					}
				}
			}
		})
	}
}

// TestSolve_PaperExamples checks the 12-note sets printed in the paper: Example
// 18 is a true partner of the rhythm, while the one in Example 7 completes the
// rhythm but repeats every 36 beats, as the README warns.
func TestSolve_PaperExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entries    []int
		wantPeriod int
	}{
		{name: "example 18", entries: []int{0, 1, 21, 24, 25, 30, 36, 45, 49, 60, 66, 69}},
		{name: "example 7", entries: []int{0, 6, 12, 17, 23, 29, 36, 42, 48, 53, 59, 65}, wantPeriod: 36},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !strikesEveryBeatOnce(72, rhythm72, tt.entries) {
				t.Errorf("the set does not complete the rhythm")
			}
			if got := period(72, tt.entries); got != tt.wantPeriod {
				t.Errorf("period = %d, want %d", got, tt.wantPeriod)
			}
		})
	}
}

// TestSolve_EntryOnBeatZero checks the 144-beat totals in the README, including
// the 8640 shapes of 24 entries that one row of Table 1 in the paper gives for six
// notes. Enumerating them all takes seconds, so this keeps only the complements
// with an entry on beat 0: of the voices that can strike beat 0, only the one
// entering there stays. A complement has exactly one of those voices, and a shift
// moves that job from one to another, so this finds 1/notes of every total. The
// 72-beat rows check that argument against totals enumerated in full. Both dlx and
// the separate search must get there, and they share no code with each other.
func TestSolve_EntryOnBeatZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		n             int
		tile          []int
		slow          bool
		wantTotal     int
		wantAperiodic int
	}{
		{name: "six notes in 72 beats", n: 72, tile: rhythm72, wantTotal: 1296, wantAperiodic: 432},
		{name: "twelve notes in 72 beats", n: 72, tile: partner72, wantTotal: 216, wantAperiodic: 216},
		{name: "six notes in 144 beats", n: 144, tile: []int{0, 16, 32, 36, 52, 68}, slow: true, wantTotal: 1679616, wantAperiodic: 8640 * 144},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: enumerates a sixth of the complements")
			}

			wantTotal, wantAperiodic := tt.wantTotal/len(tt.tile), tt.wantAperiodic/len(tt.tile)

			total, aperiodic := 0, 0
			matrixWithEntryOnZero(tt.n, tt.tile).Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
				entries := decode(tt.n, tt.tile, exactCover)
				if !strikesEveryBeatOnce(tt.n, tt.tile, entries) {
					t.Errorf("complement %v does not strike every beat once", entries)
				}
				total++
				if period(tt.n, entries) == 0 {
					aperiodic++
				}
				return false
			}))
			if total != wantTotal || aperiodic != wantAperiodic {
				t.Errorf("dlx found %d complements, %d aperiodic, want %d and %d", total, aperiodic, wantTotal, wantAperiodic)
			}

			total, aperiodic = 0, 0
			searchByHandFrom(tt.n, tt.tile, []int{0}, func(entries []int) {
				total++
				if isAperiodic(tt.n, entries) {
					aperiodic++
				}
			})
			if total != wantTotal || aperiodic != wantAperiodic {
				t.Errorf("separate search found %d complements, %d aperiodic, want %d and %d", total, aperiodic, wantTotal, wantAperiodic)
			}
		})
	}
}

// matrixWithEntryOnZero is buildMatrix without the voices that strike beat 0 from
// any entry but 0. The tile must have a note on beat 0.
func matrixWithEntryOnZero(n int, tile []int) dlx.Matrix {
	m := dlx.NewMatrix(n)
	for entry := 0; entry < n; entry++ {
		voice := make([]int, len(tile))
		strikesZero := false
		for i, note := range tile {
			voice[i] = (note + entry) % n
			strikesZero = strikesZero || voice[i] == 0
		}
		if entry == 0 || !strikesZero {
			m.AddRow(voice...)
		}
	}
	return m
}

// TestSolve_NoVuzaCanonInShortLoops searches every rhythm for the claim that
// short loops have no Vuza canon. One of the two voices has at most sqrt(n)
// notes, and shifting a canon lets that voice start on beat 0.
func TestSolve_NoVuzaCanonInShortLoops(t *testing.T) {
	t.Parallel()

	for n := 2; n <= 28; n++ {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			if n > 16 && testing.Short() {
				t.Skip("slow: tries every short rhythm")
			}
			for size := 1; size*size <= n; size++ {
				if n%size != 0 {
					continue
				}
				forEachTile(n, size, func(tile []int) {
					if !isAperiodic(n, tile) {
						return
					}
					solve(n, tile, func(entries []int) bool {
						if isAperiodic(n, entries) {
							t.Errorf("Vuza canon: tile %v, entries %v", tile, entries)
						}
						return false
					})
				})
			}
		})
	}
}

// forEachTile visits every rhythm of the given size that starts on beat 0.
func forEachTile(n, size int, visit func(tile []int)) {
	tile := make([]int, size)
	var extend func(i, from int)
	extend = func(i, from int) {
		if i == size {
			visit(tile)
			return
		}
		for beat := from; beat < n; beat++ {
			tile[i] = beat
			extend(i+1, beat+1)
		}
	}
	extend(1, 1)
}

func TestSolve_StopsWhenAccepted(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(72, rhythm72, func([]int) bool {
		calls++
		return calls == 3
	})
	if calls != 3 {
		t.Errorf("accepter called %d times, want 3", calls)
	}
}

func TestPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		n     int
		beats []int
		want  int
	}{
		{name: "aperiodic rhythm", n: 72, beats: rhythm72, want: 0},
		{name: "aperiodic partner", n: 72, beats: partner72, want: 0},
		{name: "every third beat", n: 12, beats: []int{0, 3, 6, 9}, want: 3},
		{name: "half way round", n: 6, beats: []int{0, 3}, want: 3},
		{name: "a pattern that comes back", n: 12, beats: []int{0, 1, 6, 7}, want: 6},
		{name: "every beat", n: 4, beats: []int{0, 1, 2, 3}, want: 1},
		{name: "one note", n: 6, beats: []int{2}, want: 0},
		{name: "three notes in twelve beats", n: 12, beats: []int{0, 1, 5}, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := period(tt.n, tt.beats); got != tt.want {
				t.Errorf("period = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestPeriod_DivisorsSuffice checks the claim that only divisors of n need
// trying, by comparing with every possible turn on every set in short loops.
func TestPeriod_DivisorsSuffice(t *testing.T) {
	t.Parallel()

	for n := 2; n <= 12; n++ {
		for mask := 1; mask < 1<<n; mask++ {
			beats := beatsOf(mask, n)
			if got, want := period(n, beats) == 0, isAperiodic(n, beats); got != want {
				t.Errorf("n=%d beats %v: aperiodic = %v, want %v", n, beats, got, want)
			}
		}
	}
}

func TestParseTile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		n       int
		want    []int
		wantErr string
	}{
		{name: "plain list", text: "0,1,2", n: 6, want: []int{0, 1, 2}},
		{name: "spaces are allowed", text: " 0, 1 ,2", n: 6, want: []int{0, 1, 2}},
		{name: "order is kept", text: "2,0", n: 6, want: []int{2, 0}},
		{name: "empty", text: "", n: 6, wantErr: `beat "" is not a number`},
		{name: "not a number", text: "0,x", n: 6, wantErr: `beat "x" is not a number`},
		{name: "negative", text: "-1", n: 6, wantErr: "beat -1 is outside 0 to 5"},
		{name: "past the loop", text: "6", n: 6, wantErr: "beat 6 is outside 0 to 5"},
		{name: "repeated beat", text: "1,1", n: 6, wantErr: "beat 1 appears twice"},
		{name: "repeating rhythm", text: "0,3", n: 6, wantErr: "it repeats every 3 beats"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseTile(tt.text, tt.n)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
