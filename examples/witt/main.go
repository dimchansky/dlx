// Command witt searches for Steiner systems: groups of k points, chosen from v
// points, so that every set of t points lies in exactly one group. The default
// is S(5,6,12), the small Witt design. Systems go to standard output and
// diagnostics to standard error.
//
// Usage:
//
//	witt [-t n] [-k n] [-v n] [-limit n] [-count]
//
// The -t, -k and -v flags set the parameters, 5, 6 and 12 by default. The -limit
// flag sets how many systems to print, 1 by default. Zero prints all of them. The
// -count flag enumerates every system and prints the total instead, so -limit
// has no effect with it. Two systems that differ only in the names of their
// points count as separate solutions.
//
// The exit status is 0 if at least one system exists or -help was given, 1 if
// there is none, and 2 for invalid usage, a matrix over the size limit, or an
// I/O failure.
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

const (
	// maxPoints keeps every binomial coefficient within 64 bits.
	maxPoints = 60

	// maxOnes bounds the 1s in the matrix, and with them its memory: each 1 takes
	// 48 bytes on a 64-bit machine, so this limit is just under 1 GB.
	maxOnes = 20_000_000
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("witt", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	t := flags.Int("t", 5, "Every set of t points lies in exactly one block")
	k := flags.Int("k", 6, "Points in each block")
	v := flags.Int("v", 12, "Points in all")
	limit := flags.Uint64("limit", 1, "Limit on the number of systems to print (specify zero to print all systems)")
	count := flags.Bool("count", false, "Print the total instead of systems (enumerates every system)")

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

	s := spec{t: *t, k: *k, v: *v}
	if err := validate(s); err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	var found uint64
	var err error
	if *count {
		found, err = printTotal(stdout, s)
	} else {
		found, err = printSystems(stdout, s, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		report(stderr, "no solution found")
		return exitNoSolution
	}
	return exitOK
}

// validate rejects parameters that make no sense, and matrices too big to build.
func validate(s spec) error {
	if s.t < 1 || s.t > s.k || s.k > s.v || s.v > maxPoints {
		return fmt.Errorf("invalid parameters t=%d k=%d v=%d: want 1 <= t <= k <= v <= %d", s.t, s.k, s.v, maxPoints)
	}
	rows, onesPerRow := binomial(s.v, s.k), binomial(s.k, s.t)
	if rows > maxOnes/onesPerRow {
		return fmt.Errorf("S(%d,%d,%d) needs %d rows with %d ones each: the limit is %d ones in all",
			s.t, s.k, s.v, rows, onesPerRow, maxOnes)
	}
	return nil
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "witt: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: witt [-t n] [-k n] [-v n] [-limit n] [-count]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 found, 1 none exists, 2 invalid usage, matrix too large or I/O failure\n")
}

// printSystems prints up to limit systems, or all of them if limit is zero, and
// returns how many it printed.
func printSystems(w io.Writer, s spec, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(s, func(blocks [][]int) bool {
		found++
		writeErr = printSystem(w, found, blocks)
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func printSystem(w io.Writer, number uint64, blocks [][]int) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Solution %d:\n", number)
	for _, block := range blocks {
		for i, point := range block {
			if i > 0 {
				text.WriteByte(' ')
			}
			text.WriteString(strconv.Itoa(point))
		}
		text.WriteByte('\n')
	}
	_, err := io.WriteString(w, text.String())
	return err
}

// printTotal enumerates every system, prints the total and returns it.
func printTotal(w io.Writer, s spec) (uint64, error) {
	total := countSolutions(s)
	_, err := fmt.Fprintf(w, "Solutions: %d\n", total)
	return total, err
}
