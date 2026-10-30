// Command vuza finds the rhythmic canons of a given rhythm: the beats on which
// copies of the rhythm must enter so that together they strike every beat of a
// loop exactly once. By default it looks for Vuza canons, in which neither the
// rhythm nor the pattern of entries repeats. Results go to standard output and
// diagnostics to standard error.
//
// Usage:
//
//	vuza [-n beats] [-tile list] [-limit n] [-count] [-all] [-score]
//
// The -n flag sets the length of the loop, 72 by default. The -tile flag sets
// the rhythm, as a comma-separated list of beats counted from 0. The default is
// the 72-beat rhythm 0,8,16,18,26,34. The -limit flag sets how many results to
// print, 1 by default. Zero prints all of them. The -count flag enumerates every
// complement and prints the totals instead, so -limit and -score have no effect
// with it. The -all flag also accepts entry patterns that repeat. The -score
// flag prints each voice as a line of the loop under its result.
//
// The exit status is 0 if at least one result exists or -help was given, 1 if
// there are none, and 2 for invalid usage or an I/O failure.
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

// defaultTile is a six-note rhythm that has Vuza canons in a loop of 72 beats,
// the shortest loop where any exist.
const defaultTile = "0,8,16,18,26,34"

// maxLoop bounds the memory the matrix can take. The search itself can become
// impractical well below it.
const maxLoop = 1024

// options holds the flags that shape the output.
type options struct {
	limit        uint64
	allowRepeats bool
	score        bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("vuza", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	n := flags.Int("n", 72, "Number of beats in the loop")
	tileText := flags.String("tile", defaultTile, "Rhythm, as comma-separated beats from 0 to n-1")
	limit := flags.Uint64("limit", 1, "Limit on the number of results to print (specify zero to print all)")
	count := flags.Bool("count", false, "Print the totals instead of results (enumerates every complement)")
	all := flags.Bool("all", false, "Accept entry patterns that repeat")
	score := flags.Bool("score", false, "Print each voice as a line of the loop")

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
	if *n < 2 || *n > maxLoop {
		report(stderr, "invalid loop length %d: want 2 to %d", *n, maxLoop)
		return exitFailure
	}
	tile, err := parseTile(*tileText, *n)
	if err != nil {
		report(stderr, "invalid tile %q: %v", *tileText, err)
		return exitFailure
	}

	opts := options{limit: *limit, allowRepeats: *all, score: *score}
	var found uint64
	if *count {
		found, err = printTotals(stdout, *n, tile, *all)
	} else {
		found, err = printComplements(stdout, *n, tile, opts)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		if *all {
			report(stderr, "no complement found")
		} else {
			report(stderr, "no aperiodic complement found")
		}
		return exitNoSolution
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "vuza: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: vuza [-n beats] [-tile list] [-limit n] [-count] [-all] [-score]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 found, 1 none found, 2 invalid usage or I/O failure\n")
}

// printComplements prints up to opts.limit complements, or all of them if the
// limit is zero, and returns how many it printed.
func printComplements(w io.Writer, n int, tile []int, opts options) (uint64, error) {
	var found uint64
	var writeErr error
	solve(n, tile, func(entries []int) bool {
		if !opts.allowRepeats && period(n, entries) != 0 {
			return false
		}
		found++
		_, writeErr = fmt.Fprintf(w, "Complement %d: %s\n", found, formatBeats(entries))
		if writeErr == nil && opts.score {
			_, writeErr = io.WriteString(w, formatScore(n, tile, entries))
		}
		return writeErr != nil || (opts.limit != 0 && found >= opts.limit)
	})
	return found, writeErr
}

func formatBeats(beats []int) string {
	return strings.Trim(fmt.Sprint(beats), "[]")
}

// formatScore draws one line per voice, with an x on each beat the voice strikes.
func formatScore(n int, tile, entries []int) string {
	var score strings.Builder
	for _, entry := range entries {
		line := []byte(strings.Repeat(".", n))
		for _, beat := range tile {
			line[(beat+entry)%n] = 'x'
		}
		score.Write(line)
		score.WriteByte('\n')
	}
	return score.String()
}

// printTotals enumerates every complement and prints how many there are and how
// many of them don't repeat. It returns the total that decides the exit status:
// the complements if allowRepeats is set, otherwise the aperiodic ones.
func printTotals(w io.Writer, n int, tile []int, allowRepeats bool) (uint64, error) {
	var complements, aperiodic uint64
	solve(n, tile, func(entries []int) bool {
		complements++
		if period(n, entries) == 0 {
			aperiodic++
		}
		return false
	})
	_, err := fmt.Fprintf(w, "Complements: %d\nAperiodic: %d\n", complements, aperiodic)
	if allowRepeats {
		return complements, err
	}
	return aperiodic, err
}
