// Command queens places n queens on an n by n chessboard so that no two attack
// each other, and prints the solutions to standard output. Diagnostics go to
// standard error.
//
// Usage:
//
//	queens [-n size] [-limit n] [-count]
//
// The -n flag sets the board size, 8 by default. The -limit flag sets how many
// solutions to print, 1 by default. Zero prints all of them. The -count flag
// enumerates every solution and prints the total instead, so -limit has no
// effect with it. Rotations and reflections count as separate solutions.
//
// The exit status is 0 if at least one solution exists or -help was given, 1 if
// the board has no solutions, and 2 for invalid usage or an I/O failure.
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
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status. The problem comes from
// flags, so standard input is not read.
func run(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("queens", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	size := flags.Int("n", 8, "Board size and number of queens")
	limit := flags.Uint64("limit", 1, "Limit on the number of solutions to print (specify zero to print all solutions)")
	count := flags.Bool("count", false, "Print the total instead of solutions (enumerates every solution)")

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
		report(stderr, "invalid board size %d: want 1 to %d", *size, maxSize)
		return exitFailure
	}

	var found uint64
	var err error
	if *count {
		found, err = printTotal(stdout, *size)
	} else {
		found, err = printSolutions(stdout, *size, *limit)
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

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "queens: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: queens [-n size] [-limit n] [-count]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no solution, 2 invalid usage or I/O failure\n")
}

// printSolutions prints up to limit solutions, or all of them if limit is zero,
// and returns how many it printed.
func printSolutions(w io.Writer, n int, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(n, func(queens []int) bool {
		found++
		writeErr = printSolution(w, found, queens)
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func printSolution(w io.Writer, number uint64, queens []int) error {
	if _, err := fmt.Fprintf(w, "Solution %d:\n", number); err != nil {
		return err
	}
	for _, col := range queens {
		squares := strings.Repeat(". ", col) + "Q" + strings.Repeat(" .", len(queens)-col-1)
		if _, err := fmt.Fprintln(w, squares); err != nil {
			return err
		}
	}
	return nil
}

// printTotal enumerates every solution, prints the total and returns it.
func printTotal(w io.Writer, n int) (uint64, error) {
	var total uint64
	solve(n, func([]int) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Solutions: %d\n", total)
	return total, err
}
