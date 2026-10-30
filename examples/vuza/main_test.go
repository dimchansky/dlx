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
		wantLines  int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "first Vuza canon by default",
			wantLines:  1,
			wantStdout: "Complement 1: 0 1 5 6 12 25 29 36 42 48 49 53\n",
		},
		{
			name:       "totals of the default rhythm",
			args:       []string{"-count"},
			wantLines:  2,
			wantStdout: "Complements: 1296\nAperiodic: 432\n",
		},
		{
			name:       "totals of the partner",
			args:       []string{"-count", "-tile", "0,1,5,6,12,25,29,36,42,48,49,53"},
			wantLines:  2,
			wantStdout: "Complements: 216\nAperiodic: 216\n",
		},
		{
			name:       "first complement of the partner",
			args:       []string{"-tile", "0,1,5,6,12,25,29,36,42,48,49,53"},
			wantLines:  1,
			wantStdout: "Complement 1: 0 2 16 18 34 56\n",
		},
		{
			name:       "the paper's rhythm in a longer cycle",
			args:       []string{"-n", "144", "-tile", "0,16,32,36,52,68"},
			wantLines:  1,
			wantStdout: "Complement 1: 0 1 2 3 11 12 13 14 24 25 26 51 59 72 73 74 84 85 86 96 97 98 99 107\n",
		},
		{
			name:      "limit stops the search",
			args:      []string{"-limit", "3"},
			wantLines: 3,
		},
		{
			name:      "zero limit prints all aperiodic complements",
			args:      []string{"-limit", "0"},
			wantLines: 432,
		},
		{
			name:      "all prints repeating complements too",
			args:      []string{"-all", "-limit", "0"},
			wantLines: 1296,
		},
		{
			name:       "score draws one line per voice",
			args:       []string{"-n", "6", "-tile", "0,1,2", "-all", "-score"},
			wantLines:  3,
			wantStdout: "Complement 1: 0 3\nxxx...\n...xxx\n",
		},
		{
			name:       "score wraps round the cycle",
			args:       []string{"-n", "6", "-tile", "0,1,2", "-all", "-score", "-limit", "0"},
			wantLines:  9,
			wantStdout: "Complement 1: 0 3\nxxx...\n...xxx\nComplement 2: 1 4\n.xxx..\nx...xx\nComplement 3: 2 5\n..xxx.\nxx...x\n",
		},
		{
			name:       "count ignores limit and score",
			args:       []string{"-count", "-limit", "1", "-score"},
			wantLines:  2,
			wantStdout: "Complements: 1296\nAperiodic: 432\n",
		},
		{
			name:       "complements exist but all repeat",
			args:       []string{"-n", "6", "-tile", "0,1,2"},
			wantCode:   1,
			wantStderr: "vuza: no aperiodic complement found\n",
		},
		{
			name:       "count when all complements repeat",
			args:       []string{"-n", "6", "-tile", "0,1,2", "-count"},
			wantCode:   1,
			wantLines:  2,
			wantStdout: "Complements: 3\nAperiodic: 0\n",
			wantStderr: "vuza: no aperiodic complement found\n",
		},
		{
			name:       "count with all accepts repeats",
			args:       []string{"-n", "6", "-tile", "0,1,2", "-count", "-all"},
			wantLines:  2,
			wantStdout: "Complements: 3\nAperiodic: 0\n",
		},
		{
			name:       "no complement at all",
			args:       []string{"-n", "6", "-tile", "0,1,3", "-all"},
			wantCode:   1,
			wantStderr: "vuza: no complement found\n",
		},
		{
			name:       "cycle too small",
			args:       []string{"-n", "1", "-tile", "0"},
			wantCode:   2,
			wantStderr: "vuza: invalid cycle 1: want 2 to 1024\n",
		},
		{
			name:       "cycle too large",
			args:       []string{"-n", "1025"},
			wantCode:   2,
			wantStderr: "vuza: invalid cycle 1025: want 2 to 1024\n",
		},
		{
			name:       "default rhythm does not fit the cycle",
			args:       []string{"-n", "12"},
			wantCode:   2,
			wantStderr: "vuza: invalid tile \"0,8,16,18,26,34\": beat 16 is outside 0 to 11\n",
		},
		{
			name:       "repeating tile",
			args:       []string{"-n", "4", "-tile", "0,2"},
			wantCode:   2,
			wantStderr: "vuza: invalid tile \"0,2\": it repeats every 2 beats\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: vuza",
		},
		{
			name:       "positional argument",
			args:       []string{"72"},
			wantCode:   2,
			wantStderr: "vuza: unexpected argument \"72\"\nusage: vuza",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: vuza [-n beats] [-tile list] [-limit n] [-count] [-all] [-score]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit status %d, want %d", code, tt.wantCode)
			}
			if got := strings.Count(stdout.String(), "\n"); got != tt.wantLines {
				t.Errorf("printed %d lines, want %d", got, tt.wantLines)
			}
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr:\n%s\nwant it to contain:\n%s", stderr.String(), tt.wantStderr)
			}
			if tt.wantStderr == "" && stderr.Len() != 0 {
				t.Errorf("unexpected stderr:\n%s", stderr.String())
			}
		})
	}
}

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		failOn int
	}{
		{name: "first line stops the search", args: []string{"-limit", "0"}, failOn: 1},
		{name: "later line stops the search", args: []string{"-limit", "0"}, failOn: 5},
		{name: "score stops the search", args: []string{"-limit", "0", "-score"}, failOn: 2},
		{name: "totals", args: []string{"-count"}, failOn: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			writer := &failingWriter{failOn: tt.failOn}
			code := run(tt.args, writer, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "vuza: writing output: write failure\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
			}
			if writer.writes != tt.failOn {
				t.Errorf("made %d writes, want the search to stop after %d", writer.writes, tt.failOn)
			}
		})
	}
}

// failingWriter accepts writes until write number failOn, which fails.
type failingWriter struct {
	failOn int
	writes int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failOn {
		return 0, errors.New("write failure")
	}
	return len(p), nil
}
