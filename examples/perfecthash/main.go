// Command perfecthash builds a minimal perfect hash for a list of lowercase
// words: a table with exactly one slot per word, and a small sum that sends each
// word to its own slot. It prints the table to standard output. Diagnostics go
// to standard error.
//
// Usage:
//
//	perfecthash [-keys word,word,...]
//
// The -keys flag gives the words, separated by commas. Without it, the words are
// Go's 25 keywords.
//
// The exit status is 0 if a table was found or -help was given, 1 if none
// exists, and 2 for invalid usage or an I/O failure.
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

const (
	// maxKeys bounds the size of the matrix. The search itself can become
	// impractical well below it.
	maxKeys = 256
	// maxKeyLength is a sanity bound on a word, not a limit of the hash.
	maxKeyLength = 64
)

// goKeywords is the default word list. A test compares it with go/token.
var goKeywords = []string{
	"break", "case", "chan", "const", "continue", "default", "defer", "else",
	"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
	"map", "package", "range", "return", "select", "struct", "switch", "type",
	"var",
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("perfecthash", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	keyList := flags.String("keys", "", "Comma-separated lowercase words (empty means Go's 25 keywords)")

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

	keys := goKeywords
	if *keyList != "" {
		keys = strings.Split(*keyList, ",")
	}
	if err := validateKeys(keys); err != nil {
		report(stderr, "%v", err)
		return exitFailure
	}

	var table *hashTable
	matrix := solve(keys, func(found *hashTable) bool {
		table = found
		return true
	})
	if table == nil {
		report(stderr, "no hash table found")
		return exitNoSolution
	}
	if _, err := io.WriteString(stdout, formatTable(table, matrix)); err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}
	return exitOK
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "perfecthash: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: perfecthash [-keys word,word,...]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no table exists, 2 invalid usage or I/O failure\n")
}

// validateKeys reports the first word that the hash can't take.
func validateKeys(keys []string) error {
	if len(keys) > maxKeys {
		return fmt.Errorf("too many keys: %d, want at most %d", len(keys), maxKeys)
	}
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if key == "" {
			return errors.New("empty key")
		}
		if len(key) > maxKeyLength {
			return fmt.Errorf("key %q is too long: want at most %d letters", key, maxKeyLength)
		}
		if strings.Trim(key, "abcdefghijklmnopqrstuvwxyz") != "" {
			return fmt.Errorf("invalid key %q: want lowercase letters a to z", key)
		}
		if seen[key] {
			return fmt.Errorf("duplicate key %q", key)
		}
		seen[key] = true
	}
	return nil
}

// formatTable lays out the size of the matrix, the shifts and the slots.
func formatTable(table *hashTable, matrix matrixSize) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Keys: %d, first letters: %d\n", len(table.slots), len(table.shifts))
	rowWord := "rows"
	if matrix.rows == 1 {
		rowWord = "row"
	}
	fmt.Fprintf(&b, "Matrix: %d columns, %d %s\n\n", matrix.columns, matrix.rows, rowWord)

	b.WriteString("Letter  Shift\n")
	for letter := byte('a'); letter <= 'z'; letter++ {
		if shift, ok := table.shifts[letter]; ok {
			fmt.Fprintf(&b, "%c       %d\n", letter, shift)
		}
	}

	b.WriteString("\nSlot  Key\n")
	for slot, key := range table.slots {
		fmt.Fprintf(&b, "%-4d  %s\n", slot, key)
	}
	return b.String()
}
