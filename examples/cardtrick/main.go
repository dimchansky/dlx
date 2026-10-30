// Command cardtrick finds a strategy for the five-card trick of Fitch Cheney in
// its general form, using exact cover. A spectator draws a hand of cards from a
// deck. An assistant hides one card and shows the rest in a chosen order, and a
// magician who sees only those cards names the hidden one.
//
// Usage:
//
//	cardtrick [-hand size] [-deck size]
//	cardtrick [-hand size] [-deck size] -play cards
//	cardtrick [-hand size] [-deck size] -count
//
// The -hand flag sets the cards drawn, 3 by default, from 2 to 4. The -deck flag
// sets the cards in the deck, numbered from 1, 8 by default. With no other flag,
// the command prints a strategy line by line, then performs it on every hand.
// The -play flag performs it on one hand, such as 2,5,7. The -count flag
// enumerates every strategy and prints the total instead, for tricks of up to 10
// hands.
//
// The exit status is 0 if a strategy exists or -help was given, 1 if there are
// none, and 2 for invalid usage or an I/O failure.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	exitOK         = 0
	exitNoSolution = 1
	exitFailure    = 2
)

const (
	minHand = 2
	maxHand = 4

	// maxDeck keeps the counts well inside uint64. Every deck this large has
	// no strategy for the hand sizes allowed, and that is answered by counting.
	maxDeck = 1000

	// maxDeckForFour is the largest deck the search reliably finishes on for
	// hands of 4 cards, in about a second. Decks 26 and 27 have strategies, but
	// with this column order dlx does not find one in reasonable time.
	maxDeckForFour = 25

	// maxCountedHands bounds -count. Totals grow fast: 10 hands take seconds,
	// and the next size of a hand of 3, with 20 hands, does not finish.
	maxCountedHands = 10
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// options holds the parsed command line.
type options struct {
	hand  int
	deck  int
	play  string
	count bool
}

// run executes the command and returns its exit status.
func run(args []string, stdout, stderr io.Writer) int {
	opts, code, ok := parseOptions(args, stderr)
	if !ok {
		return code
	}

	hands, messages := countHands(opts.hand, opts.deck), countMessages(opts.hand, opts.deck)
	if opts.deck > largestDeck(opts.hand) {
		report(stderr, "no strategy: %d hands but only %d messages", hands, messages)
		return exitNoSolution
	}
	if opts.hand == maxHand && opts.deck > maxDeckForFour {
		report(stderr, "deck %d has a strategy, but the search does not finish: want at most %d for hand %d",
			opts.deck, maxDeckForFour, opts.hand)
		return exitFailure
	}
	if opts.count && hands > maxCountedHands {
		report(stderr, "-count takes at most %d hands, this trick has %d", maxCountedHands, hands)
		return exitFailure
	}

	var drawn []int
	if opts.play != "" {
		var err error
		if drawn, err = parseHand(opts.play, opts.hand, opts.deck); err != nil {
			report(stderr, "invalid -play: %v", err)
			return exitFailure
		}
	}

	w := bufio.NewWriter(stdout)
	switch {
	case opts.count:
		_, _ = fmt.Fprintf(w, "Strategies: %d\n", countStrategies(opts.hand, opts.deck))
	case drawn != nil:
		printPlay(w, findStrategy(opts.hand, opts.deck), drawn, opts.deck)
	default:
		printStrategy(w, findStrategy(opts.hand, opts.deck), opts.deck, messages)
	}
	// A bufio.Writer keeps its first error, so Flush reports any failed write.
	if err := w.Flush(); err != nil {
		report(stderr, "writing output: %v", err)
		return exitFailure
	}
	return exitOK
}

// parseOptions reads and checks the command line. If ok is false, the command
// should exit with code.
func parseOptions(args []string, stderr io.Writer) (opts options, code int, ok bool) {
	flags := flag.NewFlagSet("cardtrick", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(flags) }
	flags.IntVar(&opts.hand, "hand", 3, "Cards in a hand")
	flags.IntVar(&opts.deck, "deck", 8, "Cards in the deck, numbered from 1")
	flags.StringVar(&opts.play, "play", "", "Perform the trick on one hand, such as 2,5,7")
	flags.BoolVar(&opts.count, "count", false, "Print the number of strategies (enumerates every strategy)")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, exitOK, false
		}
		return opts, exitFailure, false
	}
	switch {
	case flags.NArg() != 0:
		report(stderr, "unexpected argument %q", flags.Arg(0))
		flags.Usage()
	case opts.hand < minHand || opts.hand > maxHand:
		report(stderr, "invalid hand %d: want %d to %d", opts.hand, minHand, maxHand)
	case opts.deck < opts.hand || opts.deck > maxDeck:
		report(stderr, "invalid deck %d: want %d to %d", opts.deck, opts.hand, maxDeck)
	case opts.count && opts.play != "":
		report(stderr, "-count and -play can't be combined")
	default:
		return opts, exitOK, true
	}
	return opts, exitFailure, false
}

