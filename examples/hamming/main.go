// Command hamming finds perfect codes and uses one to play the seven prisoners
// hat game. Results go to standard output and diagnostics to standard error.
//
// A word is a row of digits. A perfect code is a set of words such that every
// word of that length is either in the set or one digit away from exactly one
// word of the set.
//
// Usage:
//
//	hamming [-base 2|3] [-n length] [-limit n] [-count]
//	hamming [-n length] -hat
//	hamming [-n length] -play hats
//
// The -base flag sets the number of digit values, 2 by default. The -n flag sets
// the word length, 7 by default. The -limit flag sets how many codes to print, 1
// by default. Zero prints all of them. The -count flag enumerates every code and
// prints the total instead, so -limit has no effect with it.
//
// The -hat flag prints the first code, then how its hat game strategy does
// against every pattern of hats. The -play flag prints the first code, then one
// round played against the given hats, such as 0110100, where 0 is a white hat
// and 1 is a black hat. Both need base 2 and ignore -limit.
//
// The exit status is 0 if a code exists or -help was given, 1 if there is no
// code, and 2 for invalid usage or an I/O failure.
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

// maxWords stops the length at 14 digits in base 2. Length 15 is the next one
// that has perfect codes, and the search does not find one in useful time.
const maxWords = 1 << 14

// options holds the parsed flags. playing tells whether -play was given at all,
// because an empty value must be rejected rather than taken for no flag.
type options struct {
	base, length int
	limit        uint64
	count, hat   bool
	playing      bool
	hats         string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hamming", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	base := flags.Int("base", 2, "Number of digit values: 2 or 3")
	length := flags.Int("n", 7, "Word length")
	limit := flags.Uint64("limit", 1, "Limit on the number of codes to print (specify zero to print all codes)")
	count := flags.Bool("count", false, "Print the total instead of codes (enumerates every code)")
	hat := flags.Bool("hat", false, "Score the hat game strategy of the first code on every pattern of hats")
	play := flags.String("play", "", "Play one round of the hat game against these hats, such as 0110100")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitFailure
	}
	opts := options{base: *base, length: *length, limit: *limit, count: *count, hat: *hat, hats: *play}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "play" {
			opts.playing = true
		}
	})
	if err := validate(flags, opts); err != nil {
		report(stderr, "%v", err)
		flags.Usage()
		return exitFailure
	}

	found, err := execute(stdout, opts)
	if err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}
	if found == 0 {
		report(stderr, "no perfect code found")
		return exitNoSolution
	}
	return exitOK
}

// execute runs the selected mode and returns how many codes it found.
func execute(w io.Writer, opts options) (uint64, error) {
	switch {
	case opts.count:
		return printTotal(w, opts.base, opts.length)
	case opts.hat:
		return printWithGame(w, opts.length, func(code []int) error {
			return printScore(w, playAll(code, opts.length))
		})
	case opts.playing:
		return printWithGame(w, opts.length, func(code []int) error {
			return printRound(w, code, opts.length, opts.hats)
		})
	default:
		return printCodes(w, opts.base, opts.length, opts.limit)
	}
}

// validate checks everything the flags can get wrong before any work starts.
func validate(flags *flag.FlagSet, opts options) error {
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if opts.base != 2 && opts.base != 3 {
		return fmt.Errorf("invalid base %d: want 2 or 3", opts.base)
	}
	if maxLength := longestWord(opts.base); opts.length < 1 || opts.length > maxLength {
		return fmt.Errorf("invalid length %d: want 1 to %d for base %d", opts.length, maxLength, opts.base)
	}
	if !opts.hat && !opts.playing {
		return nil
	}
	if opts.count || (opts.hat && opts.playing) {
		return errors.New("use only one of -count, -hat and -play")
	}
	if opts.base != 2 {
		return errors.New("the hat game needs base 2")
	}
	if !opts.playing {
		return nil
	}
	if len(opts.hats) != opts.length || strings.Trim(opts.hats, "01") != "" {
		return fmt.Errorf("invalid hats %q: want %d digits, each 0 or 1", opts.hats, opts.length)
	}
	return nil
}

