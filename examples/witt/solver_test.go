package main

import (
	"fmt"
	"math/bits"
	"sort"
	"testing"

	"github.com/dimchansky/dlx"
)

// The helpers below share nothing with solver.go. They describe each system by
// arithmetic rather than by search, and they check a system with a different
// rule from the matrix.

// cyclicDesign adds 0, 1, 2 and so on to every point of base, wrapping around at
// modulus, which is a classic way to build a system from one block.
func cyclicDesign(modulus int, base ...int) []uint64 {
	var blocks []uint64
	for shift := 0; shift < modulus; shift++ {
		var block uint64
		for _, point := range base {
			block |= 1 << ((point + shift) % modulus)
		}
		blocks = append(blocks, block)
	}
	return blocks
}

// wittDesign builds S(5,6,12) from one block by applying two shuffles until no
// new block appears. Point 11 plays the part of "infinity", and the shuffles are
// x to x+1 and x to -1/x, with arithmetic modulo 11.
func wittDesign() []uint64 {
	const infinity = 11
	addOne := func(x int) int {
		if x == infinity {
			return infinity
		}
		return (x + 1) % 11
	}
	negativeInverse := func(x int) int {
		switch x {
		case infinity:
			return 0
		case 0:
			return infinity
		}
		for y := 1; y < 11; y++ {
			if (x*y+1)%11 == 0 {
				return y
			}
		}
		panic("no inverse modulo 11")
	}
	apply := func(shuffle func(int) int, block uint64) uint64 {
		var image uint64
		for x := 0; x < 12; x++ {
			if block&(1<<x) != 0 {
				image |= 1 << shuffle(x)
			}
		}
		return image
	}

	first := uint64(1<<infinity | 1<<1 | 1<<3 | 1<<4 | 1<<5 | 1<<9)
	seen := map[uint64]bool{first: true}
	blocks := []uint64{first}
	for i := 0; i < len(blocks); i++ {
		for _, image := range []uint64{apply(addOne, blocks[i]), apply(negativeInverse, blocks[i])} {
			if !seen[image] {
				seen[image] = true
				blocks = append(blocks, image)
			}
		}
	}
	return blocks
}

// wittDerived builds S(4,5,11): take the blocks of S(5,6,12) through point 11
// and remove that point from them.
func wittDerived() []uint64 {
	var blocks []uint64
	for _, block := range wittDesign() {
		if block&(1<<11) != 0 {
			blocks = append(blocks, block&^(1<<11))
		}
	}
	return blocks
}

// isSteiner checks a system without listing t-sets. Blocks that share t points
// would put a t-set in two blocks, so the blocks must overlap in fewer than t
// points. With the right number of blocks, that leaves room for no gaps.
func isSteiner(s spec, blocks []uint64) bool {
	tSetsInAll, tSetsInBlock := 1, 1
	for i := 0; i < s.t; i++ {
		tSetsInAll *= s.v - i
		tSetsInBlock *= s.k - i
	}
	if len(blocks)*tSetsInBlock != tSetsInAll {
		return false
	}
	for i, a := range blocks {
		if bits.OnesCount64(a) != s.k {
			return false
		}
		for _, b := range blocks[i+1:] {
			if bits.OnesCount64(a&b) >= s.t {
				return false
			}
		}
	}
	return true
}

// countSymmetries counts the shuffles of the points that map the system onto
// itself, passing each to visit if visit is not nil. It builds a shuffle one
// point at a time and drops it as soon as the assigned part of some block lands
// outside every block. Parts of up to t points always fit, so only larger parts
// are checked.
func countSymmetries(s spec, blocks []uint64, visit func(image []int)) int {
	fitsInBlock := make([]bool, 1<<s.v)
	for _, block := range blocks {
		for part := block; ; part = (part - 1) & block {
			fitsInBlock[part] = true
			if part == 0 {
				break
			}
		}
	}

	// parts[p] lists what blocks through p look like once points 0 to p are set.
	parts := make([][]uint64, s.v)
	for point := range parts {
		seen := map[uint64]bool{}
		for _, block := range blocks {
			part := block & (1<<(point+1) - 1)
			if part&(1<<point) != 0 && bits.OnesCount64(part) > s.t && !seen[part] {
				seen[part] = true
				parts[point] = append(parts[point], part)
			}
		}
	}

	image := make([]int, s.v)
	isOK := func(point int) bool {
		for _, part := range parts[point] {
			var moved uint64
			for rest := part; rest != 0; rest &= rest - 1 {
				moved |= 1 << image[bits.TrailingZeros64(rest)]
			}
			if !fitsInBlock[moved] {
				return false
			}
		}
		return true
	}

	total := 0
	var used uint64
	var assign func(point int)
	assign = func(point int) {
		if point == s.v {
			total++
			if visit != nil {
				visit(image)
			}
			return
		}
		for candidate := 0; candidate < s.v; candidate++ {
			if used&(1<<candidate) != 0 {
				continue
			}
			image[point] = candidate
			if !isOK(point) {
				continue
			}
			used |= 1 << candidate
			assign(point + 1)
			used &^= 1 << candidate
		}
	}
	assign(0)
	return total
}

