package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func mustParse(t *testing.T, text string) tree {
	t.Helper()
	parsed, err := parse(text)
	if err != nil {
		t.Fatalf("parse(%q): %v", text, err)
	}
	return parsed
}

// TestPlacements pins the matrix drawn in the README, column by column.
func TestPlacements(t *testing.T) {
	t.Parallel()

	got := map[string][]int{}
	for _, placed := range placements(mustParse(t, defaultExpression), instructions) {
		got[placed.pattern.name] = placed.covered
	}

	// Columns: 0 is load, 1 is add, 2 is a, 3 is 8.
	want := map[string][]int{
		"ldr":    {0},
		"ldrimm": {0, 1, 3},
		"ldrreg": {0, 1},
		"add":    {1},
		"addi":   {1, 3},
		"reg":    {2},
		"mov":    {3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("placements = %v, want %v", got, want)
	}
}

func TestPlacements_SameNodeSetKeepsCheapest(t *testing.T) {
	t.Parallel()

	reversed := make([]pattern, len(instructions))
	for i, p := range instructions {
		reversed[len(reversed)-1-i] = p
	}

	tests := []struct {
		name string
		set  []pattern
	}{
		{name: "cheap pattern first", set: instructions},
		{name: "cheap pattern last", set: reversed},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tiles := placements(mustParse(t, "add(a, b)"), tt.set)
			var names []string
			for _, placed := range tiles {
				names = append(names, placed.pattern.name)
			}
			if len(tiles) != 3 {
				t.Fatalf("got tiles %v, want add and two registers", names)
			}
			if tiles[0].pattern.name != "add" {
				t.Errorf("root tile is %s, want add", tiles[0].pattern.name)
			}
		})
	}
}

// TestSolve checks every number the README publishes for a single expression.
func TestSolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		wantRows   int
		wantTotal  int
		wantCost   int
		wantCode   []string
		wantResult string
	}{
		{
			name:       "register needs no instruction",
			expression: "a",
			wantRows:   1,
			wantTotal:  1,
			wantResult: "a",
		},
		{
			name:       "constant",
			expression: "5",
			wantRows:   1,
			wantTotal:  1,
			wantCost:   1,
			wantCode:   []string{"mov R0, #5"},
			wantResult: "R0",
		},
		{
			name:       "field of a structure",
			expression: "load(add(a, 8))",
			wantRows:   7,
			wantTotal:  4,
			wantCost:   4,
			wantCode:   []string{"ldr R0, [a, #8]"},
			wantResult: "R0",
		},
		{
			name:       "element of an array",
			expression: "load(add(a, shl(i, 3)))",
			wantRows:   9,
			wantTotal:  4,
			wantCost:   4,
			wantCode:   []string{"ldr R0, [a, i, lsl #3]"},
			wantResult: "R0",
		},
		{
			name:       "fused multiply and add",
			expression: "add(mul(a, b), c)",
			wantRows:   6,
			wantTotal:  2,
			wantCost:   3,
			wantCode:   []string{"madd R0, a, b, c"},
			wantResult: "R0",
		},
		{
			name:       "constant on the left is not an immediate",
			expression: "add(8, a)",
			wantRows:   3,
			wantTotal:  1,
			wantCost:   2,
			wantCode:   []string{"mov R0, #8", "add R1, R0, a"},
			wantResult: "R1",
		},
		{
			name:       "shift by a register has no pattern",
			expression: "shl(a, b)",
			wantRows:   2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found := solve(mustParse(t, tt.expression), instructions, 1)

			if found.rows != tt.wantRows {
				t.Errorf("rows = %d, want %d", found.rows, tt.wantRows)
			}
			if found.total != tt.wantTotal {
				t.Errorf("total = %d, want %d", found.total, tt.wantTotal)
			}
			if tt.wantTotal == 0 {
				if len(found.tilings) != 0 {
					t.Errorf("got %d tilings, want none", len(found.tilings))
				}
				return
			}
			if len(found.tilings) != 1 {
				t.Fatalf("got %d tilings, want 1", len(found.tilings))
			}
			best := found.tilings[0]
			if best.cost != tt.wantCost {
				t.Errorf("cost = %d, want %d", best.cost, tt.wantCost)
			}
			if got := codeLines(&best); !reflect.DeepEqual(got, tt.wantCode) {
				t.Errorf("code = %q, want %q", got, tt.wantCode)
			}
			if best.result != tt.wantResult {
				t.Errorf("result = %s, want %s", best.result, tt.wantResult)
			}
		})
	}
}

