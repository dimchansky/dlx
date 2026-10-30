package main

import (
	"fmt"
	"math/bits"
	"reflect"
	"sort"
	"testing"
)

// The tables below are copied from the papers, using the papers' ray numbers.
// The rays in rays.go are recalled data, so these tests compute orthogonality
// from those coordinates and compare the outcome with what was published.

// gouldAravindTriads and gouldAravindDyads are table 1 of arXiv:0909.4502.
var (
	gouldAravindTriads = [][]int{
		{1, 2, 3},
		{1, 4, 5},
		{1, 26, 33},
		{1, 29, 30},
		{2, 6, 7},
		{2, 22, 32},
		{2, 25, 31},
		{3, 8, 9},
		{3, 23, 28},
		{3, 24, 27},
		{4, 10, 13},
		{5, 11, 12},
		{6, 14, 17},
		{7, 15, 16},
		{8, 18, 21},
		{9, 19, 20},
	}
	gouldAravindDyads = [][]int{
		{10, 24},
		{10, 25},
		{11, 23},
		{11, 25},
		{12, 22},
		{12, 24},
		{13, 22},
		{13, 23},
		{14, 28},
		{14, 29},
		{15, 27},
		{15, 29},
		{16, 26},
		{16, 28},
		{17, 26},
		{17, 27},
		{18, 32},
		{18, 33},
		{19, 31},
		{19, 33},
		{20, 30},
		{20, 32},
		{21, 30},
		{21, 31},
	}
)

// waegellAravindBases is figure 1 of arXiv:1103.6058.
var waegellAravindBases = [][]int{
	{1, 2, 3, 4},
	{5, 6, 7, 8},
	{9, 10, 11, 12},
	{13, 14, 15, 16},
	{17, 18, 19, 20},
	{21, 22, 23, 24},
	{1, 2, 15, 16},
	{3, 4, 13, 14},
	{5, 8, 21, 23},
	{6, 7, 22, 24},
	{9, 11, 18, 19},
	{10, 12, 17, 20},
	{2, 4, 19, 20},
	{1, 3, 17, 18},
	{2, 3, 21, 22},
	{1, 4, 23, 24},
	{10, 11, 21, 24},
	{9, 12, 22, 23},
	{6, 8, 17, 19},
	{5, 7, 18, 20},
	{7, 8, 13, 15},
	{5, 6, 14, 16},
	{11, 12, 14, 15},
	{9, 10, 13, 16},
}

// cabelloEquations is equations (1) to (9) of arXiv:quant-ph/9706009.
var cabelloEquations = [][][]int{
	{{0, 0, 0, 1}, {0, 0, 1, 0}, {1, 1, 0, 0}, {1, -1, 0, 0}},
	{{0, 0, 0, 1}, {0, 1, 0, 0}, {1, 0, 1, 0}, {1, 0, -1, 0}},
	{{1, -1, 1, -1}, {1, -1, -1, 1}, {1, 1, 0, 0}, {0, 0, 1, 1}},
	{{1, -1, 1, -1}, {1, 1, 1, 1}, {1, 0, -1, 0}, {0, 1, 0, -1}},
	{{0, 0, 1, 0}, {0, 1, 0, 0}, {1, 0, 0, 1}, {1, 0, 0, -1}},
	{{1, -1, -1, 1}, {1, 1, 1, 1}, {1, 0, 0, -1}, {0, 1, -1, 0}},
	{{1, 1, -1, 1}, {1, 1, 1, -1}, {1, -1, 0, 0}, {0, 0, 1, 1}},
	{{1, 1, -1, 1}, {-1, 1, 1, 1}, {1, 0, 1, 0}, {0, 1, 0, -1}},
	{{1, 1, 1, -1}, {-1, 1, 1, 1}, {1, 0, 0, 1}, {0, 1, -1, 0}},
}

