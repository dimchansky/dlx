// Command instrselect picks machine instructions for an arithmetic expression,
// the way the instruction selection step of a compiler does. It covers the
// expression tree with instruction patterns, each tree node exactly once, and
// prints the cheapest result as a small program. Patterns that fit exactly the
// same nodes count as one tile, the cheapest, in totals and in lists. Results
// go to standard output and diagnostics to standard error.
//
// An expression is built from registers (a, i, x2), numbers and the operators
// add, mul, shl and load, for example load(add(a, shl(i, 3))).
//
// Usage:
//
//	instrselect [-expr expression] [-limit n]
//
// The -expr flag sets the expression, load(add(a, 8)) by default. The -limit
// flag sets how many tilings to print, cheapest first, 1 by default. Zero
// prints all of them. The search goes through every tiling whatever the limit,
// so large expressions can take very long.
//
// The exit status is 0 if at least one tiling exists or -help was given, 1 if
// the instructions can't cover the expression, and 2 for invalid usage, an
// invalid expression or an I/O failure.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	exitOK       = 0
	exitNoTiling = 1
	exitFailure  = 2
)

const defaultExpression = "load(add(a, 8))"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("instrselect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	expression := flags.String("expr", defaultExpression, "Expression to compile")
	limit := flags.Uint64("limit", 1, "Limit on the number of tilings to print, cheapest first (specify zero to print all)")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitFailure
	}
	if flags.NArg() != 0 {
		report(stderr, "unexpected argument %q", flags.Arg(0))
		flags.Usage()
		return exitFailure
	}

	t, err := parse(*expression)
	if err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	found := solve(t, instructions, *limit)
	if found.total == 0 {
		report(stderr, "no tiling covers every node of %s", t.root)
		return exitNoTiling
	}
	if err := printSolution(stdout, t, found); err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "instrselect: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: instrselect [-expr expression] [-limit n]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no tiling, 2 invalid usage, invalid expression or I/O failure\n")
}

func printSolution(w io.Writer, t tree, found solution) error {
	_, err := fmt.Fprintf(w, "Expression: %s\nNodes: %d, rows: %d, tilings: %d\n",
		t.root, len(t.nodes), found.rows, found.total)
	if err != nil {
		return err
	}
	for i := range found.tilings {
		if err := printTiling(w, t, i+1, &found.tilings[i]); err != nil {
			return err
		}
	}
	return nil
}

func printTiling(w io.Writer, t tree, number int, chosen *tiling) error {
	if _, err := fmt.Fprintf(w, "\nTiling %d, cost %d:\n", number, chosen.cost); err != nil {
		return err
	}
	for _, line := range chosen.code {
		labels := make([]string, len(line.tile.covered))
		for i, id := range line.tile.covered {
			labels[i] = t.nodes[id].label
		}
		_, err := fmt.Fprintf(w, "  %-26s ; %s, cost %d, covers %s\n",
			line.text, line.tile.pattern.name, line.tile.pattern.cost, strings.Join(labels, " "))
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "  result in %s\n", chosen.result)
	return err
}
