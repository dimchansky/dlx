package main

import (
	"fmt"
	"math/bits"
	"math/rand"
	"sort"
	"testing"

	"github.com/dimchansky/dlx"
)

func TestLargestDeck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		handSize int
		want     int
	}{
		{name: "two cards", handSize: 2, want: 3},
		{name: "three cards", handSize: 3, want: 8},
		{name: "four cards", handSize: 4, want: 27},
		{name: "five cards", handSize: 5, want: 124},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := largestDeck(tt.handSize)
			if got != tt.want {
				t.Fatalf("largestDeck(%d) = %d, want %d", tt.handSize, got, tt.want)
			}
			// The formula must agree with the counts it claims to summarise.
			if h, m := countHands(tt.handSize, got), countMessages(tt.handSize, got); h != m {
				t.Errorf("deck %d: %d hands, %d messages, want equal", got, h, m)
			}
			if h, m := countHands(tt.handSize, got+1), countMessages(tt.handSize, got+1); h <= m {
				t.Errorf("deck %d: %d hands, %d messages, want more hands", got+1, h, m)
			}
		})
	}
}

func TestCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		handSize     int
		deck         int
		wantHands    uint64
		wantMessages uint64
	}{
		{name: "two from three", handSize: 2, deck: 3, wantHands: 3, wantMessages: 3},
		{name: "three from eight", handSize: 3, deck: 8, wantHands: 56, wantMessages: 56},
		{name: "three from nine", handSize: 3, deck: 9, wantHands: 84, wantMessages: 72},
		{name: "four from five", handSize: 4, deck: 5, wantHands: 5, wantMessages: 60},
		{name: "four from twenty-seven", handSize: 4, deck: 27, wantHands: 17550, wantMessages: 17550},
		{name: "five from fifty-two", handSize: 5, deck: 52, wantHands: 2598960, wantMessages: 6497400},
		{name: "five from 124", handSize: 5, deck: 124, wantHands: 225150024, wantMessages: 225150024},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := countHands(tt.handSize, tt.deck); got != tt.wantHands {
				t.Errorf("countHands = %d, want %d", got, tt.wantHands)
			}
			if got := countMessages(tt.handSize, tt.deck); got != tt.wantMessages {
				t.Errorf("countMessages = %d, want %d", got, tt.wantMessages)
			}
		})
	}
}

// rowCounter stands in for a matrix and only counts the rows added to it.
type rowCounter struct {
	dlx.Matrix
	real, slack int
}

func (c *rowCounter) AddRow(columns ...int) {
	if len(columns) == 1 {
		c.slack++
	} else {
		c.real++
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		handSize     int
		deck         int
		wantHands    int
		wantMessages int
		wantRows     int
		wantSlack    int
	}{
		{name: "no spare message, no slack", handSize: 2, deck: 3, wantHands: 3, wantMessages: 3, wantRows: 6},
		{name: "spare messages get slack rows", handSize: 2, deck: 2, wantHands: 1, wantMessages: 2, wantRows: 2, wantSlack: 2},
		{name: "default", handSize: 3, deck: 8, wantHands: 56, wantMessages: 56, wantRows: 336},
		{name: "deck 7", handSize: 3, deck: 7, wantHands: 35, wantMessages: 42, wantRows: 210, wantSlack: 42},
		{name: "hand 4", handSize: 4, deck: 12, wantHands: 495, wantMessages: 1320, wantRows: 11880, wantSlack: 1320},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := encode(tt.handSize, tt.deck)
			var c rowCounter
			addRows(&c, e.hands, e.messages, tt.deck)
			if len(e.hands) != tt.wantHands || len(e.messages) != tt.wantMessages {
				t.Errorf("got %d hands, %d messages, want %d, %d",
					len(e.hands), len(e.messages), tt.wantHands, tt.wantMessages)
			}
			if c.real != tt.wantRows || c.slack != tt.wantSlack {
				t.Errorf("got %d rows and %d slack rows, want %d and %d", c.real, c.slack, tt.wantRows, tt.wantSlack)
			}
		})
	}
}

