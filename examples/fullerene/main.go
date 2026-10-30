// Command fullerene finds the double bonds of a carbon molecule, by default the
// football-shaped C60, and prints them to standard output. Diagnostics go to
// standard error.
//
// Usage:
//
//	fullerene [-molecule name] [-limit n] [-count] [-clar]
//
// The -molecule flag picks the molecule: c60 (the default), benzene, naphthalene
// or benzyl. The -limit flag sets how many Kekule structures to print, 1 by
// default. Zero prints all of them. The -count flag enumerates every Kekule
// structure and prints the total instead. The -clar flag adds hexagons to the
// search and prints a table of Clar sextets, followed by the covers with the most
// sextets. The -count and -clar flags cannot be combined, and -limit has no
// effect with either of them.
//
// The exit status is 0 if at least one structure exists or -help was given, 1 if
// there are none, and 2 for invalid usage or an I/O failure.
package main

import (
	"bytes"
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

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("fullerene", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	name := flags.String("molecule", "c60", "Molecule: "+strings.Join(moleculeNames, ", "))
	limit := flags.Uint64("limit", 1, "Limit on the number of structures to print (specify zero to print all)")
	count := flags.Bool("count", false, "Print the total instead of structures (enumerates every structure)")
	clar := flags.Bool("clar", false, "Print the table of Clar sextets (enumerates every cover)")

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
	if *count && *clar {
		report(stderr, "-count and -clar cannot be combined")
		return exitFailure
	}
	mol, err := newMolecule(*name)
	if err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	var found uint64
	switch {
	case *clar:
		found, err = printClar(stdout, mol)
	case *count:
		found, err = printTotal(stdout, mol)
	default:
		found, err = printStructures(stdout, mol, *limit)
	}
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}

	if found == 0 {
		report(stderr, "no Kekule structure found")
		return exitNoSolution
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "fullerene: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: fullerene [-molecule name] [-limit n] [-count] [-clar]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 found, 1 no structure, 2 invalid usage or I/O failure\n")
}

// printStructures prints up to limit Kekule structures, or all of them if limit
// is zero, and returns how many it printed.
func printStructures(w io.Writer, mol molecule, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(mol, false, func(c cover) bool {
		found++
		_, writeErr = fmt.Fprintln(w, formatBonds(c.doubleBonds))
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

// formatBonds writes the bonds as atom pairs, such as "0-1 2-3".
func formatBonds(bonds [][2]int) string {
	if len(bonds) == 0 {
		return "none"
	}
	pairs := make([]string, len(bonds))
	for i, bond := range bonds {
		pairs[i] = fmt.Sprintf("%d-%d", bond[0], bond[1])
	}
	return strings.Join(pairs, " ")
}

// printTotal enumerates every Kekule structure, prints the total and returns it.
func printTotal(w io.Writer, mol molecule) (uint64, error) {
	var total uint64
	solve(mol, false, func(cover) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Kekule structures: %d\n", total)
	return total, err
}

// printClar prints the Clar report and returns how many Kekule structures there
// are, which is the table's first line: a cover with no sextets is a Kekule
// structure, and if any cover exists then turning its sextets into double bonds
// gives one.
func printClar(w io.Writer, mol molecule) (uint64, error) {
	clar := analyseClar(mol)
	if len(clar.lines) == 0 {
		return 0, nil
	}
	return uint64(clar.lines[0].covers), writeClarReport(w, clar)
}

// writeClarReport prints the table of sextets and the covers with the most
// sextets.
func writeClarReport(w io.Writer, clar clarReport) error {
	// Output goes to a buffer first, so one check covers every write.
	var out bytes.Buffer
	var totals clarLine
	fmt.Fprintf(&out, "%7s %7s %6s %11s\n", "Sextets", "Covers", "Sets", "Cannot grow")
	for _, line := range clar.lines {
		fmt.Fprintf(&out, "%7d %7d %6d %11d\n", line.sextets, line.covers, line.sets, line.structures)
		totals.covers += line.covers
		totals.sets += line.sets
		totals.structures += line.structures
	}
	fmt.Fprintf(&out, "%7s %7d %6d %11d\n", "Total", totals.covers, totals.sets, totals.structures)

	for _, c := range clar.largest {
		fmt.Fprintf(&out, "sextets %s, double bonds %s\n",
			strings.Trim(fmt.Sprint(sextetList(c.sextets)), "[]"), formatBonds(c.doubleBonds))
	}
	_, err := out.WriteTo(w)
	return err
}