func codeLines(chosen *tiling) []string {
	var lines []string
	for _, line := range chosen.code {
		lines = append(lines, line.text)
	}
	return lines
}

// TestSolve_RowOrder pins the order of the rows in the README's decoding example.
func TestSolve_RowOrder(t *testing.T) {
	t.Parallel()

	found := solve(mustParse(t, defaultExpression), instructions, 1)

	var got [][]int
	for _, chosen := range found.tilings[0].tiles {
		got = append(got, chosen.covered)
	}
	if want := [][]int{{2}, {0, 1, 3}}; !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestSolve_LimitKeepsTheCheapestInOrder(t *testing.T) {
	t.Parallel()

	const expression = "load(add(mul(a, b), shl(c, 2)))"
	all := solve(mustParse(t, expression), instructions, 0)
	if all.total != len(all.tilings) {
		t.Fatalf("limit 0 kept %d of %d tilings", len(all.tilings), all.total)
	}

	for limit := 1; limit <= all.total+1; limit++ {
		found := solve(mustParse(t, expression), instructions, uint64(limit))

		want := all.tilings
		if limit < len(want) {
			want = want[:limit]
		}
		if len(found.tilings) != len(want) {
			t.Fatalf("limit %d kept %d tilings, want %d", limit, len(found.tilings), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(codeLines(&found.tilings[i]), codeLines(&want[i])) {
				t.Errorf("limit %d, tiling %d differs from the full list", limit, i)
			}
		}
	}
	for i := 1; i < len(all.tilings); i++ {
		if all.tilings[i-1].cost > all.tilings[i].cost {
			t.Errorf("tiling %d costs more than the next one", i-1)
		}
	}
}

// TestSolve_Growth checks the growth table in the README. Each level is a load
// from a register plus a constant, and it can be tiled in four ways. dlx lists
// the tilings, so the largest sizes are checked by dynamic programming alone.
func TestSolve_Growth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		levels     int
		wantNodes  int
		wantRows   int
		wantTotal  int
		wantCost   int
		useDLX     bool
		skipIfSlow bool
	}{
		{levels: 1, wantNodes: 4, wantRows: 7, wantTotal: 4, wantCost: 4, useDLX: true},
		{levels: 4, wantNodes: 13, wantRows: 25, wantTotal: 256, wantCost: 16, useDLX: true},
		{levels: 6, wantNodes: 19, wantRows: 37, wantTotal: 4096, wantCost: 24, useDLX: true},
		{levels: 8, wantNodes: 25, wantRows: 49, wantTotal: 65536, wantCost: 32, useDLX: true, skipIfSlow: true},
		{levels: 10, wantNodes: 31, wantRows: 61, wantTotal: 1048576, wantCost: 40},
		{levels: 11, wantNodes: 34, wantRows: 67, wantTotal: 4194304, wantCost: 44},
		{levels: 21, wantNodes: 64, wantRows: 127, wantTotal: 4398046511104, wantCost: 84},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(strconv.Itoa(tt.levels), func(t *testing.T) {
			t.Parallel()

			e := &expr{op: "reg", name: "a"}
			for i := 0; i < tt.levels; i++ {
				e = &expr{op: "load", left: &expr{op: "add", left: e, right: &expr{op: "const", value: 8}}}
			}

			if got := e.size(); got != tt.wantNodes {
				t.Errorf("nodes = %d, want %d", got, tt.wantNodes)
			}
			want := bestCost(e, map[*expr]choices{})
			if want.ways != tt.wantTotal || want.cost != tt.wantCost {
				t.Errorf("dynamic programming: %+v, want %d tilings at cost %d", want, tt.wantTotal, tt.wantCost)
			}
			parsed := mustParse(t, e.String())
			if got := len(placements(parsed, instructions)); got != tt.wantRows {
				t.Errorf("rows = %d, want %d", got, tt.wantRows)
			}
			if !tt.useDLX || (tt.skipIfSlow && testing.Short()) {
				return
			}
			found := solve(parsed, instructions, 1)
			if found.total != tt.wantTotal || found.tilings[0].cost != tt.wantCost {
				t.Errorf("dlx: %d tilings, cost %d", found.total, found.tilings[0].cost)
			}
		})
	}
}

