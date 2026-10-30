package main

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		want      string // the tree printed back
		wantLabel []string
		wantOps   []string
	}{
		{
			name:      "register",
			text:      "a",
			want:      "a",
			wantLabel: []string{"a"},
			wantOps:   []string{opReg},
		},
		{
			name:      "constant loses leading zeros",
			text:      "007",
			want:      "7",
			wantLabel: []string{"7"},
			wantOps:   []string{opConst},
		},
		{
			name:      "largest constant",
			text:      "18446744073709551615",
			want:      "18446744073709551615",
			wantLabel: []string{"18446744073709551615"},
			wantOps:   []string{opConst},
		},
		{
			name:      "nodes come in preorder",
			text:      "load(add(a, shl(i, 3)))",
			want:      "load(add(a, shl(i, 3)))",
			wantLabel: []string{"load", "add", "a", "shl", "i", "3"},
			wantOps:   []string{opLoad, opAdd, opReg, opShl, opReg, opConst},
		},
		{
			name:      "spaces are optional",
			text:      "  mul ( x2 ,add(b,1) ) ",
			want:      "mul(x2, add(b, 1))",
			wantLabel: []string{"mul", "x2", "add", "b", "1"},
			wantOps:   []string{opMul, opReg, opAdd, opReg, opConst},
		},
		{
			name:      "an operator's name can be a register",
			text:      "add(add, 1)",
			want:      "add(add, 1)",
			wantLabel: []string{"add", "add", "1"},
			wantOps:   []string{opAdd, opReg, opConst},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := parse(tt.text)
			if err != nil {
				t.Fatalf("parse(%q): %v", tt.text, err)
			}
			if got := parsed.root.String(); got != tt.want {
				t.Errorf("printed back as %q, want %q", got, tt.want)
			}
			var labels, ops []string
			for i, n := range parsed.nodes {
				if n.id != i {
					t.Errorf("node %d has id %d", i, n.id)
				}
				labels = append(labels, n.label)
				ops = append(ops, n.op)
			}
			if !reflect.DeepEqual(labels, tt.wantLabel) {
				t.Errorf("labels = %q, want %q", labels, tt.wantLabel)
			}
			if !reflect.DeepEqual(ops, tt.wantOps) {
				t.Errorf("ops = %q, want %q", ops, tt.wantOps)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: "", want: "column 1: expected a name or a number"},
		{name: "only spaces", text: "  ", want: "column 3: expected a name or a number"},
		{name: "uppercase", text: "A", want: "column 1: expected a name or a number"},
		{name: "unknown operator", text: "sub(a, b)", want: `column 5: unknown operator "sub"`},
		{name: "trailing text", text: "a b", want: `column 3: unexpected 'b' after the expression`},
		{name: "missing operand", text: "add(a)", want: "column 6: add needs 2 operands"},
		{name: "extra operand", text: "load(a, b)", want: `column 7: expected ')' to close load`},
		{name: "unclosed", text: "load(a", want: `column 7: expected ')' to close load`},
		{name: "operand is not an expression", text: "add(a, )", want: "column 8: expected a name or a number"},
		{name: "constant too big", text: "18446744073709551616", want: "column 21: constant 18446744073709551616 does not fit in 64 bits"},
		{name: "digit then letter", text: "8a", want: `column 3: invalid constant "8a"`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parse(tt.text)
			if err == nil {
				t.Fatalf("parse(%q) succeeded, want an error", tt.text)
			}
			if err.Error() != tt.want {
				t.Errorf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

// TestParse_NodeCap builds a chain of loads, which is also the deepest tree the
// recursion can meet.
func TestParse_NodeCap(t *testing.T) {
	t.Parallel()

	chain := func(nodes int) string {
		return strings.Repeat("load(", nodes-1) + "a" + strings.Repeat(")", nodes-1)
	}

	if _, err := parse(chain(maxNodes)); err != nil {
		t.Errorf("%d nodes: %v", maxNodes, err)
	}
	_, err := parse(chain(maxNodes + 1))
	want := "the expression has more than " + strconv.Itoa(maxNodes) + " nodes"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%d nodes: error = %v, want it to contain %q", maxNodes+1, err, want)
	}
}
