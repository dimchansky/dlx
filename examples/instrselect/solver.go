package main

import (
	"fmt"
	"sort"

	"github.com/dimchansky/dlx"
)

// opOperand marks the place in a shape where a tile stops: whatever subtree is
// there is computed by another tile, and this one reads its result.
const opOperand = "operand"

// shape is the part of a tree that an instruction replaces.
type shape struct {
	op   string
	kids []shape
}

func op(name string, kids ...shape) shape { return shape{op: name, kids: kids} }

// pattern is one instruction of the example machine. The costs are invented.
type pattern struct {
	name  string
	cost  int
	shape shape
	// asm is a format for the instruction text. It takes the destination
	// register, then one value for each operand and constant of the shape, from
	// left to right. It is empty for a register, which needs no instruction.
	asm string
}

// instructions is the example machine. It looks like a small part of ARM64.
//
// adds does the same job as add at a higher cost, so that there are two
// patterns with the same node set. A real instruction set has many such pairs.
var instructions = []pattern{
	{"reg", 0, op(opReg), ""},
	{"mov", 1, op(opConst), "mov %s, %s"},
	{"add", 1, op(opAdd, op(opOperand), op(opOperand)), "add %s, %s, %s"},
	{"adds", 2, op(opAdd, op(opOperand), op(opOperand)), "adds %s, %s, %s"},
	{"addi", 1, op(opAdd, op(opOperand), op(opConst)), "add %s, %s, %s"},
	{"mul", 3, op(opMul, op(opOperand), op(opOperand)), "mul %s, %s, %s"},
	{"madd", 3, op(opAdd, op(opMul, op(opOperand), op(opOperand)), op(opOperand)), "madd %s, %s, %s, %s"},
	{"lsl", 1, op(opShl, op(opOperand), op(opConst)), "lsl %s, %s, %s"},
	{"addshift", 1, op(opAdd, op(opOperand), op(opShl, op(opOperand), op(opConst))), "add %s, %s, %s, lsl %s"},
	{"ldr", 4, op(opLoad, op(opOperand)), "ldr %s, [%s]"},
	{"ldrimm", 4, op(opLoad, op(opAdd, op(opOperand), op(opConst))), "ldr %s, [%s, %s]"},
	{"ldrreg", 4, op(opLoad, op(opAdd, op(opOperand), op(opOperand))), "ldr %s, [%s, %s]"},
	{"ldrshift", 4, op(opLoad, op(opAdd, op(opOperand), op(opShl, op(opOperand), op(opConst)))), "ldr %s, [%s, %s, lsl %s]"},
}

// argument is a value in an instruction: the result of the tile below an
// operand, or a constant that the tile covers itself.
type argument struct {
	node      *node
	isOperand bool
}

// tile is a pattern placed on a tree. It is one row of the matrix.
type tile struct {
	pattern pattern
	root    *node
	covered []int // the node ids it covers, in increasing order; the root is first
	args    []argument
}

// matches reports whether s fits at n, and records what the tile covers. A
// shape and the node it lies on visit nodes in preorder, so covered stays
// sorted. Nodes of the same kind have the same number of operands.
func (t *tile) matches(s shape, n *node) bool {
	if s.op == opOperand {
		t.args = append(t.args, argument{node: n, isOperand: true})
		return true
	}
	if s.op != n.op {
		return false
	}
	t.covered = append(t.covered, n.id)
	if n.op == opConst {
		t.args = append(t.args, argument{node: n})
	}
	for i := range s.kids {
		if !t.matches(s.kids[i], n.kids[i]) {
			return false
		}
	}
	return true
}

// placements returns every distinct way to put a pattern on the tree, in order
// of first appearance. dlx reports a row by its columns alone, so two patterns
// on the same node set would be two rows that can't be told apart. Only the
// cheaper can be in the cheapest tiling, so each node set keeps one tile.
func placements(t tree, set []pattern) []tile {
	var tiles []tile
	index := map[string]int{}
	for _, n := range t.nodes {
		for _, p := range set {
			candidate := tile{pattern: p, root: n}
			if !candidate.matches(p.shape, n) {
				continue
			}
			key := fmt.Sprint(candidate.covered)
			i, isSeen := index[key]
			switch {
			case !isSeen:
				index[key] = len(tiles)
				tiles = append(tiles, candidate)
			case p.cost < tiles[i].pattern.cost:
				tiles[i] = candidate
			}
		}
	}
	return tiles
}