// TestEncode_columnNumbers pins the worked example in the README.
func TestEncode_columnNumbers(t *testing.T) {
	t.Parallel()

	e := encode(3, 8)
	const hand, message = 31, 43
	if got := fmt.Sprint(e.hands[hand]); got != "[2 5 7]" {
		t.Errorf("hand %d = %s, want [2 5 7]", hand, got)
	}
	if got := fmt.Sprint(e.messages[message]); got != "[7 2]" {
		t.Errorf("message %d = %s, want [7 2]", message, got)
	}
	got := e.decode([][]int{{hand, len(e.hands) + message}})[hand]
	if got.hidden != 5 || fmt.Sprint(got.shown) != "[7 2]" {
		t.Errorf("decoded %+v, want hidden 5, shown [7 2]", got)
	}
}

func TestMissingCard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		hand  []int
		shown []int
		want  int
	}{
		{name: "first", hand: []int{2, 5, 7}, shown: []int{7, 5}, want: 2},
		{name: "middle", hand: []int{2, 5, 7}, shown: []int{7, 2}, want: 5},
		{name: "last", hand: []int{2, 5, 7}, shown: []int{5, 2}, want: 7},
		{name: "one card shown", hand: []int{1, 2}, shown: []int{1}, want: 2},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := missingCard(tt.hand, tt.shown); got != tt.want {
				t.Errorf("missingCard = %d, want %d", got, tt.want)
			}
		})
	}
}

// checkStrategy tests a strategy against the rules of the trick alone, without
// the matrix: every hand appears once, the shown cards and the hidden card are
// exactly that hand, and no message is used twice.
func checkStrategy(t *testing.T, plays []play, handSize, deck int) {
	t.Helper()

	if uint64(len(plays)) != countHands(handSize, deck) {
		t.Fatalf("%d plays, want %d", len(plays), countHands(handSize, deck))
	}
	hands := make(map[string]bool, len(plays))
	messages := make(map[string]bool, len(plays))
	for _, p := range plays {
		cards := append([]int{p.hidden}, p.shown...)
		sort.Ints(cards)
		if len(cards) != handSize {
			t.Fatalf("play %v uses %d cards, want %d", p, len(cards), handSize)
		}
		for i, card := range cards {
			if card != p.hand[i] || card < 1 || card > deck {
				t.Fatalf("play %v does not use its own hand", p)
			}
		}
		if hand := fmt.Sprint(p.hand); hands[hand] {
			t.Fatalf("hand %s appears twice", hand)
		} else {
			hands[hand] = true
		}
		if message := fmt.Sprint(p.shown); messages[message] {
			t.Fatalf("message %s is used twice", message)
		} else {
			messages[message] = true
		}
	}
}

func TestFindStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		handSize int
		deck     int
		slow     bool
	}{
		{handSize: 2, deck: 2},
		{handSize: 2, deck: 3},
		{handSize: 3, deck: 3},
		{handSize: 3, deck: 4},
		{handSize: 3, deck: 5},
		{handSize: 3, deck: 6},
		{handSize: 3, deck: 7},
		{handSize: 3, deck: 8},
		{handSize: 4, deck: 4},
		{handSize: 4, deck: 8},
		{handSize: 4, deck: 12},
		{handSize: 4, deck: 16, slow: true},
		{handSize: 4, deck: maxDeckForFour, slow: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("hand %d deck %d", tt.handSize, tt.deck), func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow with -race")
			}

			plays := findStrategy(tt.handSize, tt.deck)
			checkStrategy(t, plays, tt.handSize, tt.deck)
			if got := perform(plays, tt.deck); got != len(plays) {
				t.Errorf("perform = %d, want %d", got, len(plays))
			}
		})
	}
}