// longestWord returns the longest word length that keeps to maxWords.
func longestWord(base int) int {
	length := 0
	for wordCount(base, length+1) <= maxWords {
		length++
	}
	return length
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "hamming: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: hamming [-base 2|3] [-n length] [-limit n] [-count]\n")
	_, _ = fmt.Fprint(w, "       hamming [-n length] -hat\n")
	_, _ = fmt.Fprint(w, "       hamming [-n length] -play hats\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no code, 2 invalid usage or I/O failure\n")
}

// printCodes prints up to limit codes, or all of them if limit is zero, and
// returns how many it printed.
func printCodes(w io.Writer, base, length int, limit uint64) (uint64, error) {
	var found uint64
	var writeErr error
	solve(base, length, func(code []int) bool {
		found++
		writeErr = printCode(w, found, code, base, length)
		return writeErr != nil || (limit != 0 && found >= limit)
	})
	return found, writeErr
}

func printCode(w io.Writer, number uint64, code []int, base, length int) error {
	if _, err := fmt.Fprintf(w, "Code %d:\n", number); err != nil {
		return err
	}
	for _, word := range code {
		if _, err := fmt.Fprintln(w, formatWord(word, base, length)); err != nil {
			return err
		}
	}
	return nil
}

// formatWord writes a word number with its leading zeros.
func formatWord(word, base, length int) string {
	digits := strconv.FormatInt(int64(word), base)
	return strings.Repeat("0", length-len(digits)) + digits
}

// printTotal enumerates every code, prints the total and returns it.
func printTotal(w io.Writer, base, length int) (uint64, error) {
	var total uint64
	solve(base, length, func([]int) bool {
		total++
		return false
	})
	_, err := fmt.Fprintf(w, "Codes: %d\n", total)
	return total, err
}

// printWithGame prints the first binary code, then lets game print what the
// prisoners do with it.
func printWithGame(w io.Writer, length int, game func(code []int) error) (uint64, error) {
	var found uint64
	var writeErr error
	solve(2, length, func(code []int) bool {
		found = 1
		writeErr = printCode(w, found, code, 2, length)
		if writeErr == nil {
			writeErr = game(code)
		}
		return true
	})
	return found, writeErr
}

func printScore(w io.Writer, score outcome) error {
	_, err := fmt.Fprintf(w, "Hat patterns:  %d\nTeam wins:     %d\nTeam loses:    %d\nRight guesses: %d\nWrong guesses: %d\n",
		score.wins+score.losses, score.wins, score.losses, score.right, score.wrong)
	return err
}

// printRound shows what each prisoner sees and says. The hats were validated by
// validate, so parsing them cannot fail.
func printRound(w io.Writer, code []int, length int, play string) error {
	number, _ := strconv.ParseInt(play, 2, 0)
	hats := int(number)
	isCodeWord := membership(code, 1<<length)
	if _, err := fmt.Fprintf(w, "Hats: %s\n", play); err != nil {
		return err
	}

	right, wrong := 0, 0
	for prisoner := 0; prisoner < length; prisoner++ {
		seen := play[:prisoner] + "?" + play[prisoner+1:]
		colour, speaks := guess(isCodeWord, hats, length, prisoner)
		line := "passes"
		switch {
		case !speaks:
		case colour == hatOf(hats, length, prisoner):
			line = fmt.Sprintf("says %d: right", colour)
			right++
		default:
			line = fmt.Sprintf("says %d: wrong", colour)
			wrong++
		}
		if _, err := fmt.Fprintf(w, "Prisoner %d sees %s and %s\n", prisoner+1, seen, line); err != nil {
			return err
		}
	}

	verdict := "Team loses"
	if teamWins(right, wrong) {
		verdict = "Team wins"
	}
	_, err := fmt.Fprintln(w, verdict)
	return err
}