// countByBacktracking counts systems without exact cover. It finds the first
// t-set no picked block contains, tries every block through that t-set that
// overlaps the picked blocks in fewer than t points, and backs out of dead ends.
// Unlike dlx it has a fixed order and no links to undo.
func countByBacktracking(s spec) int {
	subsetsOf := func(size int) []uint64 {
		var subsets []uint64
		for mask := uint64(0); mask < 1<<s.v; mask++ {
			if bits.OnesCount64(mask) == size {
				subsets = append(subsets, mask)
			}
		}
		return subsets
	}
	tSets, allBlocks := subsetsOf(s.t), subsetsOf(s.k)
	through := make([][]uint64, len(tSets))
	for i, tSet := range tSets {
		for _, block := range allBlocks {
			if block&tSet == tSet {
				through[i] = append(through[i], block)
			}
		}
	}

	var picked []uint64
	isCovered := func(tSet uint64) bool {
		for _, block := range picked {
			if block&tSet == tSet {
				return true
			}
		}
		return false
	}
	isCompatible := func(candidate uint64) bool {
		for _, block := range picked {
			if bits.OnesCount64(block&candidate) >= s.t {
				return false
			}
		}
		return true
	}

	total := 0
	var search func(next int)
	search = func(next int) {
		for next < len(tSets) && isCovered(tSets[next]) {
			next++
		}
		if next == len(tSets) {
			total++
			return
		}
		for _, candidate := range through[next] {
			if isCompatible(candidate) {
				picked = append(picked, candidate)
				search(next + 1)
				picked = picked[:len(picked)-1]
			}
		}
	}
	search(0)
	return total
}

func factorial(n int) uint64 {
	result := uint64(1)
	for i := 2; i <= n; i++ {
		result *= uint64(i)
	}
	return result
}