// TestNoStrategy confirms the counting shortcut on the matrix itself, for the
// decks where the search can finish the proof.
func TestNoStrategy(t *testing.T) {
	t.Parallel()

	for _, deck := range []int{4, 5, 6} {
		deck := deck
		t.Run(fmt.Sprintf("hand 2 deck %d", deck), func(t *testing.T) {
			t.Parallel()

			if plays := findStrategy(2, deck); plays != nil {
				t.Errorf("found %v, want none", plays)
			}
			if got := countStrategies(2, deck); got != 0 {
				t.Errorf("countStrategies = %d, want 0", got)
			}
		})
	}
}

func TestPerform_noticesAReusedMessage(t *testing.T) {
	t.Parallel()

	plays := []play{
		{hand: []int{1, 2}, hidden: 2, shown: []int{1}},
		{hand: []int{1, 3}, hidden: 3, shown: []int{1}},
	}
	if got := perform(plays, 3); got != 1 {
		t.Errorf("perform = %d, want 1", got)
	}
}

// countByHand counts strategies with no matrix. It hands out messages one hand
// at a time, remembering how many ways finish from each set of used messages.
// Hands are sets of bits and messages are numbers in its own numbering, so it
// shares nothing with encode.
func countByHand(handSize, deck int) uint64 {
	var handMessages [][]int
	numbers := map[int]int{}
	for mask := 0; mask < 1<<deck; mask++ {
		if bits.OnesCount(uint(mask)) != handSize {
			continue
		}
		var own []int
		var pick func(used, key, picked int)
		pick = func(used, key, picked int) {
			if picked == handSize-1 {
				if _, seen := numbers[key]; !seen {
					numbers[key] = len(numbers)
				}
				own = append(own, numbers[key])
				return
			}
			for card := 0; card < deck; card++ {
				if bit := 1 << card; mask&bit != 0 && used&bit == 0 {
					pick(used|bit, key*deck+card, picked+1)
				}
			}
		}
		pick(0, 0, 0)
		handMessages = append(handMessages, own)
	}

	type state struct {
		hand int
		used [2]uint64
	}
	memo := map[state]uint64{}
	var ways func(s state) uint64
	ways = func(s state) uint64 {
		if s.hand == len(handMessages) {
			return 1
		}
		if total, ok := memo[s]; ok {
			return total
		}
		var total uint64
		for _, message := range handMessages[s.hand] {
			word, bit := message/64, uint64(1)<<(message%64)
			if s.used[word]&bit != 0 {
				continue
			}
			next := state{hand: s.hand + 1, used: s.used}
			next.used[word] |= bit
			total += ways(next)
		}
		memo[s] = total
		return total
	}
	return ways(state{})
}

func TestCountStrategies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		handSize int
		deck     int
		want     uint64
		noSearch bool // dlx takes several seconds, so only countByHand runs
	}{
		{handSize: 2, deck: 2, want: 2},
		{handSize: 2, deck: 3, want: 2},
		{handSize: 3, deck: 3, want: 6},
		{handSize: 3, deck: 4, want: 876},
		{handSize: 3, deck: 5, want: 7198848, noSearch: true},
		{handSize: 4, deck: 4, want: 24},
		{handSize: 4, deck: 5, want: 7146144, noSearch: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("hand %d deck %d", tt.handSize, tt.deck), func(t *testing.T) {
			t.Parallel()
			if tt.noSearch && testing.Short() {
				t.Skip("slow with -race")
			}

			if got := countByHand(tt.handSize, tt.deck); got != tt.want {
				t.Errorf("countByHand = %d, want %d", got, tt.want)
			}
			if tt.noSearch {
				return
			}
			if got := countStrategies(tt.handSize, tt.deck); got != tt.want {
				t.Errorf("countStrategies = %d, want %d", got, tt.want)
			}
		})
	}
}

