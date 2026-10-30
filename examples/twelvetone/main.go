// Command twelvetone finds all-interval series: orderings of the n pitches on a
// circle in which the n-1 steps between neighbours make every interval from 1
// to n-1 exactly once. With n=12 these are the all-interval twelve-tone rows of
// music. The series go to standard output and diagnostics to standard error.
//
// Usage:
//
//	twelvetone [-n size] [-first pitch] [-limit n] [-count]
//
// The -n flag sets the number of pitches, 12 by default. The -first flag sets
// the pitch a series starts on, 0 by default. A value of -1 tries every pitch.
// The -limit flag sets how many series to print, 1 by default. Zero prints all
// of them. The -count flag enumerates every series and prints the total
// instead, so -limit has no effect with it.
//
// Series of 12 pitches are printed as note names, others as pitch numbers.
//
// The exit status is 0 if at least one series exists or -help was given, 1 if
// there are none, and 2 for invalid usage or an I/O failure.
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

const minSize = 2

// maxSize bounds the memory the matrix can take. The search itself can become
// impractical well below it.
const maxSize = 32

// noteNames are the twelve pitches, starting from C.
var noteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("twelvetone", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	size := flags.Int("n", 12, "Number of pitches")
	first := flags.Int("first", 0, "Pitch the series starts on (specify -1 to try every pitch)")
	limit := flags.Uint64("limit", 1, "Limit on the number of series to print (specify zero to print all)")
	count := flags.Bool("count", false, "Print the total instead of series (enumerates every series)")

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
	if *size < minSize || *size > maxSize {
		report(stderr, "invalid size %d: want %d to %d", *size, minSize, maxSize)
		return exitFailure
	}
	if *first < anyFirst || *first >= *size {
		report(stderr, "invalid first pitch %d: want %d to %d", *first, anyFirst, *size-1)
		return exitFailure
	}

	var found uint64
	var err error
	if *count {
		found, err = printTotal(stdout, *size, *first)
	} else {
		found, err = printSeries(stdout, *size, *first, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		report(stderr, "no series found")
		return exitNoSolution
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "twelvetone: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: twelvetone [-n size] [-first pitch] [-limit n] [-count]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no series, 2 invalid usage or I/O failure\n")
}

// printSeries prints up to limit series, or all of them if limit is zero, and
// returns how many it printed.
func printSeries(w io.Writer, n, first int, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(n, first, func(series []int) bool {
		found++
		_, writeErr = fmt.Fprintln(w, formatSeries(series))
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

// formatSeries writes 12 pitches as note names, because they are the notes of
// the octave, and any other size as pitch numbers.
func formatSeries(series []int) string {
	words := make([]string, len(series))
	for i, pitch := range series {
		if len(series) == len(noteNames) {
			words[i] = noteNames[pitch]
		} else {
			words[i] = strconv.Itoa(pitch)
		}
	}
	return strings.Join(words, " ")
}

// printTotal enumerates every series, prints the total and returns it.
func printTotal(w io.Writer, n, first int) (uint64, error) {
	var total uint64
	solve(n, first, func([]int) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Series: %d\n", total)
	return total, err
}
