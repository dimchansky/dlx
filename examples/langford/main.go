// Command langford arranges two copies of each number from 1 to n in a row, so
// that the copies of a number k have exactly k numbers between them, and prints
// the arrangements to standard output. Diagnostics go to standard error.
//
// Usage:
//
//	langford [-n size] [-limit n] [-count] [-mirrors]
//
// The -n flag sets the largest number, 7 by default. The -limit flag sets how
// many arrangements to print, 1 by default. Zero prints all of them. The -count
// flag enumerates every arrangement and prints the total instead, so -limit has
// no effect with it. An arrangement and its reversal count as one, unless
// -mirrors is given.
//
// The exit status is 0 if at least one arrangement exists or -help was given, 1
// if there are none, and 2 for invalid usage or an I/O failure.
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
	exitOK         = 0
	exitNoSolution = 1
	exitFailure    = 2
)

// maxSize bounds the memory the matrix can take. The search itself can become
// impractical well below it.
const maxSize = 64

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("langford", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	size := flags.Int("n", 7, "Largest number in the arrangement")
	limit := flags.Uint64("limit", 1, "Limit on the number of arrangements to print (specify zero to print all)")
	count := flags.Bool("count", false, "Print the total instead of arrangements (enumerates every arrangement)")
	withMirrors := flags.Bool("mirrors", false, "Count an arrangement and its reversal as two")

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
	if *size < 1 || *size > maxSize {
		report(stderr, "invalid size %d: want 1 to %d", *size, maxSize)
		return exitFailure
	}

	var found uint64
	var err error
	if *count {
		found, err = printTotal(stdout, *size, *withMirrors)
	} else {
		found, err = printArrangements(stdout, *size, *withMirrors, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		report(stderr, "no arrangement found")
		return exitNoSolution
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "langford: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: langford [-n size] [-limit n] [-count] [-mirrors]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no arrangement, 2 invalid usage or I/O failure\n")
}

// printArrangements prints up to limit arrangements, or all of them if limit is
// zero, and returns how many it printed.
func printArrangements(w io.Writer, n int, withMirrors bool, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(n, withMirrors, func(sequence []int) bool {
		found++
		_, writeErr = fmt.Fprintln(w, formatSequence(sequence))
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func formatSequence(sequence []int) string {
	return strings.Trim(fmt.Sprint(sequence), "[]")
}

// printTotal enumerates every arrangement, prints the total and returns it.
func printTotal(w io.Writer, n int, withMirrors bool) (uint64, error) {
	var total uint64
	solve(n, withMirrors, func([]int) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Arrangements: %d\n", total)
	return total, err
}