// tiling is a set of tiles that covers every node of the tree exactly once.
type tiling struct {
	cost   int
	tiles  []tile
	code   []instruction // operands first, so that it can run from top to bottom
	result string        // the register that holds the value of the tree
}

// instruction is one line of the generated program and the tile it came from.
type instruction struct {
	text string
	tile tile
}

// solution is what solve finds.
type solution struct {
	rows    int      // distinct tiles placed on the tree
	total   int      // tilings found
	tilings []tiling // the cheapest, at most limit of them unless limit is zero
}

// solve finds every tiling of the tree and returns the limit cheapest. Tilings
// of equal cost stay in the order the search found them.
//
// Each tree node is a column and each tile is a row, so an exact cover is a
// tiling. A tile that stops at an operand doesn't cover it, and the tile that
// does must start there: if it started higher, it would also cover the node
// above the operand, which already has a tile. So tiles always join up, and the
// matrix needs no rule for that.
//
// dlx has no costs, so finding the cheapest tiling means going through them all
// and comparing totals here.
func solve(t tree, set []pattern, limit uint64) solution {
	tiles := placements(t, set)
	m := dlx.NewMatrix(len(t.nodes))
	byColumns := make(map[string]tile, len(tiles))
	for _, placed := range tiles {
		m.AddRow(placed.covered...)
		byColumns[fmt.Sprint(placed.covered)] = placed
	}

	found := solution{rows: len(tiles)}
	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		found.total++
		next := tiling{tiles: make([]tile, len(exactCover))}
		for i, row := range exactCover {
			next.tiles[i] = byColumns[fmt.Sprint(row)]
			next.cost += next.tiles[i].pattern.cost
		}
		found.tilings = keepCheapest(found.tilings, next, limit)
		return false
	}))

	for i := range found.tilings {
		found.tilings[i].decode(t.root)
	}
	return found
}

// keepCheapest adds next to tilings, which is sorted by cost, and drops the
// last one if there are more than limit. A tie goes after the tilings already
// there. Without a limit it keeps them all, so memory follows the total.
func keepCheapest(tilings []tiling, next tiling, limit uint64) []tiling {
	at := sort.Search(len(tilings), func(i int) bool { return tilings[i].cost > next.cost })
	if limit != 0 && uint64(at) == limit {
		return tilings
	}
	tilings = append(tilings, tiling{})
	copy(tilings[at+1:], tilings[at:])
	tilings[at] = next
	if limit != 0 && uint64(len(tilings)) > limit {
		tilings = tilings[:limit]
	}
	return tilings
}

// decode turns the chosen tiles into a program, starting from the tile at the
// root of the tree.
func (t *tiling) decode(root *node) {
	byRoot := make(map[*node]tile, len(t.tiles))
	for _, chosen := range t.tiles {
		byRoot[chosen.root] = chosen
	}
	t.result = t.emit(root, byRoot)
}

// emit writes the program for the subtree at n and returns the name of the
// register that holds its value. Registers are never reused: choosing real
// ones is a later step in a compiler.
func (t *tiling) emit(n *node, byRoot map[*node]tile) string {
	chosen := byRoot[n]
	if chosen.pattern.asm == "" {
		return n.label
	}
	values := []any{""}
	for _, arg := range chosen.args {
		if arg.isOperand {
			values = append(values, t.emit(arg.node, byRoot))
		} else {
			values = append(values, "#"+arg.node.label)
		}
	}
	dest := fmt.Sprintf("R%d", len(t.code))
	values[0] = dest
	t.code = append(t.code, instruction{text: fmt.Sprintf(chosen.pattern.asm, values...), tile: chosen})
	return dest
}
