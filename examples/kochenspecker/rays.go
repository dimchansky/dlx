package main

import (
	"fmt"
	"strconv"
	"strings"
)

// surd is the exact number a + b√2. Coordinates that hold √2 cannot be stored
// exactly as floating point numbers, and this example depends on telling apart
// "exactly perpendicular" from "very nearly". The sum a + b√2 is zero only when
// a and b are both zero, because √2 is not a fraction.
type surd struct{ a, b int }

var (
	zero         = surd{}
	one          = surd{a: 1}
	minus        = surd{a: -1}
	root         = surd{b: 1}
	negativeRoot = surd{b: -1}
)

func (x surd) plus(y surd) surd {
	return surd{x.a + y.a, x.b + y.b}
}

// times multiplies out (a + b√2)(c + d√2), using √2·√2 = 2.
func (x surd) times(y surd) surd {
	return surd{x.a*y.a + 2*x.b*y.b, x.a*y.b + x.b*y.a}
}

func (x surd) String() string {
	switch x {
	case root:
		return "√2"
	case negativeRoot:
		return "-√2"
	}
	if x.b == 0 {
		return strconv.Itoa(x.a)
	}
	return fmt.Sprintf("%d%+d√2", x.a, x.b)
}

// ray is a line through the origin, given by the coordinates of one point on
// it. Any nonzero multiple of those coordinates, negative ones included, is the
// same ray.
type ray []surd

// perpendicular reports whether two rays meet at a right angle, which they do
// exactly when the sum of the products of their coordinates is zero.
func (u ray) perpendicular(v ray) bool {
	sum := zero
	for i := range u {
		sum = sum.plus(u[i].times(v[i]))
	}
	return sum == zero
}

func (u ray) String() string {
	parts := make([]string, len(u))
	for i, x := range u {
		parts[i] = x.String()
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// raySet is a published set of rays. Its ray numbers, which count from 1 in all
// output, follow the source given for each set.
type raySet struct {
	name      string
	dimension int
	rays      []ray
}

// raySets lists the sets in the order the usage text names them.
var raySets = []raySet{
	{"peres33", 3, peres33},
	{"peres24", 4, peres24},
	{"cabello18", 4, cabello18},
}

// peres33 holds the 33 rays of Peres in three dimensions, numbered as in
// equation (1) of Gould and Aravind, arXiv:0909.4502.
var peres33 = []ray{
	{one, zero, zero},
	{zero, one, zero},
	{zero, zero, one},
	{zero, one, one},
	{zero, one, minus},
	{one, zero, one},
	{one, zero, minus},
	{one, minus, zero},
	{one, one, zero},
	{root, minus, one},
	{root, one, one},
	{root, minus, minus},
	{root, one, minus},
	{minus, root, one},
	{one, root, one},
	{minus, root, minus},
	{one, root, minus},
	{one, one, root},
	{minus, one, root},
	{one, minus, root},
	{minus, minus, root},
	{one, zero, root},
	{minus, root, zero},
	{one, root, zero},
	{minus, zero, root},
	{zero, one, root},
	{root, minus, zero},
	{root, one, zero},
	{zero, minus, root},
	{zero, root, one},
	{root, zero, one},
	{root, zero, minus},
	{zero, root, minus},
}

// peres24 holds the 24 rays of Peres in four dimensions, numbered as in table 1
// of Waegell and Aravind, arXiv:1103.6058.
var peres24 = []ray{
	{one, zero, zero, zero},
	{zero, one, zero, zero},
	{zero, zero, one, zero},
	{zero, zero, zero, one},
	{one, one, one, one},
	{one, one, minus, minus},
	{one, minus, one, minus},
	{one, minus, minus, one},
	{one, minus, minus, minus},
	{one, minus, one, one},
	{one, one, minus, one},
	{one, one, one, minus},
	{one, one, zero, zero},
	{one, minus, zero, zero},
	{zero, zero, one, one},
	{zero, zero, one, minus},
	{zero, one, zero, one},
	{zero, one, zero, minus},
	{one, zero, one, zero},
	{one, zero, minus, zero},
	{one, zero, zero, minus},
	{one, zero, zero, one},
	{zero, one, minus, zero},
	{zero, one, one, zero},
}

// cabello18 holds the 18 rays of Cabello, Estebaranz and García-Alcaine,
// arXiv:quant-ph/9706009, numbered in the order equations (1) to (9) of that
// paper first use them.
var cabello18 = []ray{
	{zero, zero, zero, one},
	{zero, zero, one, zero},
	{one, one, zero, zero},
	{one, minus, zero, zero},
	{zero, one, zero, zero},
	{one, zero, one, zero},
	{one, zero, minus, zero},
	{one, minus, one, minus},
	{one, minus, minus, one},
	{zero, zero, one, one},
	{one, one, one, one},
	{zero, one, zero, minus},
	{one, zero, zero, one},
	{one, zero, zero, minus},
	{zero, one, minus, zero},
	{one, one, minus, one},
	{one, one, one, minus},
	{minus, one, one, one},
}
