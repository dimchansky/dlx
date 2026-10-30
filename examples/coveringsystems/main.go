// Command coveringsystems splits the integers into remainder classes, such as
// the even numbers, the numbers that leave 1 when divided by 4, and the numbers
// that leave 3, so that every integer lands in exactly one class. It prints each
// such split, called an exact covering system, to standard output. Diagnostics
// go to standard error.
//
// Usage:
//
//	coveringsystems [-n number] [-limit n] [-count] [-distinct]
//
// Every modulus in a printed system divides the -n number, 8 by default. The
// -limit flag sets how many systems to print, 30 by default. Zero prints all of
// them. The -count flag enumerates every system and prints the total instead, so
// -limit has no effect with it. The -distinct flag asks for at least two classes
// with all their moduli different, and no such system exists.
//
// The exit status is 0 if at least one system exists or -help was given, 1 if
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

// maxNumber bounds the memory the matrix can take. The search itself can become
// impractical well below it.
const maxNumber = 4096

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("coveringsystems", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	number := flags.Int("n", 8, "Number that every modulus in a system must divide")
	limit := flags.Uint64("limit", 30, "Limit on the number of systems to print (specify zero to print all)")
	count := flags.Bool("count", false, "Print the total instead of systems (enumerates every system)")
	distinct := flags.Bool("distinct", false, "Ask for at least two classes with all moduli different")

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
	if *number < 1 || *number > maxNumber {
		report(stderr, "invalid number %d: want 1 to %d", *number, maxNumber)
		return exitFailure
	}

	var found uint64
	var err error
	if *count {
		found, err = printTotal(stdout, *number, *distinct)
	} else {
		found, err = printSystems(stdout, *number, *distinct, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		report(stderr, "no system found")
		return exitNoSolution
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "coveringsystems: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: coveringsystems [-n number] [-limit n] [-count] [-distinct]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no system, 2 invalid usage or I/O failure\n")
}

// printSystems prints up to limit systems, or all of them if limit is zero, and
// returns how many it printed.
func printSystems(w io.Writer, n int, distinct bool, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(n, distinct, func(system []class) bool {
		found++
		_, writeErr = fmt.Fprintln(w, formatSystem(system))
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func formatSystem(system []class) string {
	parts := make([]string, len(system))
	for i, c := range system {
		parts[i] = fmt.Sprintf("%d mod %d", c.residue, c.modulus)
	}
	return strings.Join(parts, ", ")
}

// printTotal enumerates every system, prints the total and returns it.
func printTotal(w io.Writer, n int, distinct bool) (uint64, error) {
	var total uint64
	solve(n, distinct, func([]class) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Systems: %d\n", total)
	return total, err
}
