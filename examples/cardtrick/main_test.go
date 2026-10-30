package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name: "smallest trick with no spare message",
			args: []string{"-hand", "2", "-deck", "3"},
			wantStdout: "Hands: 3\nMessages: 3\n\n" +
				"hand  1  2  hide  2  show  1\n" +
				"hand  1  3  hide  1  show  3\n" +
				"hand  2  3  hide  3  show  2\n" +
				"\nMagician named 3 of 3 hidden cards\n",
		},
		{
			name: "spare messages stay unused",
			args: []string{"-hand", "2", "-deck", "2"},
			wantStdout: "Hands: 1\nMessages: 2\n\n" +
				"hand  1  2  hide  2  show  1\n" +
				"\nMagician named 1 of 1 hidden cards\n",
		},
		{
			name: "one hand",
			args: []string{"-play", "2,5,7"},
			wantStdout: "Spectator draws 2 5 7\n" +
				"Assistant hides 5 and shows 7 2\n" +
				"Magician sees 7 2 and names 5\n",
		},
		{
			name: "hand may be given in any order",
			args: []string{"-play", "7, 2,5"},
			wantStdout: "Spectator draws 2 5 7\n" +
				"Assistant hides 5 and shows 7 2\n" +
				"Magician sees 7 2 and names 5\n",
		},
		{
			name:       "count",
			args:       []string{"-deck", "4", "-count"},
			wantStdout: "Strategies: 876\n",
		},
		{
			name:       "too many hands by one",
			args:       []string{"-deck", "9"},
			wantCode:   1,
			wantStderr: "cardtrick: no strategy: 84 hands but only 72 messages\n",
		},
		{
			name:       "count answers an impossible trick by counting too",
			args:       []string{"-hand", "2", "-deck", "4", "-count"},
			wantCode:   1,
			wantStderr: "cardtrick: no strategy: 6 hands but only 4 messages\n",
		},
		{
			name:       "largest deck allowed is still answered at once",
			args:       []string{"-hand", "4", "-deck", "1000"},
			wantCode:   1,
			wantStderr: "cardtrick: no strategy: 41417124750 hands but only 997002000 messages\n",
		},
		{
			name:       "deck the search does not finish",
			args:       []string{"-hand", "4", "-deck", "27"},
			wantCode:   2,
			wantStderr: "cardtrick: deck 27 has a strategy, but the search does not finish: want at most 25 for hand 4\n",
		},
		{
			name:       "count too large",
			args:       []string{"-deck", "6", "-count"},
			wantCode:   2,
			wantStderr: "cardtrick: -count takes at most 10 hands, this trick has 20\n",
		},
		{
			name:       "hand too small",
			args:       []string{"-hand", "1"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid hand 1: want 2 to 4\n",
		},
		{
			name:       "hand too large",
			args:       []string{"-hand", "5"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid hand 5: want 2 to 4\n",
		},
		{
			name:       "deck smaller than hand",
			args:       []string{"-deck", "2"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid deck 2: want 3 to 1000\n",
		},
		{
			name:       "deck too large",
			args:       []string{"-deck", "1001"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid deck 1001: want 3 to 1000\n",
		},
		{
			name:       "count with play",
			args:       []string{"-count", "-play", "1,2,3"},
			wantCode:   2,
			wantStderr: "cardtrick: -count and -play can't be combined\n",
		},
		{
			name:       "play with too few cards",
			args:       []string{"-play", "1,2"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid -play: got 2 cards, want 3\n",
		},
		{
			name:       "play with a word",
			args:       []string{"-play", "1,two,3"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid -play: card \"two\" is not a number\n",
		},
		{
			name:       "play with a card outside the deck",
			args:       []string{"-play", "1,2,9"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid -play: card 9 is not in a deck of 8\n",
		},
		{
			name:       "play with a repeated card",
			args:       []string{"-play", "1,2,2"},
			wantCode:   2,
			wantStderr: "cardtrick: invalid -play: card 2 appears twice\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			if code := run(tt.args, &stdout, &stderr); code != tt.wantCode {
				t.Errorf("run(%q) = %d, want %d; stderr: %s", tt.args, code, tt.wantCode, stderr.String())
			}
			if got := stdout.String(); got != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", got, tt.wantStdout)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", got, tt.wantStderr)
			}
		})
	}
}

func TestRun_defaultPlaysEveryHand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != exitOK {
		t.Fatalf("run = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	output := stdout.String()
	if got := strings.Count(output, "hand "); got != 56 {
		t.Errorf("printed %d hands, want 56", got)
	}
	for _, want := range []string{"Hands: 56\nMessages: 56\n", "Magician named 56 of 56 hidden cards\n"} {
		if !strings.Contains(output, want) {
			t.Errorf("output lacks %q", want)
		}
	}
}

func TestRun_usage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantHead string
	}{
		{name: "help", args: []string{"-help"}, wantCode: 0, wantHead: "usage: cardtrick"},
		{name: "unknown flag", args: []string{"-nope"}, wantCode: 2, wantHead: "flag provided but not defined"},
		{name: "stray argument", args: []string{"stray"}, wantCode: 2, wantHead: "cardtrick: unexpected argument \"stray\""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			if code := run(tt.args, &stdout, &stderr); code != tt.wantCode {
				t.Errorf("run = %d, want %d", code, tt.wantCode)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want nothing", stdout.String())
			}
			if !strings.HasPrefix(stderr.String(), tt.wantHead) {
				t.Errorf("stderr = %q, want prefix %q", stderr.String(), tt.wantHead)
			}
			if !strings.Contains(stderr.String(), "exit status:") {
				t.Errorf("stderr = %q, want the usage text", stderr.String())
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestRun_writeFailure(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"-play", "1,2,3"}, {"-hand", "2", "-deck", "3", "-count"}} {
		var stderr bytes.Buffer
		if code := run(args, failingWriter{}, &stderr); code != exitFailure {
			t.Errorf("run(%q) = %d, want %d", args, code, exitFailure)
		}
		if want := "cardtrick: writing output: disk full\n"; stderr.String() != want {
			t.Errorf("stderr = %q, want %q", stderr.String(), want)
		}
	}
}
