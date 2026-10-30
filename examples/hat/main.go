// Command hat tries to cover a disk with copies of the hat, the aperiodic
// monotile found in 2023, and prints what it finds to standard output.
// Diagnostics go to standard error.
//
// Usage:
//
//	hat [-r radius] [-rotations] [-count] [-svg]
//
// The -r flag sets the radius of the disk, 6 by default. The -rotations flag
// allows only rotated copies of the hat, without its mirror image. The -count
// flag enumerates every cover and prints the total. The -svg flag prints the
// first cover as an SVG picture instead of a summary.
//
// The exit status is 0 if a cover exists or -help was given, 1 if there is
// none, and 2 for invalid usage or an I/O failure.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	exitOK         = 0
	exitNoSolution = 1
	exitFailure    = 2
)

const (
	// minRadius is the smallest radius with a kite inside the disk.
	minRadius = 2
	// maxRadius bounds the memory the matrix can take. The search itself can
	// become impractical far below it.
	maxRadius = 64
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hat", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	radius := flags.Int("r", 6, "Radius of the disk, in short kite sides")
	rotationsOnly := flags.Bool("rotations", false, "Allow only rotated copies of the hat, not its mirror image")
	count := flags.Bool("count", false, "Print the total number of covers (enumerates every cover)")
	svg := flags.Bool("svg", false, "Print the first cover as an SVG picture")

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
	if *radius < minRadius || *radius > maxRadius {
		report(stderr, "invalid radius %d: want %d to %d", *radius, minRadius, maxRadius)
		return exitFailure
	}
	if *count && *svg {
		report(stderr, "-count and -svg cannot be used together")
		return exitFailure
	}

	p := newPuzzle(*radius, !*rotationsOnly)
	var found bool
	var err error
	if *count {
		found, err = printTotal(stdout, p)
	} else {
		found, err = printCover(stdout, p, *radius, *svg)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if !found {
		report(stderr, "no cover found")
		return exitNoSolution
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "hat: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: hat [-r radius] [-rotations] [-count] [-svg]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 cover found, 1 no cover, 2 invalid usage or I/O failure\n")
}

// printCover finds the first cover and prints it, as a summary or a picture. It
// reports whether there was one.
func printCover(w io.Writer, p *puzzle, radius int, svg bool) (bool, error) {
	var first []placement
	p.solve(func(hats []placement) bool {
		first = hats
		return true
	})
	found := first != nil
	if svg {
		if !found {
			return false, nil
		}
		return true, writeSVG(w, first, radius)
	}

	_, err := fmt.Fprintf(w, "Disk of radius %d: %d kites, %d hat placements\n", radius, p.disk, len(p.rows))
	if err == nil && found {
		_, err = fmt.Fprintf(w, "Cover: %d hats, %d of them reflected\n", len(first), countReflected(first))
	}
	return found, err
}

func countReflected(hats []placement) int {
	reflected := 0
	for _, hat := range hats {
		if hat.mirrored {
			reflected++
		}
	}
	return reflected
}

// printTotal enumerates every cover, prints the total and reports whether there
// was at least one.
func printTotal(w io.Writer, p *puzzle) (bool, error) {
	var total uint64
	p.solve(func([]placement) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Covers: %d\n", total)
	return total != 0, err
}