// parseHand reads a comma-separated hand and returns its cards in increasing
// order.
func parseHand(text string, handSize, deck int) ([]int, error) {
	fields := strings.Split(text, ",")
	if len(fields) != handSize {
		return nil, fmt.Errorf("got %d cards, want %d", len(fields), handSize)
	}
	cards := make([]int, 0, handSize)
	for _, field := range fields {
		card, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return nil, fmt.Errorf("card %q is not a number", field)
		}
		if card < 1 || card > deck {
			return nil, fmt.Errorf("card %d is not in a deck of %d", card, deck)
		}
		cards = append(cards, card)
	}
	sort.Ints(cards)
	for i := 1; i < len(cards); i++ {
		if cards[i] == cards[i-1] {
			return nil, fmt.Errorf("card %d appears twice", cards[i])
		}
	}
	return cards, nil
}

// report writes a diagnostic. There is nowhere left to report a failure of the
// write itself, and the exit status still tells the caller what happened.
func report(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "cardtrick: "+format+"\n", args...)
}

func printUsage(flags *flag.FlagSet) {
	w := flags.Output()
	// As in report, a failure to write usage text cannot be reported.
	_, _ = fmt.Fprint(w, "usage: cardtrick [-hand size] [-deck size] [-play cards | -count]\n\n")
	flags.PrintDefaults()
	_, _ = fmt.Fprint(w, "\nexit status: 0 solved, 1 no strategy, 2 invalid usage or I/O failure\n")
}

// printStrategy prints each line of the strategy, then the score of a
// performance over every hand. Write errors are left to the caller's Flush.
func printStrategy(w *bufio.Writer, plays []play, deck int, messages uint64) {
	_, _ = fmt.Fprintf(w, "Hands: %d\nMessages: %d\n\n", len(plays), messages)
	for _, p := range plays {
		_, _ = fmt.Fprintf(w, "hand %s  hide %2d  show %s\n", formatCards(p.hand), p.hidden, formatCards(p.shown))
	}
	_, _ = fmt.Fprintf(w, "\nMagician named %d of %d hidden cards\n", perform(plays, deck), len(plays))
}

// printPlay prints one performance. The magician's line comes from the
// notebook, which has never seen the hand.
func printPlay(w *bufio.Writer, plays []play, drawn []int, deck int) {
	magician := newNotebook(plays, deck)
	for _, p := range plays {
		if !sameCards(p.hand, drawn) {
			continue
		}
		shown := strings.Trim(fmt.Sprint(p.shown), "[]")
		_, _ = fmt.Fprintf(w, "Spectator draws %s\n", strings.Trim(fmt.Sprint(p.hand), "[]"))
		_, _ = fmt.Fprintf(w, "Assistant hides %d and shows %s\n", p.hidden, shown)
		_, _ = fmt.Fprintf(w, "Magician sees %s and names %d\n", shown, magician.name(p.shown))
		return
	}
}

func sameCards(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func formatCards(cards []int) string {
	fields := make([]string, len(cards))
	for i, card := range cards {
		fields[i] = fmt.Sprintf("%2d", card)
	}
	return strings.Join(fields, " ")
}