// expr is the tests' own expression tree. The solver's parser, tree and pattern
// table never touch it: bestCost below works from its own copy of the rules.
type expr struct {
	op          string
	left, right *expr
	name        string // for a register
	value       uint64 // for a constant
}

func (e *expr) String() string {
	switch e.op {
	case "reg":
		return e.name
	case "const":
		return strconv.FormatUint(e.value, 10)
	case "load":
		return "load(" + e.left.String() + ")"
	}
	return e.op + "(" + e.left.String() + ", " + e.right.String() + ")"
}

func (e *expr) size() int {
	switch e.op {
	case "reg", "const":
		return 1
	case "load":
		return 1 + e.left.size()
	}
	return 1 + e.left.size() + e.right.size()
}

// randomExpr builds a tree of the given depth. One shift in twenty is by a
// register, which no instruction can do, so some trees have no tiling.
func randomExpr(r *rand.Rand, depth int) *expr {
	if depth == 0 || r.Intn(5) == 0 {
		return randomLeaf(r)
	}
	switch []string{"add", "add", "mul", "shl", "load"}[r.Intn(5)] {
	case "add":
		return &expr{op: "add", left: randomExpr(r, depth-1), right: randomExpr(r, depth-1)}
	case "mul":
		return &expr{op: "mul", left: randomExpr(r, depth-1), right: randomExpr(r, depth-1)}
	case "load":
		return &expr{op: "load", left: randomExpr(r, depth-1)}
	}
	amount := &expr{op: "const", value: uint64(r.Intn(8))}
	if r.Intn(20) == 0 {
		amount = &expr{op: "reg", name: "s"}
	}
	return &expr{op: "shl", left: randomExpr(r, depth-1), right: amount}
}

func randomLeaf(r *rand.Rand) *expr {
	if r.Intn(2) == 0 {
		return &expr{op: "const", value: uint64(r.Intn(100))}
	}
	return &expr{op: "reg", name: []string{"a", "b", "c", "d"}[r.Intn(4)]}
}

// choices gathers the ways to tile a node: the cheapest cost and how many.
type choices struct {
	cost int
	ways int
}

// add records a tile of the given cost whose operands are the given subtrees.
func (c *choices) add(cost int, operands ...choices) {
	ways := 1
	for _, operand := range operands {
		cost += operand.cost
		ways *= operand.ways
	}
	if ways == 0 {
		return
	}
	if c.ways == 0 || cost < c.cost {
		c.cost = cost
	}
	c.ways += ways
}

