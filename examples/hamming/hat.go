package main

// The hat game uses base 2: digit 0 is a white hat and digit 1 is a black hat.
// Hats are a word of the code's length, and prisoners are numbered from 0, left
// to right along the word.

// outcome adds up a strategy's results over every possible pattern of hats.
type outcome struct {
	wins, losses int // patterns on which the team wins or loses
	right, wrong int // individual guesses, across all patterns
}

// withHat returns hats with the given prisoner's hat set to colour.
func withHat(hats, length, prisoner, colour int) int {
	bit := 1 << (length - 1 - prisoner)
	return hats&^bit | colour*bit
}

// hatOf returns the colour of the given prisoner's hat.
func hatOf(hats, length, prisoner int) int {
	return hats >> (length - 1 - prisoner) & 1
}

// guess is the strategy of one prisoner. It looks at the other hats only: the
// prisoner's own hat is replaced by each colour in turn. If one colour would make
// a code word, the prisoner says the other colour. Otherwise the prisoner passes.
// Two code words never differ in just one hat, so the two colours can't both make
// a code word.
func guess(isCodeWord []bool, hats, length, prisoner int) (colour int, speaks bool) {
	switch {
	case isCodeWord[withHat(hats, length, prisoner, 0)]:
		return 1, true
	case isCodeWord[withHat(hats, length, prisoner, 1)]:
		return 0, true
	default:
		return 0, false
	}
}

// membership turns a code into a table indexed by word.
func membership(code []int, words int) []bool {
	isCodeWord := make([]bool, words)
	for _, word := range code {
		isCodeWord[word] = true
	}
	return isCodeWord
}

// teamWins is the rule of the game: at least one prisoner guesses and every guess
// is right.
func teamWins(right, wrong int) bool {
	return right > 0 && wrong == 0
}

// playAll plays the strategy against every pattern of hats.
func playAll(code []int, length int) outcome {
	patterns := 1 << length
	isCodeWord := membership(code, patterns)
	var total outcome
	for hats := 0; hats < patterns; hats++ {
		right, wrong := 0, 0
		for prisoner := 0; prisoner < length; prisoner++ {
			colour, speaks := guess(isCodeWord, hats, length, prisoner)
			switch {
			case !speaks:
			case colour == hatOf(hats, length, prisoner):
				right++
			default:
				wrong++
			}
		}
		total.right += right
		total.wrong += wrong
		if teamWins(right, wrong) {
			total.wins++
		} else {
			total.losses++
		}
	}
	return total
}
