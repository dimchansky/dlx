package main

import (
	"sort"

	"github.com/dimchansky/dlx"
)

// play is one line of a strategy: what the assistant does with one hand. The
// cards of the hand are in increasing order, and shown keeps the order the
// magician sees.
type play struct {
	hand   []int
	hidden int
	shown  []int
}

// largestDeck returns the largest deck that can have a strategy. A deck is too
// big exactly when it has more hands than messages. Dividing both counts by
// their common factors leaves deck-handSize+1 against handSize factorial.
func largestDeck(handSize int) int {
	factorial := 1
	for i := 2; i <= handSize; i++ {
		factorial *= i
	}
	return factorial + handSize - 1
}

// countHands returns how many different hands of handSize cards a deck has.
// Hands are unordered. Multiplying before dividing keeps each step whole.
func countHands(handSize, deck int) uint64 {
	total := uint64(1)
	for i := 1; i <= handSize; i++ {
		total = total * uint64(deck-handSize+i) / uint64(i)
	}
	return total
}

// countMessages returns how many ordered rows of handSize-1 cards a deck has.
func countMessages(handSize, deck int) uint64 {
	total := uint64(1)
	for i := 0; i < handSize-1; i++ {
		total *= uint64(deck - i)
	}
	return total
}

// combinations returns every increasing list of size cards from 1 to deck.
func combinations(size, deck int) [][]int {
	var result [][]int
	current := make([]int, 0, size)
	var extend func(next int)
	extend = func(next int) {
		if len(current) == size {
			result = append(result, append([]int(nil), current...))
			return
		}
		for card := next; card <= deck; card++ {
			current = append(current, card)
			extend(card + 1)
			current = current[:len(current)-1]
		}
	}
	extend(1)
	return result
}

// arrangements returns every ordered list of size different cards taken from
// cards. If cards is increasing, the lists come in dictionary order.
func arrangements(cards []int, size int) [][]int {
	var result [][]int
	current := make([]int, 0, size)
	used := make([]bool, len(cards))
	var extend func()
	extend = func() {
		if len(current) == size {
			result = append(result, append([]int(nil), current...))
			return
		}
		for i, card := range cards {
			if used[i] {
				continue
			}
			used[i] = true
			current = append(current, card)
			extend()
			current = current[:len(current)-1]
			used[i] = false
		}
	}
	extend()
	return result
}

// messageKey turns a message into a number, so that it can be a map key. Each
// card is a digit in base deck+1, so two different messages get different keys.
func messageKey(message []int, deck int) int {
	key := 0
	for _, card := range message {
		key = key*(deck+1) + card
	}
	return key
}

// encoding is the exact cover form of one trick, with what decode needs to read
// a solution.
type encoding struct {
	hands    [][]int
	messages [][]int
	matrix   dlx.Matrix
}

// encode builds the matrix. It has a column for each hand, then a column for
// each message.
func encode(handSize, deck int) encoding {
	hands := combinations(handSize, deck)
	wholeDeck := make([]int, deck)
	for i := range wholeDeck {
		wholeDeck[i] = i + 1
	}
	messages := arrangements(wholeDeck, handSize-1)
	m := dlx.NewMatrix(len(hands) + len(messages))
	addRows(m, hands, messages, deck)
	return encoding{hands: hands, messages: messages, matrix: m}
}

// addRows fills the matrix. Each row is one possible line of the strategy: it
// covers a hand and a message made of cards from that hand. So every hand gets
// exactly one message, and no message serves two hands, which is what lets the
// magician work backwards.
//
// A deck below the largest has more messages than hands, and some messages go
// unused. Their columns must still be covered, so each message also gets a
// slack row covering it alone. When the counts are equal every message is
// needed and slack rows would only slow the search down.
func addRows(m dlx.Matrix, hands, messages [][]int, deck int) {
	column := make(map[int]int, len(messages))
	for i, message := range messages {
		column[messageKey(message, deck)] = len(hands) + i
	}

	for h, hand := range hands {
		for _, message := range arrangements(hand, len(hand)-1) {
			m.AddRow(h, column[messageKey(message, deck)])
		}
	}
	if len(messages) > len(hands) {
		for i := range messages {
			m.AddRow(len(hands) + i)
		}
	}
}

// decode turns an exact cover into a strategy, one play per hand in the order
// of e.hands. It skips slack rows, the only rows with a single column, and
// relies on dlx keeping each row's columns in AddRow order: hand, then message.
func (e encoding) decode(exactCover [][]int) []play {
	plays := make([]play, len(e.hands))
	for _, row := range exactCover {
		if len(row) == 1 {
			continue
		}
		hand := e.hands[row[0]]
		shown := e.messages[row[1]-len(e.hands)]
		plays[row[0]] = play{hand: hand, hidden: missingCard(hand, shown), shown: shown}
	}
	return plays
}

// missingCard returns the card of hand that shown leaves out.
func missingCard(hand, shown []int) int {
	sorted := append([]int(nil), shown...)
	sort.Ints(sorted)
	for i, card := range sorted {
		if card != hand[i] {
			return hand[i]
		}
	}
	return hand[len(hand)-1]
}

// findStrategy returns the first strategy dlx finds, or nil if there is none.
func findStrategy(handSize, deck int) []play {
	e := encode(handSize, deck)
	var plays []play
	e.matrix.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		plays = e.decode(exactCover)
		return true
	}))
	return plays
}

// countStrategies goes through every strategy and returns the total.
func countStrategies(handSize, deck int) uint64 {
	var total uint64
	encode(handSize, deck).matrix.Solve(dlx.SolutionAccepterFunc(func([][]int) bool {
		total++
		return false
	}))
	return total
}

// notebook is all the magician knows: for each message, the card it stands for.
// It holds no hands, so naming a card is a lookup and nothing else.
type notebook struct {
	deck   int
	hidden map[int]int
}

// newNotebook writes down a strategy from the magician's side.
func newNotebook(plays []play, deck int) notebook {
	n := notebook{deck: deck, hidden: make(map[int]int, len(plays))}
	for _, p := range plays {
		n.hidden[messageKey(p.shown, deck)] = p.hidden
	}
	return n
}

// name returns the card the magician names on seeing message.
func (n notebook) name(message []int) int {
	return n.hidden[messageKey(message, n.deck)]
}

// perform runs the trick once for every hand and returns how many times the
// magician named the hidden card. Two hands sharing a message would share a
// notebook entry, so a strategy that reuses a message scores below len(plays).
func perform(plays []play, deck int) int {
	magician := newNotebook(plays, deck)
	correct := 0
	for _, p := range plays {
		if magician.name(p.shown) == p.hidden {
			correct++
		}
	}
	return correct
}