// bestCost is the dynamic programming method: the memo means it works out each
// node once, and at each node it tries the tiles that can sit there. It is
// written out rule by rule, without the pattern table, and adds the tilings up
// instead of listing them. adds is left out, as in the solver: tiles on the same
// nodes count once.
func bestCost(e *expr, memo map[*expr]choices) choices {
	if known, isKnown := memo[e]; isKnown {
		return known
	}
	var best choices
	switch e.op {
	case "reg":
		best.add(0)
	case "const":
		best.add(1)
	case "mul":
		best.add(3, bestCost(e.left, memo), bestCost(e.right, memo))
	case "shl":
		if e.right.op == "const" {
			best.add(1, bestCost(e.left, memo))
		}
	case "add":
		best.add(1, bestCost(e.left, memo), bestCost(e.right, memo))
		if e.right.op == "const" {
			best.add(1, bestCost(e.left, memo))
		}
		if e.left.op == "mul" {
			best.add(3, bestCost(e.left.left, memo), bestCost(e.left.right, memo), bestCost(e.right, memo))
		}
		if isShiftByConstant(e.right) {
			best.add(1, bestCost(e.left, memo), bestCost(e.right.left, memo))
		}
	case "load":
		best.add(4, bestCost(e.left, memo))
		if address := e.left; address.op == "add" {
			best.add(4, bestCost(address.left, memo), bestCost(address.right, memo))
			if address.right.op == "const" {
				best.add(4, bestCost(address.left, memo))
			}
			if isShiftByConstant(address.right) {
				best.add(4, bestCost(address.left, memo), bestCost(address.right.left, memo))
			}
		}
	}
	memo[e] = best
	return best
}

func isShiftByConstant(e *expr) bool {
	return e.op == "shl" && e.right.op == "const"
}

// memory stands in for a computer's memory: a fixed scramble of the address, so
// that reading the wrong address gives the wrong value.
func memory(address uint64) uint64 {
	return (address ^ 0x5bd1e995) * 0x9e3779b97f4a7c15
}

// registers holds the input values, chosen to be large and unrelated.
func registers() map[string]uint64 {
	return map[string]uint64{"a": 0x1234567, "b": 0xfedcba987, "c": 0x31415926, "d": 0x27182818, "s": 5}
}

// evaluate is the meaning of the expression, computed from the tree itself.
func evaluate(e *expr, regs map[string]uint64) uint64 {
	switch e.op {
	case "reg":
		return regs[e.name]
	case "const":
		return e.value
	case "load":
		return memory(evaluate(e.left, regs))
	case "add":
		return evaluate(e.left, regs) + evaluate(e.right, regs)
	case "mul":
		return evaluate(e.left, regs) * evaluate(e.right, regs)
	}
	return evaluate(e.left, regs) << evaluate(e.right, regs)
}

// execute runs a generated program as text, the way a very small machine
// would, and returns the value in the result register.
func execute(t *testing.T, chosen *tiling) uint64 {
	t.Helper()

	regs := registers()
	value := func(text string) uint64 {
		if strings.HasPrefix(text, "#") {
			number, err := strconv.ParseUint(text[1:], 10, 64)
			if err != nil {
				t.Fatalf("bad constant %q: %v", text, err)
			}
			return number
		}
		number, isSet := regs[text]
		if !isSet {
			t.Fatalf("register %s is read before it is written", text)
		}
		return number
	}

	for _, line := range chosen.code {
		fields := strings.FieldsFunc(line.text, func(c rune) bool { return strings.ContainsRune(" ,[]", c) })
		name, dest, args := fields[0], fields[1], fields[2:]
		switch {
		case name == "mov":
			regs[dest] = value(args[0])
		case name == "lsl":
			regs[dest] = value(args[0]) << value(args[1])
		case name == "mul":
			regs[dest] = value(args[0]) * value(args[1])
		case name == "madd":
			regs[dest] = value(args[0])*value(args[1]) + value(args[2])
		case name == "add" && len(args) == 2:
			regs[dest] = value(args[0]) + value(args[1])
		case name == "add":
			regs[dest] = value(args[0]) + value(args[1])<<value(args[3])
		case name == "ldr" && len(args) == 1:
			regs[dest] = memory(value(args[0]))
		case name == "ldr" && len(args) == 2:
			regs[dest] = memory(value(args[0]) + value(args[1]))
		case name == "ldr":
			regs[dest] = memory(value(args[0]) + value(args[1])<<value(args[3]))
		default:
			t.Fatalf("unknown instruction %q", line.text)
		}
	}
	return value(chosen.result)
}