func keyOf(blocks []uint64) string {
	sorted := append([]uint64(nil), blocks...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return fmt.Sprint(sorted)
}

func masksOf(blocks [][]int) []uint64 {
	masks := make([]uint64, len(blocks))
	for i, block := range blocks {
		for _, point := range block {
			masks[i] |= 1 << point
		}
	}
	return masks
}

// knownSystems holds the totals the README publishes, with an independent
// construction of one system of each kind.
var knownSystems = []struct {
	name          string
	spec          spec
	wantSolutions uint64
	wantBlocks    int
	wantShuffles  int
	design        func() []uint64
	isSharp       bool // exactly one shuffle sends the first t points to any t targets
	isSlow        bool // skipped under -short, where -race makes it take seconds
	isHuge        bool // too many systems to decode and compare one by one
}{
	{
		name:          "S(1,2,4) pairs",
		spec:          spec{1, 2, 4},
		wantSolutions: 3,
		wantBlocks:    2,
		wantShuffles:  8,
		design:        func() []uint64 { return []uint64{0b0011, 0b1100} },
	},
	{
		name:          "S(2,3,7) Fano plane",
		spec:          spec{2, 3, 7},
		wantSolutions: 30,
		wantBlocks:    7,
		wantShuffles:  168,
		design:        func() []uint64 { return cyclicDesign(7, 0, 1, 3) },
	},
	{
		name:          "S(4,5,11) small Witt design, derived",
		spec:          spec{4, 5, 11},
		wantSolutions: 5040,
		wantBlocks:    66,
		wantShuffles:  7920,
		design:        wittDerived,
		isSharp:       true,
		isSlow:        true,
	},
	{
		name:          "S(5,6,12) small Witt design",
		spec:          spec{5, 6, 12},
		wantSolutions: 5040,
		wantBlocks:    132,
		wantShuffles:  95040,
		design:        wittDesign,
		isSharp:       true,
		isSlow:        true,
	},
	{
		name:          "S(2,4,13) plane of order 3",
		spec:          spec{2, 4, 13},
		wantSolutions: 1108800,
		wantBlocks:    13,
		wantShuffles:  5616,
		design:        func() []uint64 { return cyclicDesign(13, 0, 1, 3, 9) },
		isSlow:        true,
		isHuge:        true,
	},
}

// TestSolve_KnownCounts checks each published total three ways: the solver's own
// count, the number of relabellings of an independently built system, which is
// v! divided by the shuffles that leave that system unchanged, and a separate
// search that never builds a matrix.
func TestSolve_KnownCounts(t *testing.T) {
	t.Parallel()

	for _, tt := range knownSystems {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.isSlow && testing.Short() {
				t.Skip("slow: enumerates every system")
			}

			design := tt.design()
			if !isSteiner(tt.spec, design) {
				t.Fatalf("the independent construction is not S%v", tt.spec)
			}
			if len(design) != tt.wantBlocks {
				t.Errorf("design has %d blocks, want %d", len(design), tt.wantBlocks)
			}

			targets := map[string]bool{}
			shuffles := countSymmetries(tt.spec, design, func(image []int) {
				if tt.isSharp {
					targets[fmt.Sprint(image[:tt.spec.t])] = true
				}
			})
			if shuffles != tt.wantShuffles {
				t.Errorf("%d shuffles keep the design, want %d", shuffles, tt.wantShuffles)
			}
			if tt.isSharp && len(targets) != shuffles {
				t.Errorf("%d shuffles reach %d target lists, want the same number", shuffles, len(targets))
			}
			if got := factorial(tt.spec.v) / uint64(shuffles); got != tt.wantSolutions {
				t.Errorf("%d! / %d = %d, want %d", tt.spec.v, shuffles, got, tt.wantSolutions)
			}

			if tt.isHuge {
				if got := countSolutions(tt.spec); got != tt.wantSolutions {
					t.Errorf("countSolutions = %d, want %d", got, tt.wantSolutions)
				}
				return
			}

			if got := uint64(countByBacktracking(tt.spec)); got != tt.wantSolutions {
				t.Errorf("backtracking found %d systems, want %d", got, tt.wantSolutions)
			}

			found := map[string]bool{}
			var total uint64
			solve(tt.spec, func(blocks [][]int) bool {
				total++
				masks := masksOf(blocks)
				found[keyOf(masks)] = true
				if !isSteiner(tt.spec, masks) {
					t.Errorf("invalid system %v", blocks)
				}
				return false
			})
			if total != tt.wantSolutions {
				t.Errorf("found %d systems, want %d", total, tt.wantSolutions)
			}
			if uint64(len(found)) != total {
				t.Errorf("found %d different systems among %d", len(found), total)
			}
			if !found[keyOf(design)] {
				t.Error("solve did not find the independent construction")
			}
		})
	}
}

// TestSolve_FirstSystem checks the larger systems quickly, for -short runs.
func TestSolve_FirstSystem(t *testing.T) {
	t.Parallel()

	for _, tt := range knownSystems {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			solve(tt.spec, func(blocks [][]int) bool {
				calls++
				if !isSteiner(tt.spec, masksOf(blocks)) {
					t.Errorf("invalid system %v", blocks)
				}
				return true
			})
			if calls != 1 {
				t.Errorf("accepter called %d times, want 1", calls)
			}
		})
	}
}

// TestSolve_EverySolutionIsARelabelling applies all v! shuffles to one system.
// The results must be exactly the systems the solver lists.
func TestSolve_EverySolutionIsARelabelling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		spec   spec
		design []uint64
	}{
		{name: "pairs of 4", spec: spec{1, 2, 4}, design: []uint64{0b0011, 0b1100}},
		{name: "Fano plane", spec: spec{2, 3, 7}, design: cyclicDesign(7, 0, 1, 3)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			relabelled := map[string]bool{}
			image := make([]int, tt.spec.v)
			var used uint64
			var assign func(point int)
			assign = func(point int) {
				if point == tt.spec.v {
					renamed := make([]uint64, len(tt.design))
					for i, block := range tt.design {
						for from, to := range image {
							if block&(1<<from) != 0 {
								renamed[i] |= 1 << to
							}
						}
					}
					relabelled[keyOf(renamed)] = true
					return
				}
				for candidate := 0; candidate < tt.spec.v; candidate++ {
					if used&(1<<candidate) == 0 {
						used |= 1 << candidate
						image[point] = candidate
						assign(point + 1)
						used &^= 1 << candidate
					}
				}
			}
			assign(0)

			found := map[string]bool{}
			solve(tt.spec, func(blocks [][]int) bool {
				found[keyOf(masksOf(blocks))] = true
				return false
			})
			if len(found) != len(relabelled) {
				t.Errorf("solve found %d systems, relabelling gives %d", len(found), len(relabelled))
			}
			for key := range found {
				if !relabelled[key] {
					t.Errorf("system %s is not a relabelling", key)
				}
			}
		})
	}
}

