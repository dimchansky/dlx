// Command sudoku solves the Sudoku board read from standard input and prints its
// solutions to standard output. Diagnostics go to standard error.
//
// The board is nine lines of nine whitespace-separated cells, each a digit from
// 1 to 9, or '_' or '0' for a blank.
//
// Usage:
//
//	sudoku [-limit n] < board.txt
//
// The -limit flag sets how many solutions to print, 1 by default. Zero prints
// all of them.
//
// The exit status is 0 if at least one solution was printed, 1 if the board has
// no solutions, and 2 for invalid usage, invalid input or an I/O failure.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dimchansky/dlx/examples/sudoku/parser"
	"github.com/dimchansky/dlx/examples/sudoku/solver"
)

const (
	exitSolved     = 0
	exitNoSolution = 1
	exitFailure    = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sudoku", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	limit := flags.Uint64("limit", 1, "Limit on the number of solutions (specify zero to find all solutions)")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitSolved
		}
		return exitFailure
	}
	if flags.NArg() != 0 {
		report(stderr, "unexpected argument %q", flags.Arg(0))
		flags.Usage()
		return exitFailure
	}

	board, err := parser.ParseReader(stdin)
	if err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	out := bufio.NewWriter(stdout)
	solutionsFound := uint64(0)
	var writeErr error
	solver.Solve(board, solver.SolutionAccepterFunc(func(solution string) bool {
		solutionsFound++
		writeErr = printSolution(out, solutionsFound, solution)
		return writeErr != nil || (*limit != 0 && solutionsFound >= *limit)
	}))
	if writeErr == nil {
		writeErr = out.Flush()
	}
	if writeErr != nil {
		report(stderr, "writing solution: %v", writeErr)
		return exitFailure
	}

	if solutionsFound == 0 {
		report(stderr, "no solution found")
		return exitNoSolution
	}
	return exitSolved
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "sudoku: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: sudoku [-limit n] < board.txt\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no solution, 2 invalid usage, invalid input or I/O failure\n")
}

func printSolution(w io.Writer, number uint64, solution string) error {
	if _, err := fmt.Fprintf(w, "Solution %d:\n", number); err != nil {
		return err
	}
	return printSudoku(w, solution)
}

// printSudoku writes a solved board as nine lines of nine digits.
func printSudoku(w io.Writer, s string) error {
	line := make([]byte, 0, 18)
	for r := 0; r < 9; r++ {
		line = line[:0]
		for _, digit := range []byte(s[r*9 : (r+1)*9]) {
			line = append(line, digit, ' ')
		}
		line[len(line)-1] = '\n'
		if _, err := w.Write(line); err != nil {
			return err
		}
	}
	return nil
}