// recipeShow is the recipe Kleber credits to Elwyn Berlekamp, which needs no
// search. Cards are numbered from 0. It hides the card at position (sum of the
// hand) mod k, then uses the order of the rest to send the hidden card's number
// once the shown cards are deleted from the deck.
func recipeShow(hand []int) (hidden int, shown []int) {
	k := len(hand)
	sum := 0
	for _, card := range hand {
		sum += card
	}
	position := sum % k
	hidden = hand[position]
	rest := make([]int, 0, k-1)
	rest = append(rest, hand[:position]...)
	rest = append(rest, hand[position+1:]...)
	return hidden, orderByRank(rest, (hidden-position)/k)
}

// recipeName is the magician's side of the recipe.
func recipeName(shown []int) int {
	k := len(shown) + 1
	sum := 0
	for _, card := range shown {
		sum += card
	}
	renumbered := (k-sum%k)%k + k*rankOfOrder(shown)
	sorted := append([]int(nil), shown...)
	sort.Ints(sorted)
	for _, card := range sorted {
		if renumbered >= card {
			renumbered++
		}
	}
	return renumbered
}

// orderByRank returns the cards, given in increasing order, in the rank-th of
// their orders, counting in dictionary order from 0.
func orderByRank(cards []int, rank int) []int {
	left := append([]int(nil), cards...)
	result := make([]int, 0, len(cards))
	for len(left) > 0 {
		orders := 1
		for i := 2; i < len(left); i++ {
			orders *= i
		}
		i := rank / orders
		rank %= orders
		result = append(result, left[i])
		left = append(left[:i], left[i+1:]...)
	}
	return result
}

// rankOfOrder is the inverse of orderByRank.
func rankOfOrder(cards []int) int {
	rank := 0
	for i, card := range cards {
		smallerLater := 0
		for _, later := range cards[i+1:] {
			if later < card {
				smallerLater++
			}
		}
		rank = rank*(len(cards)-i) + smallerLater
	}
	return rank
}

func TestRecipe_workedExample(t *testing.T) {
	t.Parallel()

	hidden, shown := recipeShow([]int{1, 4, 6})
	if hidden != 6 || fmt.Sprint(shown) != "[4 1]" {
		t.Errorf("recipeShow = %d, %v, want 6, [4 1]", hidden, shown)
	}
	if got := recipeName([]int{4, 1}); got != 6 {
		t.Errorf("recipeName = %d, want 6", got)
	}
}

// TestRecipe proves by construction that a strategy exists on the largest deck
// for each hand size, including deck 27, which the search can't reach. Smaller
// decks need no test of their own: a hand from a smaller deck is also a hand of
// the larger one, and its message uses only its own cards.
func TestRecipe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		handSize int
	}{
		{name: "hand 2", handSize: 2},
		{name: "hand 3", handSize: 3},
		{name: "hand 4", handSize: 4},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deck := largestDeck(tt.handSize)
			messages := make(map[string]bool)
			hand := make([]int, tt.handSize)
			var choose func(position, next int)
			choose = func(position, next int) {
				if position == tt.handSize {
					hidden, shown := recipeShow(hand)
					if got := recipeName(shown); got != hidden {
						t.Fatalf("hand %v: hid %d, named %d", hand, hidden, got)
					}
					messages[fmt.Sprint(shown)] = true
					return
				}
				for card := next; card < deck; card++ {
					hand[position] = card
					choose(position+1, card+1)
				}
			}
			choose(0, 0)
			if uint64(len(messages)) != countHands(tt.handSize, deck) {
				t.Errorf("%d different messages, want one per hand, %d", len(messages), countHands(tt.handSize, deck))
			}
		})
	}
}