// TestSolve_Planes checks the card game claim: in a projective plane of order n,
// which is S(2,n+1,n*n+n+1), any two cards share exactly one symbol.
func TestSolve_Planes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		order int
		slow  bool
	}{
		{order: 2},
		{order: 3},
		{order: 4, slow: true},
		{order: 5, slow: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("order %d", tt.order), func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: builds a matrix of up to 11 million ones")
			}

			cards := tt.order*tt.order + tt.order + 1
			calls := 0
			solve(spec{2, tt.order + 1, cards}, func(blocks [][]int) bool {
				calls++
				if len(blocks) != cards {
					t.Errorf("found %d cards, want %d", len(blocks), cards)
				}
				masks := masksOf(blocks)
				for i, a := range masks {
					for _, b := range masks[i+1:] {
						if shared := bits.OnesCount64(a & b); shared != 1 {
							t.Errorf("cards %v and %v share %d symbols", pointsOf(a), pointsOf(b), shared)
						}
					}
				}
				return true
			})
			if calls != 1 {
				t.Errorf("accepter called %d times, want 1", calls)
			}
		})
	}
}

// TestCountSolutions_SmallCases covers the edges of the parameters, and the
// parameters that the counting arguments in the README rule out.
func TestCountSolutions_SmallCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec spec
		want uint64
	}{
		{name: "blocks of t points are all forced", spec: spec{2, 2, 4}, want: 1},
		{name: "one block holds every point", spec: spec{2, 4, 4}, want: 1},
		{name: "odd number of points cannot pair up", spec: spec{1, 2, 5}, want: 0},
		{name: "28 pairs do not divide by 3", spec: spec{2, 3, 8}, want: 0},
		{name: "a point cannot be shared out evenly", spec: spec{2, 4, 12}, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := countSolutions(tt.spec); got != tt.want {
				t.Errorf("countSolutions = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSolve_StopsWhenAccepted(t *testing.T) {
	t.Parallel()

	calls := 0
	solve(spec{5, 6, 12}, func([][]int) bool {
		calls++
		return calls == 3
	})
	if calls != 3 {
		t.Errorf("accepter called %d times, want 3", calls)
	}
}

// TestDecode_ReadmeExample checks the worked example in the README: the block
// 1 3 5 of S(2,3,7) comes back from dlx as columns 7, 9 and 16.
func TestDecode_ReadmeExample(t *testing.T) {
	t.Parallel()

	matrix, subsets := buildMatrix(spec{2, 3, 7})
	if len(subsets) != 21 {
		t.Fatalf("matrix has %d columns, want 21", len(subsets))
	}

	const want = "[7 9 16]"
	isReported := false
	matrix.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		for _, row := range exactCover {
			isReported = isReported || fmt.Sprint(row) == want
		}
		return isReported
	}))
	if !isReported {
		t.Errorf("no solution reports the row %s", want)
	}

	blocks := decode(subsets, [][]int{{7, 9, 16}})
	if got := fmt.Sprint(blocks); got != "[[1 3 5]]" {
		t.Errorf("decode = %s, want [[1 3 5]]", got)
	}
}

func TestBinomial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n, k int
		want uint64
	}{
		{n: 12, k: 5, want: 792},
		{n: 12, k: 6, want: 924},
		{n: 7, k: 0, want: 1},
		{n: 7, k: 7, want: 1},
		{n: 57, k: 8, want: 1652411475},
		{n: 24, k: 8, want: 735471},
		{n: 60, k: 30, want: 118264581564861424},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%d choose %d", tt.n, tt.k), func(t *testing.T) {
			t.Parallel()

			if got := binomial(tt.n, tt.k); got != tt.want {
				t.Errorf("binomial(%d, %d) = %d, want %d", tt.n, tt.k, got, tt.want)
			}
		})
	}
}
