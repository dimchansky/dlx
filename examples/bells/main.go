// Command bells searches for extents of change ringing on 3 or 4 bells, and
// prints them to standard output. Diagnostics go to standard error.
//
// A row is one order of the bells. An extent starts from rounds, the row
// 1234, visits every row exactly once and comes back to rounds. From one row to
// the next, a bell may move by at most one place.
//
// Usage:
//
//	bells [-bells 3|4] [-changes letters] [-limit n] [-count]
//
// The -bells flag sets the number of bells, 4 by default. The -changes flag
// lists the allowed changes, in any order: a swaps places 1 and 2, b swaps
// places 2 and 3, c swaps places 3 and 4, and d does a and c at the same time.
// The default is every single swap: ab on 3 bells and abc on 4. An explicitly
// empty list is an error. The -limit flag
// sets how many extents to print, 1 by default. Zero prints all of them. The
// -count flag goes through every cycle cover and prints the totals instead, so
// -limit has no effect with it. An extent and its reverse count separately.
//
// The exit status is 0 if an extent exists or -help was given, 1 if there is no
// extent, and 2 for invalid usage or an I/O failure.
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
	exitNoExtent = 1
	exitFailed   = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("bells", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	bells := flags.Int("bells", 4, "Number of bells: 3 or 4")
	names := flags.String("changes", "", "Allowed changes, from abcd (default every single swap)")
	limit := flags.Uint64("limit", 1, "Limit on the number of extents to print (specify zero to print all extents)")
	count := flags.Bool("count", false, "Print the totals instead of extents (goes through every cycle cover)")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitFailed
	}
	allowed, err := parseArgs(flags, *bells, *names)
	if err != nil {
		report(stderr, "%v", err)
		flags.Usage()
		return exitFailed
	}

	rows := allRows(*bells)
	var found uint64
	if *count {
		found, err = printTotals(stdout, rows, allowed)
	} else {
		found, err = printExtents(stdout, rows, allowed, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailed
	}
	if found == 0 {
		report(stderr, "no extent found")
		return exitNoExtent
	}
	return exitOK
}

// parseArgs checks everything the flags can get wrong before any work starts.
// It returns the allowed changes in the order of allChanges, so the order of the
// letters and any repeats make no difference.
func parseArgs(flags *flag.FlagSet, bells int, names string) ([]change, error) {
	if flags.NArg() != 0 {
		return nil, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if bells != 3 && bells != 4 {
		return nil, fmt.Errorf("invalid number of bells %d: want 3 or 4", bells)
	}
	if names == "" && !isSet(flags, "changes") {
		names = singleSwaps(bells)
	}
	if names == "" || strings.Trim(names, "abcd") != "" {
		return nil, fmt.Errorf("invalid changes %q: want letters from abcd", names)
	}

	var allowed []change
	for _, c := range allChanges {
		if !strings.Contains(names, string(c.name)) {
			continue
		}
		if !c.fits(bells) {
			return nil, fmt.Errorf("change %c needs 4 bells", c.name)
		}
		allowed = append(allowed, c)
	}
	return allowed, nil
}

// isSet reports whether the named flag was given on the command line.
func isSet(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(f *flag.Flag) { found = found || f.Name == name })
	return found
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "bells: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: bells [-bells 3|4] [-changes letters] [-limit n] [-count]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nchanges: a swaps places 1-2, b swaps 2-3, c swaps 3-4, d does a and c together\n")
	_, _ = fmt.Fprint(w, "exit status: 0 found, 1 no extent, 2 invalid usage or I/O failure\n")
}

// printExtents prints up to limit extents, or all of them if limit is zero, and
// returns how many it printed.
func printExtents(w io.Writer, rows []string, allowed []change, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(rows, allowed, func(next []int) bool {
		extent := extentOf(rows, next)
		if extent == nil {
			return false
		}
		found++
		writeErr = printExtent(w, found, extent)
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func printExtent(w io.Writer, number uint64, extent []string) error {
	if _, err := fmt.Fprintf(w, "Extent %d:\n", number); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, strings.Join(extent, "\n"))
	return err
}

// printTotals goes through every cycle cover, prints the totals and returns the
// number of extents.
func printTotals(w io.Writer, rows []string, allowed []change) (uint64, error) {
	var covers, extents uint64
	solve(rows, allowed, func(next []int) bool {
		covers++
		if extentOf(rows, next) != nil {
			extents++
		}
		return false
	})
	_, err := fmt.Fprintf(w, "Cycle covers: %d\nExtents: %d\n", covers, extents)
	return extents, err
}
