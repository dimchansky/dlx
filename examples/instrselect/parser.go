package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// maxNodes caps the size of an expression. It only stops the program building
// a huge matrix and bounds the recursion: the number of tilings can double with
// every few nodes, so trees near the cap may take far too long to enumerate.
const maxNodes = 64

// The kinds of tree node. A pattern shape uses the same names.
const (
	opAdd   = "add"
	opMul   = "mul"
	opShl   = "shl"
	opLoad  = "load"
	opReg   = "reg"
	opConst = "const"
)

// operators maps each operator to its number of operands.
var operators = map[string]int{opAdd: 2, opMul: 2, opShl: 2, opLoad: 1}

// node is one node of an expression tree.
type node struct {
	id    int    // position in the tree's preorder, which is also its matrix column
	op    string // one of the op constants
	label string // what to print: the operator, the register name or the number
	kids  []*node
}

// String returns the node's subtree in the input syntax.
func (n *node) String() string {
	if len(n.kids) == 0 {
		return n.label
	}
	kids := make([]string, len(n.kids))
	for i, kid := range n.kids {
		kids[i] = kid.String()
	}
	return n.label + "(" + strings.Join(kids, ", ") + ")"
}

// tree is a parsed expression. nodes is in preorder: a node comes before its
// operands, and the left operand's subtree comes before the right one's.
type tree struct {
	root  *node
	nodes []*node
}

// parse reads an expression such as load(add(a, shl(i, 3))). A lowercase name
// followed by "(" is an operator, any other name is a register, and a number
// is a constant.
func parse(text string) (tree, error) {
	p := parser{text: text}
	root, err := p.expression()
	if err != nil {
		return tree{}, err
	}
	p.skipSpaces()
	if p.pos != len(p.text) {
		next, _ := utf8.DecodeRuneInString(p.text[p.pos:])
		return tree{}, p.errorf("unexpected %q after the expression", next)
	}
	return tree{root: root, nodes: p.nodes}, nil
}

// parser is a recursive descent parser. The node cap bounds its recursion.
type parser struct {
	text  string
	pos   int
	nodes []*node
}

// errorf returns an error that names the current position, counted from 1.
func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("column %d: %s", p.pos+1, fmt.Sprintf(format, args...))
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.text) && p.text[p.pos] == ' ' {
		p.pos++
	}
}

// accept skips spaces, then consumes c if it comes next.
func (p *parser) accept(c byte) bool {
	p.skipSpaces()
	if p.pos < len(p.text) && p.text[p.pos] == c {
		p.pos++
		return true
	}
	return false
}

// expression parses one subtree and appends its nodes in preorder.
func (p *parser) expression() (*node, error) {
	p.skipSpaces()
	if len(p.nodes) == maxNodes {
		return nil, p.errorf("the expression has more than %d nodes", maxNodes)
	}
	word := p.word()
	switch {
	case word == "":
		return nil, p.errorf("expected a name or a number")
	case isDigit(word[0]):
		return p.constant(word)
	case p.accept('('):
		return p.operator(word)
	}
	n := &node{id: len(p.nodes), op: opReg, label: word}
	p.nodes = append(p.nodes, n)
	return n, nil
}

// word consumes a run of lowercase letters and digits.
func (p *parser) word() string {
	start := p.pos
	for p.pos < len(p.text) && (isDigit(p.text[p.pos]) || isLower(p.text[p.pos])) {
		p.pos++
	}
	return p.text[start:p.pos]
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isLower(c byte) bool { return 'a' <= c && c <= 'z' }

func (p *parser) constant(word string) (*node, error) {
	value, err := strconv.ParseUint(word, 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return nil, p.errorf("constant %s does not fit in 64 bits", word)
		}
		return nil, p.errorf("invalid constant %q", word)
	}
	n := &node{id: len(p.nodes), op: opConst, label: strconv.FormatUint(value, 10)}
	p.nodes = append(p.nodes, n)
	return n, nil
}

// operator parses the operands of word, whose "(" has been consumed.
func (p *parser) operator(word string) (*node, error) {
	arity, isKnown := operators[word]
	if !isKnown {
		return nil, p.errorf("unknown operator %q", word)
	}
	n := &node{id: len(p.nodes), op: word, label: word}
	p.nodes = append(p.nodes, n)
	for i := 0; i < arity; i++ {
		if i > 0 && !p.accept(',') {
			return nil, p.errorf("%s needs %d operands", word, arity)
		}
		kid, err := p.expression()
		if err != nil {
			return nil, err
		}
		n.kids = append(n.kids, kid)
	}
	if !p.accept(')') {
		return nil, p.errorf("expected %q to close %s", ')', word)
	}
	return n, nil
}