// TestRecipe_deck124 samples the recipe at five cards, where the 225150024 hands
// are too many to walk through in a test.
func TestRecipe_deck124(t *testing.T) {
	t.Parallel()

	samples := 50000
	if testing.Short() {
		samples = 2000
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < samples; i++ {
		hand := rng.Perm(124)[:5]
		sort.Ints(hand)
		hidden, shown := recipeShow(hand)
		if got := recipeName(shown); got != hidden {
			t.Fatalf("hand %v: hid %d, named %d", hand, hidden, got)
		}
	}
}

// Cards of the classic trick are numbers from 0 to 51: suit*13 + rank, with
// ranks from 0 (ace) to 12 (king). Cards order by rank first, then by suit.
func cheneyLess(a, b int) bool {
	if a%13 != b%13 {
		return a%13 < b%13
	}
	return a < b
}

// cheneyOrders lists the six orders of small (0), middle (1) and large (2) in
// the convention the README gives, for the distances 1 to 6.
var cheneyOrders = [6][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}

// sortThree orders three cards with cheneyLess.
func sortThree(cards [3]int) [3]int {
	if cheneyLess(cards[1], cards[0]) {
		cards[0], cards[1] = cards[1], cards[0]
	}
	if cheneyLess(cards[2], cards[1]) {
		cards[1], cards[2] = cards[2], cards[1]
	}
	if cheneyLess(cards[1], cards[0]) {
		cards[0], cards[1] = cards[1], cards[0]
	}
	return cards
}

// cheneyShow follows the rule for five cards: two cards share a suit, the one a
// short way round the clock is hidden, the other goes first, and the order of
// the other three gives the distance.
func cheneyShow(hand [5]int) (hidden int, shown [4]int, ok bool) {
	for _, first := range hand {
		for _, other := range hand {
			distance := (other%13 - first%13 + 13) % 13
			if first == other || first/13 != other/13 || distance > 6 {
				continue
			}
			var rest [3]int
			n := 0
			for _, card := range hand {
				if card != first && card != other {
					rest[n] = card
					n++
				}
			}
			rest = sortThree(rest)
			order := cheneyOrders[distance-1]
			return other, [4]int{first, rest[order[0]], rest[order[1]], rest[order[2]]}, true
		}
	}
	return 0, shown, false
}

// cheneyName is the magician's side: the suit of the first card, its rank plus
// the distance, and the distance read from the order of the other three.
func cheneyName(shown [4]int) int {
	rest := sortThree([3]int{shown[1], shown[2], shown[3]})
	for i, order := range cheneyOrders {
		if rest[order[0]] == shown[1] && rest[order[1]] == shown[2] && rest[order[2]] == shown[3] {
			return shown[0]/13*13 + (shown[0]%13+i+1)%13
		}
	}
	return -1
}

func TestCheney_workedExample(t *testing.T) {
	t.Parallel()

	const (
		clubs, diamonds, hearts, spades = 0, 13, 26, 39
	)
	threeOfClubs, nineOfClubs := clubs+2, clubs+8
	fiveOfDiamonds, jackOfHearts, twoOfSpades := diamonds+4, hearts+10, spades+1

	hidden, shown, ok := cheneyShow([5]int{threeOfClubs, fiveOfDiamonds, nineOfClubs, jackOfHearts, twoOfSpades})
	if !ok || hidden != nineOfClubs {
		t.Errorf("hidden = %d, %t, want %d", hidden, ok, nineOfClubs)
	}
	if want := [4]int{threeOfClubs, jackOfHearts, fiveOfDiamonds, twoOfSpades}; shown != want {
		t.Errorf("shown = %v, want %v", shown, want)
	}
	if got := cheneyName(shown); got != nineOfClubs {
		t.Errorf("cheneyName = %d, want %d", got, nineOfClubs)
	}
}

func TestCheney(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("walks through every hand of a 52-card deck")
	}

	hands := uint64(0)
	for a := 0; a < 52; a++ {
		for b := a + 1; b < 52; b++ {
			for c := b + 1; c < 52; c++ {
				for d := c + 1; d < 52; d++ {
					for e := d + 1; e < 52; e++ {
						hand := [5]int{a, b, c, d, e}
						hidden, shown, ok := cheneyShow(hand)
						if !ok || cheneyName(shown) != hidden {
							t.Fatalf("hand %v: hid %d, showed %v", hand, hidden, shown)
						}
						hands++
					}
				}
			}
		}
	}
	if want := countHands(5, 52); hands != want {
		t.Errorf("played %d hands, want %d", hands, want)
	}
}
