// Command crew reads the flights of a day and a list of candidate crew pairings
// with their costs from standard input. It finds every way to pick pairings so
// that each flight is flown by exactly one crew, and prints the cheapest rosters
// to standard output. Diagnostics go to standard error.
//
// The first line of the input that is not blank or a '#' comment is "flights"
// followed by the flight names. Each later line is a pairing: a name, a cost from
// 0 to 1000000, and the flights it covers. Input is read until end of file.
//
// Usage:
//
//	crew [-limit n] < pairings.txt
//
// The -limit flag sets how many rosters to print, cheapest first, 1 by default.
// Zero prints all of them. The search enumerates and keeps every roster whatever
// the limit, so it suits small inputs only.
//
// The exit status is 0 if at least one roster exists or -help was given, 1 if
// no roster covers every flight, and 2 for invalid usage, invalid input or an
// I/O failure.
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
	exitNoRoster = 1
	exitFailure  = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("crew", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	limit := flags.Uint64("limit", 1, "Limit on the number of rosters to print, cheapest first (specify zero to print all)")

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

	p, err := parse(stdin)
	if err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	rosters := solve(p)
	if len(rosters) == 0 {
		report(stderr, "no roster covers every flight")
		return exitNoRoster
	}
	if err := printRosters(stdout, p, rosters, *limit); err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "crew: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: crew [-limit n] < pairings.txt\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no roster, 2 invalid usage, invalid input or I/O failure\n")
}

// printRosters prints a summary and up to limit rosters, or all if limit is zero.
func printRosters(w io.Writer, p problem, rosters []roster, limit uint64) error {
	if _, err := fmt.Fprintf(w, "Rosters covering every flight: %d\n", len(rosters)); err != nil {
		return err
	}
	for i, r := range rosters {
		if limit != 0 && uint64(i) >= limit {
			break
		}
		if err := printRoster(w, p, i+1, r); err != nil {
			return err
		}
	}
	return nil
}

func printRoster(w io.Writer, p problem, number int, r roster) error {
	if _, err := fmt.Fprintf(w, "\nRoster %d, cost %d:\n", number, r.cost); err != nil {
		return err
	}
	for _, chosen := range r.pairings {
		names := make([]string, len(chosen.flights))
		for i, flight := range chosen.flights {
			names[i] = p.flights[flight]
		}
		_, err := fmt.Fprintf(w, "  %-4s %5d  %s\n", chosen.name, chosen.cost, strings.Join(names, " "))
		if err != nil {
			return err
		}
	}
	return nil
}
