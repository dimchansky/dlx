// Command pentomino tiles a rectangle of 60 cells with the 12 pentominoes, each
// used exactly once, and prints the tilings to standard output. Diagnostics go
// to standard error.
//
// Usage:
//
//	pentomino [-size RxC] [-limit n] [-count]
//
// The -size flag sets the board as rows and columns, 6x10 by default. The -limit
// flag sets how many tilings to print, 1 by default. Zero prints all of them.
// The -count flag enumerates every tiling and prints totals instead of tilings,
// so -limit has no effect with it. Mirror images count as separate tilings in
// the printed tilings and the "Tilings" total. The "Distinct" total counts them
// once.
//
// The exit status is 0 if at least one tiling exists or -help was given, 1 if
// the board has no tilings, and 2 for invalid usage or an I/O failure.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	exitOK       = 0
	exitNoTiling = 1
	exitFailure  = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status. The board comes from a
// flag, so standard input is not read.
func run(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pentomino", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	size := flags.String("size", "6x10", "Board as rows and columns, with 60 cells in all")
	limit := flags.Uint64("limit", 1, "Limit on the number of tilings to print (specify zero to print all tilings)")
	count := flags.Bool("count", false, "Print totals instead of tilings (enumerates every tiling)")

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

	b, err := parseBoard(*size)
	if err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	var found uint64
	if *count {
		found, err = printTotals(stdout, b)
	} else {
		found, err = printTilings(stdout, b, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		report(stderr, "no tiling found")
		return exitNoTiling
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "pentomino: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: pentomino [-size RxC] [-limit n] [-count]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no tiling, 2 invalid usage or I/O failure\n")
}

// printTilings prints up to limit tilings, or all of them if limit is zero, and
// returns how many it printed.
func printTilings(w io.Writer, b board, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(b, func(tiling string) bool {
		found++
		writeErr = printTiling(w, b, found, tiling)
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func printTiling(w io.Writer, b board, number uint64, tiling string) error {
	if _, err := fmt.Fprintf(w, "Tiling %d:\n", number); err != nil {
		return err
	}
	for row := 0; row < b.rows; row++ {
		if _, err := fmt.Fprintln(w, tiling[row*b.cols:(row+1)*b.cols]); err != nil {
			return err
		}
	}
	return nil
}

// printTotals enumerates every tiling, prints both totals and returns the first.
func printTotals(w io.Writer, b board) (uint64, error) {
	var total, distinct uint64
	solve(b, func(tiling string) bool {
		total++
		if isCanonical(b, tiling) {
			distinct++
		}
		return false
	})
	_, err := fmt.Fprintf(w, "Tilings: %d\nDistinct: %d\n", total, distinct)
	return total, err
}