func TestSurd_Times(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		x, y surd
		want surd
	}{
		{name: "by zero", x: root, y: zero, want: zero},
		{name: "by one", x: negativeRoot, y: one, want: negativeRoot},
		{name: "root squared", x: root, y: root, want: surd{a: 2}},
		{name: "signs cancel", x: negativeRoot, y: negativeRoot, want: surd{a: 2}},
		{name: "mixed", x: surd{1, 1}, y: surd{1, -1}, want: surd{a: -1}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.x.times(tt.y); got != tt.want {
				t.Errorf("%v times %v = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestSurd_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		x    surd
		want string
	}{
		{x: zero, want: "0"},
		{x: minus, want: "-1"},
		{x: root, want: "√2"},
		{x: negativeRoot, want: "-√2"},
		{x: surd{3, -2}, want: "3-2√2"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.x.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRay_Perpendicular(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		u, v ray
		want bool
	}{
		{name: "axes", u: ray{one, zero, zero}, v: ray{zero, one, zero}, want: true},
		{name: "same ray", u: ray{one, one, zero}, v: ray{one, one, zero}, want: false},
		{name: "roots cancel", u: ray{root, minus, zero}, v: ray{one, root, zero}, want: true},
		{name: "roots square to whole numbers", u: ray{root, zero, one}, v: ray{root, zero, minus}, want: false},
		{name: "whole part left over", u: ray{root, one, zero}, v: ray{one, one, zero}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.u.perpendicular(tt.v); got != tt.want {
				t.Errorf("%v perpendicular to %v = %t, want %t", tt.u, tt.v, got, tt.want)
			}
		})
	}
}

// TestRaySets_FollowDefiningRules builds each set again from the rule that
// defines it, and checks each set has no ray listed twice.
func TestRaySets_FollowDefiningRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rays  []ray
		want  [][]surd // patterns; every order and sign of each is in the set
		count int
	}{
		{
			name:  "peres33",
			rays:  peres33,
			want:  [][]surd{{zero, zero, one}, {zero, one, one}, {zero, one, root}, {one, one, root}},
			count: 33,
		},
		{
			name:  "peres24",
			rays:  peres24,
			want:  [][]surd{{zero, zero, zero, one}, {zero, zero, one, one}, {one, one, one, one}},
			count: 24,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := make(map[string]bool)
			for _, r := range tt.rays {
				key := canonical(r)
				if got[key] {
					t.Errorf("ray %v is listed twice", r)
				}
				got[key] = true
			}

			want := make(map[string]bool)
			for _, pattern := range tt.want {
				for _, r := range arrangements(pattern) {
					want[canonical(r)] = true
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("rays differ from the rule:\n got %v\nwant %v", got, want)
			}
			if len(got) != tt.count {
				t.Errorf("%d rays, want %d", len(got), tt.count)
			}
		})
	}
}

// arrangements lists every order of the coordinates with every choice of signs.
func arrangements(pattern []surd) []ray {
	var all []ray
	var place func(r ray, left []surd)
	place = func(r ray, left []surd) {
		if len(left) == 0 {
			all = append(all, append(ray(nil), r...))
			return
		}
		for i, x := range left {
			rest := append(append([]surd(nil), left[:i]...), left[i+1:]...)
			place(append(r, x), rest)
			place(append(r, surd{-x.a, -x.b}), rest)
		}
	}
	place(nil, pattern)
	return all
}

// canonical names a ray so that a point and its opposite get the same name.
func canonical(r ray) string {
	for _, x := range r {
		if x == zero {
			continue
		}
		if x.a < 0 || x.b < 0 {
			flipped := make(ray, len(r))
			for i, y := range r {
				flipped[i] = surd{-y.a, -y.b}
			}
			return flipped.String()
		}
		break
	}
	return r.String()
}

func TestPeres33_MatchesGouldAravind(t *testing.T) {
	t.Parallel()

	p := newProblem(peres33, 3, noRay)
	if got := fromOne(gouldAravindTriads); !equalGroups(p.bases, got) {
		t.Errorf("bases = %v, want table 1 of the paper: %v", p.bases, got)
	}
	if got := fromOne(gouldAravindDyads); !equalGroups(p.pairs, got) {
		t.Errorf("pairs = %v, want table 1 of the paper: %v", p.pairs, got)
	}
}

func TestPeres24_MatchesWaegellAravind(t *testing.T) {
	t.Parallel()

	p := newProblem(peres24, 4, noRay)
	if got := fromOne(waegellAravindBases); !equalGroups(p.bases, got) {
		t.Errorf("bases = %v, want figure 1 of the paper: %v", p.bases, got)
	}
	if len(p.pairs) != 0 {
		t.Errorf("pairs = %v, want none", p.pairs)
	}

	// The paper says each ray lies in four bases, and that the other rays in
	// those bases are nine rays, three of them meeting it twice and six once.
	for r := range peres24 {
		bases := 0
		companions := make(map[int]int)
		for _, basis := range p.bases {
			if !contains(basis, r) {
				continue
			}
			bases++
			for _, other := range basis {
				if other != r {
					companions[other]++
				}
			}
		}
		twice := 0
		for _, seen := range companions {
			if seen == 2 {
				twice++
			}
		}
		if bases != 4 || len(companions) != 9 || twice != 3 {
			t.Errorf("ray %d: %d bases, %d companions, %d seen twice; want 4, 9 and 3",
				r+1, bases, len(companions), twice)
		}
	}
}

func TestCabello18_MatchesEquations(t *testing.T) {
	t.Parallel()

	numberOf := make(map[string]int)
	for i, r := range cabello18 {
		numberOf[canonical(r)] = i
	}

	var equations [][]int
	uses := make(map[int]int)
	for _, equation := range cabelloEquations {
		for i, u := range equation {
			for _, v := range equation[i+1:] {
				if product := dotInts(u, v); product != 0 {
					t.Errorf("%v and %v: product %d, want 0", u, v, product)
				}
			}
		}

		var group []int
		for _, vector := range equation {
			r, ok := numberOf[canonical(rayOf(vector))]
			if !ok {
				t.Fatalf("vector %v is not in the set", vector)
			}
			group = append(group, r)
			uses[r]++
		}
		equations = append(equations, group)
	}

	if len(numberOf) != 18 {
		t.Errorf("%d rays, want 18", len(numberOf))
	}
	for r := range cabello18 {
		if uses[r] != 2 {
			t.Errorf("ray %d is in %d equations, want 2", r+1, uses[r])
		}
	}
	if p := newProblem(cabello18, 4, noRay); !equalGroups(p.bases, equations) {
		t.Errorf("bases = %v, want the equations: %v", p.bases, equations)
	}

	// The paper says each ray is orthogonal to seven others of the set.
	for i := range cabello18 {
		count := 0
		for j := range cabello18 {
			if i != j && cabello18[i].perpendicular(cabello18[j]) {
				count++
			}
		}
		if count != 7 {
			t.Errorf("ray %d is perpendicular to %d rays, want 7", i+1, count)
		}
	}
}

func TestCabello18_IsInsidePeres24(t *testing.T) {
	t.Parallel()

	inside := make(map[string]bool)
	for _, r := range peres24 {
		inside[canonical(r)] = true
	}
	for _, r := range cabello18 {
		if !inside[canonical(r)] {
			t.Errorf("ray %v is not one of the 24", r)
		}
	}
}

// TestPeres24_ParityProofs recounts table 2 of arXiv:1103.6058. A parity proof
// is an odd number of bases in which every ray occurs an even number of times.
// This walks through all 2^24 groups of bases, changing one basis at a time.
func TestPeres24_ParityProofs(t *testing.T) {
	t.Parallel()

	masks := make([]uint32, len(waegellAravindBases))
	for i, basis := range waegellAravindBases {
		for _, r := range basis {
			masks[i] |= 1 << (r - 1)
		}
	}

	found := make(map[[2]int]int)
	var chosen, odd uint32
	for step := uint32(1); step < 1<<24; step++ {
		flip := bits.TrailingZeros32(step)
		chosen ^= 1 << flip
		odd ^= masks[flip]
		if odd != 0 || bits.OnesCount32(chosen)%2 == 0 {
			continue
		}
		var rays uint32
		for i, mask := range masks {
			if chosen&(1<<i) != 0 {
				rays |= mask
			}
		}
		found[[2]int{bits.OnesCount32(rays), bits.OnesCount32(chosen)}]++
	}

	want := map[[2]int]int{{18, 9}: 16, {20, 11}: 240, {22, 13}: 240, {24, 15}: 16}
	if !reflect.DeepEqual(found, want) {
		t.Errorf("parity proofs by rays and bases = %v, want %v", found, want)
	}
}

// dotInts is a product of coordinates that shares no code with ray.perpendicular.
func dotInts(u, v []int) int {
	sum := 0
	for i := range u {
		sum += u[i] * v[i]
	}
	return sum
}

func rayOf(vector []int) ray {
	r := make(ray, len(vector))
	for i, x := range vector {
		r[i] = surd{a: x}
	}
	return r
}

// fromOne converts the papers' ray numbers, counted from 1, to those used here.
func fromOne(groups [][]int) [][]int {
	converted := make([][]int, len(groups))
	for i, group := range groups {
		converted[i] = make([]int, len(group))
		for j, r := range group {
			converted[i][j] = r - 1
		}
	}
	return converted
}

// equalGroups compares two lists of groups, ignoring the order of each.
func equalGroups(a, b [][]int) bool {
	return reflect.DeepEqual(sortedGroups(a), sortedGroups(b))
}

func sortedGroups(groups [][]int) []string {
	names := make([]string, len(groups))
	for i, group := range groups {
		sorted := append([]int(nil), group...)
		sort.Ints(sorted)
		names[i] = fmt.Sprint(sorted)
	}
	sort.Strings(names)
	return names
}

func contains(group []int, r int) bool {
	for _, x := range group {
		if x == r {
			return true
		}
	}
	return false
}
