// Command kochenspecker tries to colour a set of rays green and red under the
// rules of the Kochen-Specker theorem: exactly one green ray in every group of
// mutually perpendicular rays that fills all directions, and at most one green
// ray in every other perpendicular pair. An exact cover with no solutions is
// then a proof that no such colouring exists.
//
// Usage:
//
//	kochenspecker [-set name] [-remove ray] [-bases-only] [-limit n] [-rays] [-critical]
//
// The -set flag picks the rays: peres33 by default, peres24 or cabello18. The
// -remove flag leaves out one ray, counted from 1. The -bases-only flag drops
// the at-most-one rule for pairs. It prints the total only, and cannot be
// combined with -remove. The -limit flag sets how many colourings to print, 1 by
// default. Zero prints all of them. The -rays flag lists the rays and their
// groups instead of solving. The -critical flag removes each ray in turn and
// reports what remains. It cannot be combined with the other flags except -set.
//
// The exit status is 0 if at least one colouring exists or -help or -rays was
// given, 1 if no colouring exists, and 2 for invalid usage or an I/O failure.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	exitOK         = 0
	exitNoSolution = 1
	exitFailure    = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("kochenspecker", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	setName := flags.String("set", "peres33", "Set of rays: "+setNames())
	remove := flags.Int("remove", 0, "Ray to leave out, counted from 1 (specify zero to keep them all)")
	basesOnly := flags.Bool("bases-only", false, "Drop the rule for pairs, and print the total only")
	limit := flags.Uint64("limit", 1, "Limit on the number of colourings to print (specify zero to print all colourings)")
	list := flags.Bool("rays", false, "List the rays and their groups instead of solving")
	critical := flags.Bool("critical", false, "Remove each ray in turn and report what remains")

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
	set, ok := findSet(*setName)
	if !ok {
		report(stderr, "unknown set %q: want %s", *setName, setNames())
		return exitFailure
	}
	if *remove < 0 || *remove > len(set.rays) {
		report(stderr, "invalid ray %d: want 1 to %d, or 0 for none", *remove, len(set.rays))
		return exitFailure
	}
	if *critical && (*remove != 0 || *basesOnly || *list) {
		report(stderr, "-critical cannot be combined with -remove, -bases-only or -rays")
		return exitFailure
	}
	if *basesOnly && *remove != 0 {
		report(stderr, "-bases-only cannot be combined with -remove")
		return exitFailure
	}

	p := newProblem(set.rays, set.dimension, *remove-1)
	if *basesOnly {
		p.pairs = nil
	}

	solved := true
	var err error
	switch {
	case *list:
		err = printGroups(stdout, set, p)
	case *critical:
		solved, err = printCritical(stdout, set)
	default:
		solved, err = printColourings(stdout, set, p, *limit, !*basesOnly)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if !solved {
		report(stderr, "no solution found")
		return exitNoSolution
	}
	return exitOK
}

func setNames() string {
	names := make([]string, len(raySets))
	for i, set := range raySets {
		names[i] = set.name
	}
	return strings.Join(names, ", ")
}

func findSet(name string) (raySet, bool) {
	for _, set := range raySets {
		if set.name == name {
			return set, true
		}
	}
	return raySet{}, false
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "kochenspecker: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: kochenspecker [-set name] [-remove ray] [-bases-only] [-limit n] [-rays] [-critical]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no solution, 2 invalid usage or I/O failure\n")
}

// printColourings prints a summary and up to limit colourings, or all of them
// if limit is zero, unless show is false. It reports whether any colouring
// exists. The total always takes the whole search, because the point is to prove
// what is not there.
func printColourings(w io.Writer, set raySet, p problem, limit uint64, show bool) (bool, error) {
	if err := printSummary(w, set, p); err != nil {
		return false, err
	}

	var total uint64
	var writeErr error
	p.solve(func(green []int) bool {
		total++
		if !show || (limit != 0 && total > limit) {
			return false
		}
		_, writeErr = fmt.Fprintf(w, "Colouring %d: green rays %s\n", total, rayNumbers(green))
		return writeErr != nil
	})
	if writeErr != nil {
		return false, writeErr
	}
	_, err := fmt.Fprintf(w, "Colourings: %d\n", total)
	return total > 0, err
}

func printSummary(w io.Writer, set raySet, p problem) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Set %s: %d rays in %d dimensions\n", set.name, len(set.rays), set.dimension)
	if p.removed != noRay {
		fmt.Fprintf(&text, "Removed: ray %d %s\n", p.removed+1, set.rays[p.removed])
	}
	fmt.Fprintf(&text, "Bases: %d\nPairs: %d\nMatrix: %d columns, %d rows\n",
		len(p.bases), len(p.pairs), p.columns(), p.matrixRows())
	_, err := io.WriteString(w, text.String())
	return err
}

// printCritical removes each ray in turn and prints how many colourings remain.
// It reports whether any removal leaves a colouring.
func printCritical(w io.Writer, set raySet) (bool, error) {
	colourable := 0
	for r, ray := range set.rays {
		var total uint64
		newProblem(set.rays, set.dimension, r).solve(func([]int) bool {
			total++
			return false
		})
		if total > 0 {
			colourable++
		}
		if _, err := fmt.Fprintf(w, "Without ray %d %s: %d colourings\n", r+1, ray, total); err != nil {
			return false, err
		}
	}
	_, err := fmt.Fprintf(w, "Colourable after removing one ray: %d of %d\n", colourable, len(set.rays))
	return colourable > 0, err
}

// printGroups lists the rays and groups of a problem.
func printGroups(w io.Writer, set raySet, p problem) error {
	var text strings.Builder
	text.WriteString("Rays:\n")
	for r, ray := range set.rays {
		if r != p.removed {
			fmt.Fprintf(&text, "%d %s\n", r+1, ray)
		}
	}
	text.WriteString("Bases:\n")
	for _, basis := range p.bases {
		text.WriteString(rayNumbers(basis) + "\n")
	}
	text.WriteString("Pairs:\n")
	for _, pair := range p.pairs {
		text.WriteString(rayNumbers(pair) + "\n")
	}
	_, err := io.WriteString(w, text.String())
	return err
}

// rayNumbers formats rays the way all output counts them, from 1.
func rayNumbers(rays []int) string {
	parts := make([]string, len(rays))
	for i, r := range rays {
		parts[i] = strconv.Itoa(r + 1)
	}
	return strings.Join(parts, " ")
}