// cheapestOverlappingCover returns the lowest cost of any set of tiles that
// covers each node at least once. It doesn't ask that the tiles join up, so it
// can only do better than a real program could. costs[m] is the cheapest way
// found so far to cover exactly the nodes in the bit mask m, and each tile
// extends every mask. Using a tile twice would change nothing but the cost.
func cheapestOverlappingCover(tiles []tile, nodes int) (int, bool) {
	const unreached = -1
	costs := make([]int, 1<<nodes)
	for m := range costs {
		costs[m] = unreached
	}
	costs[0] = 0
	for _, placed := range tiles {
		mask := 0
		for _, id := range placed.covered {
			mask |= 1 << id
		}
		for m, cost := range costs {
			if cost == unreached {
				continue
			}
			next := cost + placed.pattern.cost
			if known := costs[m|mask]; known == unreached || next < known {
				costs[m|mask] = next
			}
		}
	}
	best := costs[len(costs)-1]
	return best, best != unreached
}

// TestSolve_OverlapsDontHelp checks the README's claim that requiring each node
// to be covered exactly once loses nothing. The published count is for this seed
// and the full size cap. The table of costs doubles with every node, so the
// short run caps the size lower.
func TestSolve_OverlapsDontHelp(t *testing.T) {
	t.Parallel()

	mostNodes, trees := 18, 5000
	if testing.Short() {
		mostNodes, trees = 14, 500
	}

	r := rand.New(rand.NewSource(2))
	compared := 0
	for i := 0; i < trees; i++ {
		e := randomExpr(r, 5)
		if e.size() > mostNodes {
			continue
		}
		compared++
		parsed := mustParse(t, e.String())

		found := solve(parsed, instructions, 1)
		cost, isCovered := cheapestOverlappingCover(placements(parsed, instructions), len(parsed.nodes))

		if isCovered != (found.total > 0) {
			t.Fatalf("%s: covered with overlaps: %v, tilings: %d", e, isCovered, found.total)
		}
		if isCovered && cost != found.tilings[0].cost {
			t.Fatalf("%s: overlaps cost %d, exact cover %d", e, cost, found.tilings[0].cost)
		}
	}

	if testing.Short() {
		return
	}
	if want := 3432; compared != want {
		t.Errorf("compared %d trees, want %d", compared, want)
	}
}

// TestSolve_RandomTrees compares dlx with the dynamic programming method and
// runs every program of every tiling. The published numbers are for the seed
// and sizes below.
func TestSolve_RandomTrees(t *testing.T) {
	t.Parallel()

	trees := 2000
	if testing.Short() {
		trees = 200
	}

	r := rand.New(rand.NewSource(1))
	regs := registers()
	var mostNodes, mostRows, mostTilings, allTilings, withoutTiling int
	for i := 0; i < trees; i++ {
		e := randomExpr(r, 5)
		text := e.String()
		want := bestCost(e, map[*expr]choices{})

		found := solve(mustParse(t, text), instructions, 0)

		if found.total != want.ways {
			t.Fatalf("%s: dlx found %d tilings, dynamic programming %d", text, found.total, want.ways)
		}
		if want.ways == 0 {
			withoutTiling++
			continue
		}
		if found.tilings[0].cost != want.cost {
			t.Fatalf("%s: dlx found cost %d, dynamic programming %d", text, found.tilings[0].cost, want.cost)
		}
		wantValue := evaluate(e, regs)
		for j := range found.tilings {
			if got := execute(t, &found.tilings[j]); got != wantValue {
				t.Fatalf("%s: tiling %d computes %d, want %d:\n%s",
					text, j, got, wantValue, strings.Join(codeLines(&found.tilings[j]), "\n"))
			}
		}

		allTilings += found.total
		if size := e.size(); size > mostNodes {
			mostNodes = size
		}
		if found.rows > mostRows {
			mostRows = found.rows
		}
		if found.total > mostTilings {
			mostTilings = found.total
		}
	}

	if testing.Short() {
		return
	}
	got := fmt.Sprintf("nodes %d, rows %d, most tilings %d, all tilings %d, no tiling %d",
		mostNodes, mostRows, mostTilings, allTilings, withoutTiling)
	if want := "nodes 48, rows 53, most tilings 512, all tilings 14045, no tiling 136"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
